/*
 *     Copyright 2025 The CNAI Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package backend

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	common "d7y.io/api/v2/pkg/apis/common/v2"
	dfdaemon "d7y.io/api/v2/pkg/apis/dfdaemon/v2"
	"github.com/avast/retry-go/v4"
	legacymodelspec "github.com/dragonflyoss/model-spec/specs-go/v1"
	modelspec "github.com/modelpack/model-spec/specs-go/v1"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	internalpb "github.com/modelpack/modctl/internal/pb"
	"github.com/modelpack/modctl/pkg/archiver"
	"github.com/modelpack/modctl/pkg/backend/remote"
	"github.com/modelpack/modctl/pkg/config"
)

// fetchByDragonfly fetches partial files via Dragonfly gRPC service based on pattern matching.
func (b *backend) fetchByDragonfly(ctx context.Context, target string, cfg *config.Fetch) error {
	logrus.Infof("fetch: starting dragonfly fetch operation for target %s", target)

	// Parse reference and initialize remote client.
	ref, err := ParseReference(target)
	if err != nil {
		return fmt.Errorf("failed to parse target: %w", err)
	}

	registry, repo, tag := ref.Domain(), ref.Repository(), ref.Tag()
	src, err := remote.New(repo, remote.WithPlainHTTP(cfg.PlainHTTP), remote.WithInsecure(cfg.Insecure), remote.WithProxy(cfg.Proxy))
	if err != nil {
		return fmt.Errorf("failed to create remote client: %w", err)
	}

	// Fetch and decode manifest.
	_, manifestReader, err := src.Manifests().FetchReference(ctx, tag)
	if err != nil {
		return fmt.Errorf("failed to fetch manifest: %w", err)
	}
	defer manifestReader.Close()

	var manifest ocispec.Manifest
	if err := json.NewDecoder(manifestReader).Decode(&manifest); err != nil {
		return fmt.Errorf("failed to decode manifest: %w", err)
	}

	logrus.Debugf("fetch: loaded manifest for target %s [manifest: %+v]", target, manifest)

	// Filter layers by patterns.
	layers := []ocispec.Descriptor{}
	for _, layer := range manifest.Layers {
		for _, pattern := range cfg.Patterns {
			if anno := layer.Annotations; anno != nil {
				path := anno[modelspec.AnnotationFilepath]
				if path == "" {
					path = anno[legacymodelspec.AnnotationFilepath]
				}
				matched, err := filepath.Match(pattern, path)
				if err != nil {
					return fmt.Errorf("failed to match pattern: %w", err)
				}

				if matched {
					layers = append(layers, layer)
				}
			}
		}
	}

	if len(layers) == 0 {
		return fmt.Errorf("no layers matched the patterns")
	}

	// Get authentication token.
	authToken, err := getAuthToken(ctx, src, registry, repo)
	if err != nil {
		return fmt.Errorf("failed to get auth token: %w", err)
	}

	// Connect to Dragonfly gRPC.
	conn, err := grpc.NewClient(cfg.DragonflyEndpoint, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("failed to dial gRPC server: %w", err)
	}
	defer conn.Close()

	// TODO: need refactor as currently use a global flag to control the progress bar render.
	if cfg.DisableProgress {
		internalpb.SetDisableProgress(true)
	}

	pb := internalpb.NewProgressBar(cfg.ProgressWriter)
	pb.Start()
	defer pb.Stop()

	// Process layers concurrently.
	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(cfg.Concurrency)

	logrus.Infof("fetch: processing matched layers via dragonfly [count: %d]", len(layers))
	for _, layer := range layers {
		g.Go(func() error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			logrus.Debugf("fetch: processing layer %s via dragonfly", layer.Digest)
			if err := fetchLayerByDragonfly(ctx, pb, dfdaemon.NewDfdaemonDownloadClient(conn), ref, manifest, layer, authToken, cfg); err != nil {
				return err
			}
			logrus.Debugf("fetch: successfully processed layer %s via dragonfly", layer.Digest)
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return err
	}

	logrus.Infof("fetch: successfully fetched layers via dragonfly [count: %d]", len(layers))
	return nil
}

// fetchLayerByDragonfly handles downloading and extracting a single layer via Dragonfly.
func fetchLayerByDragonfly(ctx context.Context, pb *internalpb.ProgressBar, client dfdaemon.DfdaemonDownloadClient, ref Referencer, manifest ocispec.Manifest, desc ocispec.Descriptor, authToken string, cfg *config.Fetch) error {
	err := retry.Do(func() error {
		logrus.Debugf("fetch: processing layer %s", desc.Digest)
		cfg.Hooks.BeforePullLayer(desc, manifest) // Call before hook
		err := downloadAndExtractFetchLayer(ctx, pb, client, ref, desc, authToken, cfg)
		cfg.Hooks.AfterPullLayer(desc, err) // Call after hook
		if err != nil {
			err = fmt.Errorf("pull: failed to download and extract layer %s: %w", desc.Digest, err)
			logrus.Error(err)
		}

		return err
	}, append(defaultRetryOpts, retry.Context(ctx))...)

	if err != nil {
		err = fmt.Errorf("fetch: failed to download and extract layer %s: %w", desc.Digest, err)
		logrus.Error(err)
	}

	return err
}

// downloadAndExtractFetchLayer downloads a layer via Dragonfly and extracts it if necessary.
func downloadAndExtractFetchLayer(ctx context.Context, pb *internalpb.ProgressBar, client dfdaemon.DfdaemonDownloadClient, ref Referencer, desc ocispec.Descriptor, authToken string, cfg *config.Fetch) error {
	// Resolve output path.
	outputAbs, err := filepath.Abs(cfg.Output)
	if err != nil {
		return fmt.Errorf("failed to resolve output dir: %w", err)
	}

	var annoFilepath string
	if desc.Annotations != nil {
		if desc.Annotations[modelspec.AnnotationFilepath] != "" {
			annoFilepath = desc.Annotations[modelspec.AnnotationFilepath]
		} else {
			annoFilepath = desc.Annotations[legacymodelspec.AnnotationFilepath]
		}
	}

	if annoFilepath == "" {
		return fmt.Errorf("missing annotation filepath")
	}

	outputPath := filepath.Join(outputAbs, annoFilepath)
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	isTar := strings.HasSuffix(desc.MediaType, mediaTypeTarSuffix)
	if isTar {
		outputPath += mediaTypeTarSuffix
	}

	// Download layer via Dragonfly.
	request := &dfdaemon.DownloadTaskRequest{
		Download: &common.Download{
			Url:      buildBlobURL(ref, cfg.PlainHTTP, desc.Digest.String()),
			Type:     common.TaskType_STANDARD,
			Priority: common.Priority_LEVEL6,
			RequestHeader: map[string]string{
				"Authorization": authToken,
			},
			OutputPath:    &outputPath,
			ForceHardLink: false,
		},
	}

	stream, err := client.DownloadTask(ctx, request)
	if err != nil {
		return fmt.Errorf("failed to download layer: %w", err)
	}

	// Process stream responses.
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		resp, err := stream.Recv()
		if err == io.EOF {
			break
		}

		if err != nil {
			pb.Abort(desc.Digest.String(), err)
			return fmt.Errorf("failed to receive response: %w", err)
		}

		switch taskResp := resp.Response.(type) {
		case *dfdaemon.DownloadTaskResponse_DownloadTaskStartedResponse:
			logrus.Debugf("fetch: dragonfly download started for layer %s", desc.Digest.String())
			pb.Add(internalpb.NormalizePrompt("Fetching blob"), desc.Digest.String(), desc.Size, nil)
		case *dfdaemon.DownloadTaskResponse_DownloadPieceFinishedResponse:
			logrus.Debugf("fetch: dragonfly download progress for layer %s [piece length: %d]", desc.Digest.String(), taskResp.DownloadPieceFinishedResponse.Piece.Length)
			if bar := pb.Get(desc.Digest.String()); bar != nil {
				bar.SetCurrent(bar.Current() + int64(taskResp.DownloadPieceFinishedResponse.Piece.Length))
			}
		}
	}

	// Extract tar if applicable.
	if isTar {
		return extractFetchTar(outputPath, outputAbs)
	}

	return nil
}

// extractFetchTar untars a file and removes it afterward.
func extractFetchTar(tarPath, extractDir string) error {
	file, err := os.Open(tarPath)
	if err != nil {
		return fmt.Errorf("failed to open tar: %w", err)
	}
	defer file.Close()

	if err := archiver.Untar(file, extractDir); err != nil {
		return fmt.Errorf("failed to untar: %w", err)
	}

	if err := os.Remove(tarPath); err != nil {
		return fmt.Errorf("failed to remove tar: %w", err)
	}
	return nil
}
