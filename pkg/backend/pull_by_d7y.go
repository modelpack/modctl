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
	modelspec "github.com/CloudNativeAI/model-spec/specs-go/v1"
	"github.com/avast/retry-go/v4"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"oras.land/oras-go/v2/registry/remote/auth"

	internalpb "github.com/CloudNativeAI/modctl/internal/pb"
	"github.com/CloudNativeAI/modctl/pkg/archiver"
	"github.com/CloudNativeAI/modctl/pkg/backend/remote"
	"github.com/CloudNativeAI/modctl/pkg/config"
)

const (
	mediaTypeTarSuffix = ".tar"
)

// pullByDragonfly pulls and hardlinks blobs from Dragonfly gRPC service for remote extraction.
func (b *backend) pullByDragonfly(ctx context.Context, target string, cfg *config.Pull) error {
	logrus.Infof("pull: starting dragonfly pull operation for target %s", target)
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

	logrus.Debugf("pull: loaded manifest for target %s [manifest: %+v]", target, manifest)

	// Get authentication token.
	authToken, err := getAuthToken(ctx, src, registry, repo)
	if err != nil {
		return fmt.Errorf("failed to get auth token: %w", err)
	}

	// Connect to Dragonfly gRPC.
	// TODO: configure the credentials or certs in future.
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

	logrus.Infof("pull: processing layers via dragonfly [count: %d]", len(manifest.Layers))
	for _, layer := range manifest.Layers {
		g.Go(func() error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			logrus.Debugf("pull: processing layer %s via dragonfly", layer.Digest)
			if err := processLayer(ctx, pb, dfdaemon.NewDfdaemonDownloadClient(conn), ref, manifest, layer, authToken, cfg); err != nil {
				return err
			}
			logrus.Debugf("pull: successfully processed layer %s via dragonfly", layer.Digest)
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return err
	}

	logrus.Infof("pull: successfully pulled artifact %s via dragonfly", target)
	return nil
}

// getAuthToken retrieves the authentication token for the registry.
func getAuthToken(ctx context.Context, src *remote.Repository, registry, repo string) (string, error) {
	client, ok := src.Client.(*auth.Client)
	if !ok {
		return "", fmt.Errorf("failed to client is not an auth client")
	}

	scheme, err := client.Cache.GetScheme(ctx, registry)
	if err != nil {
		return "", fmt.Errorf("failed to get scheme: %w", err)
	}

	repo = strings.TrimPrefix(repo, registry+"/")
	token, err := client.Cache.GetToken(ctx, registry, scheme, fmt.Sprintf("repository:%s:pull", repo))
	if err != nil {
		return "", fmt.Errorf("failed to get token: %w", err)
	}

	if token == "" {
		return "", fmt.Errorf("failed to empty token from cache")
	}

	return token, nil
}

// buildBlobURL constructs the URL for a blob.
func buildBlobURL(ref Referencer, plainHTTP bool, digest string) string {
	scheme := "https"
	if plainHTTP {
		scheme = "http"
	}
	repo := strings.TrimPrefix(ref.Repository(), ref.Domain()+"/")
	return fmt.Sprintf("%s://%s/v2/%s/blobs/%s", scheme, ref.Domain(), repo, digest)
}

// processLayer handles downloading and extracting a single layer.
func processLayer(ctx context.Context, pb *internalpb.ProgressBar, client dfdaemon.DfdaemonDownloadClient, ref Referencer, manifest ocispec.Manifest, desc ocispec.Descriptor, authToken string, cfg *config.Pull) error {
	cfg.Hooks.BeforePullLayer(desc, manifest) // Call before hook
	err := retry.Do(func() error {
		return downloadAndExtractLayer(ctx, pb, client, ref, desc, authToken, cfg)
	}, append(defaultRetryOpts, retry.Context(ctx))...)
	cfg.Hooks.AfterPullLayer(desc, err) // Call after hook
	return err
}

// downloadAndExtractLayer downloads a layer and extracts it if necessary.
func downloadAndExtractLayer(ctx context.Context, pb *internalpb.ProgressBar, client dfdaemon.DfdaemonDownloadClient, ref Referencer, desc ocispec.Descriptor, authToken string, cfg *config.Pull) error {
	// Resolve output path.
	extractDirAbs, err := filepath.Abs(cfg.ExtractDir)
	if err != nil {
		return fmt.Errorf("failed to resolve extract dir: %w", err)
	}

	annoFilepath, ok := desc.Annotations[modelspec.AnnotationFilepath]
	if !ok || annoFilepath == "" {
		return fmt.Errorf("missing annotation filepath")
	}

	outputPath := filepath.Join(extractDirAbs, annoFilepath)
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	isTar := strings.HasSuffix(desc.MediaType, mediaTypeTarSuffix)
	if isTar {
		outputPath += mediaTypeTarSuffix
	}

	// Download layer.
	request := &dfdaemon.DownloadTaskRequest{
		Download: &common.Download{
			Url:      buildBlobURL(ref, cfg.PlainHTTP, desc.Digest.String()),
			Type:     common.TaskType_STANDARD,
			Priority: common.Priority_LEVEL6,
			RequestHeader: map[string]string{
				"Authorization": fmt.Sprintf("Bearer %s", authToken),
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
			logrus.Debugf("pull: dragonfly download started for layer %s", desc.Digest.String())
			pb.Add(internalpb.NormalizePrompt("Pulling blob"), desc.Digest.String(), desc.Size, nil)
		case *dfdaemon.DownloadTaskResponse_DownloadPieceFinishedResponse:
			logrus.Debugf("pull: dragonfly download progress for layer %s [piece length: %d]", desc.Digest.String(), taskResp.DownloadPieceFinishedResponse.Piece.Length)
			if bar := pb.Get(desc.Digest.String()); bar != nil {
				bar.SetCurrent(bar.Current() + int64(taskResp.DownloadPieceFinishedResponse.Piece.Length))
			}
		}
	}

	// Extract tar if applicable.
	if isTar {
		return extractTar(outputPath, extractDirAbs)
	}

	return nil
}

// extractTar untars a file and removes it afterward.
func extractTar(tarPath, extractDir string) error {
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
