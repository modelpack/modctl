/*
 *     Copyright 2024 The CNAI Authors
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
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/CloudNativeAI/modctl/pkg/codec"
	"github.com/CloudNativeAI/modctl/pkg/config"
	"github.com/CloudNativeAI/modctl/pkg/storage"
	modelspec "github.com/CloudNativeAI/model-spec/specs-go/v1"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

const (
	// defaultBufferSize is the default buffer size for reading the blob, default is 4MB.
	defaultBufferSize = 4 * 1024 * 1024
)

// Extract extracts the model artifact.
func (b *backend) Extract(ctx context.Context, target string, cfg *config.Extract) error {
	logrus.Infof("Extracting model artifact %s to %s", target, cfg.Output)

	// parse the repository and tag from the target.
	ref, err := ParseReference(target)
	if err != nil {
		logrus.Errorf("failed to parse the target: %v", err)
		return fmt.Errorf("failed to parse the target: %w", err)
	}

	repo, tag := ref.Repository(), ref.Tag()
	// pull the manifest from the storage.
	manifestRaw, _, err := b.store.PullManifest(ctx, repo, tag)
	if err != nil {
		logrus.Errorf("failed to pull the manifest from storage: %v", err)
		return fmt.Errorf("failed to pull the manifest from storage: %w", err)
	}
	// unmarshal the manifest.
	var manifest ocispec.Manifest
	if err := json.Unmarshal(manifestRaw, &manifest); err != nil {
		logrus.Errorf("failed to unmarshal the manifest: %v", err)
		return fmt.Errorf("failed to unmarshal the manifest: %w", err)
	}

	if err := exportModelArtifact(ctx, b.store, manifest, repo, cfg); err != nil {
		logrus.Errorf("failed to export the model artifact: %v", err)
		return fmt.Errorf("failed to export the model artifact: %w", err)
	}

	logrus.Infof("Extracted model artifact %s successfully", target)
	return nil
}

// exportModelArtifact exports the target model artifact to the output directory, which will open the artifact and extract to restore the original repo structure.
func exportModelArtifact(ctx context.Context, store storage.Storage, manifest ocispec.Manifest, repo string, cfg *config.Extract) error {
	g := &errgroup.Group{}
	g.SetLimit(cfg.Concurrency)

	for _, layer := range manifest.Layers {
		g.Go(func() error {
			// pull the blob from the storage.
			reader, err := store.PullBlob(ctx, repo, layer.Digest.String())
			if err != nil {
				logrus.Errorf("failed to pull the blob from storage: %v", err)
				return fmt.Errorf("failed to pull the blob from storage: %w", err)
			}
			defer reader.Close()

			bufferedReader := bufio.NewReaderSize(reader, defaultBufferSize)
			if err := extractLayer(layer, cfg.Output, bufferedReader); err != nil {
				logrus.Errorf("failed to extract layer %s: %v", layer.Digest.String(), err)
				return fmt.Errorf("failed to extract layer %s: %w", layer.Digest.String(), err)
			}

			return nil
		})
	}

	return g.Wait()
}

// extractLayer extracts the layer to the output directory.
func extractLayer(desc ocispec.Descriptor, outputDir string, reader io.Reader) error {
	var filepath string
	if desc.Annotations != nil && desc.Annotations[modelspec.AnnotationFilepath] != "" {
		filepath = desc.Annotations[modelspec.AnnotationFilepath]
	}

	codec, err := codec.New(codec.TypeFromMediaType(desc.MediaType))
	if err != nil {
		logrus.Errorf("failed to create codec for media type %s: %v", desc.MediaType, err)
		return fmt.Errorf("failed to create codec for media type %s: %w", desc.MediaType, err)
	}

	if err := codec.Decode(reader, outputDir, filepath); err != nil {
		logrus.Errorf("failed to decode the layer %s to output directory: %v", desc.Digest.String(), err)
		return fmt.Errorf("failed to decode the layer %s to output directory: %w", desc.Digest.String(), err)
	}

	return nil
}
