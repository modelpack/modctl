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

	"github.com/bmatcuk/doublestar/v4"
	legacymodelspec "github.com/dragonflyoss/model-spec/specs-go/v1"
	modelspec "github.com/modelpack/model-spec/specs-go/v1"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"

	internalpb "github.com/modelpack/modctl/internal/pb"
	"github.com/modelpack/modctl/pkg/backend/remote"
	"github.com/modelpack/modctl/pkg/config"
)

// Fetch fetches partial files to the output.
func (b *backend) Fetch(ctx context.Context, target string, cfg *config.Fetch) error {
	logrus.Infof("fetch: starting fetch operation for target %s [config: %+v]", target, cfg)

	// fetchByDragonfly is called if a Dragonfly endpoint is specified in the configuration.
	if cfg.DragonflyEndpoint != "" {
		logrus.Infof("fetch: using dragonfly for target %s", target)
		return b.fetchByDragonfly(ctx, target, cfg)
	}

	// parse the repository and tag from the target.
	ref, err := ParseReference(target)
	if err != nil {
		return fmt.Errorf("failed to parse the target: %w", err)
	}

	repo, tag := ref.Repository(), ref.Tag()
	client, err := remote.New(repo, remote.WithPlainHTTP(cfg.PlainHTTP), remote.WithInsecure(cfg.Insecure))
	if err != nil {
		return fmt.Errorf("failed to create remote client: %w", err)
	}

	_, manifestReader, err := client.Manifests().FetchReference(ctx, tag)
	if err != nil {
		return fmt.Errorf("failed to fetch the manifest: %w", err)
	}

	defer manifestReader.Close()

	var manifest ocispec.Manifest
	if err := json.NewDecoder(manifestReader).Decode(&manifest); err != nil {
		return fmt.Errorf("failed to decode the manifest: %w", err)
	}

	logrus.Debugf("fetch: loaded manifest for target %s [manifest: %+v]", target, manifest)

	layers := []ocispec.Descriptor{}
	// filter the layers by patterns.
	for _, layer := range manifest.Layers {
		for _, pattern := range cfg.Patterns {
			if anno := layer.Annotations; anno != nil {
				path := anno[modelspec.AnnotationFilepath]
				if path == "" {
					path = anno[legacymodelspec.AnnotationFilepath]
				}
				// Use doublestar.PathMatch for pattern matching to support ** recursive matching
				// PathMatch uses the system's native path separator (like filepath.Match) while
				// also supporting recursive patterns like **/*.json
				matched, err := doublestar.PathMatch(pattern, path)
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

	pb := internalpb.NewProgressBar()
	pb.Start()
	defer pb.Stop()

	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(cfg.Concurrency)

	logrus.Infof("fetch: processing matched layers [count: %d]", len(layers))
	for _, layer := range layers {
		g.Go(func() error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			logrus.Debugf("fetch: processing layer %s", layer.Digest)
			if err := pullAndExtractFromRemote(ctx, pb, internalpb.NormalizePrompt("Fetching blob"), client, cfg.Output, layer); err != nil {
				return err
			}

			logrus.Debugf("fetch: successfully processed layer %s", layer.Digest)
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return err
	}

	logrus.Infof("fetch: successfully fetched layers [count: %d]", len(layers))
	return nil
}
