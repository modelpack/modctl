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
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/CloudNativeAI/modctl/pkg/archiver"
	"github.com/CloudNativeAI/modctl/pkg/storage"
	modelspec "github.com/CloudNativeAI/model-spec/specs-go/v1"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

// Extract extracts the model artifact.
func (b *backend) Extract(ctx context.Context, target string, output string) error {
	// parse the repository and tag from the target.
	ref, err := ParseReference(target)
	if err != nil {
		return fmt.Errorf("failed to parse the target: %w", err)
	}

	repo, tag := ref.Repository(), ref.Tag()
	// pull the manifest from the storage.
	manifestRaw, _, err := b.store.PullManifest(ctx, repo, tag)
	if err != nil {
		return fmt.Errorf("failed to pull the manifest from storage: %w", err)
	}
	// unmarshal the manifest.
	var manifest ocispec.Manifest
	if err := json.Unmarshal(manifestRaw, &manifest); err != nil {
		return fmt.Errorf("failed to unmarshal the manifest: %w", err)
	}

	return exportModelArtifact(ctx, b.store, manifest, repo, output)
}

// exportModelArtifact exports the target model artifact to the output directory, which will open the artifact and extract to restore the original repo structure.
func exportModelArtifact(ctx context.Context, store storage.Storage, manifest ocispec.Manifest, repo, output string) error {
	for _, layer := range manifest.Layers {
		// pull the blob from the storage.
		reader, err := store.PullBlob(ctx, repo, layer.Digest.String())
		if err != nil {
			return fmt.Errorf("failed to pull the blob from storage: %w", err)
		}

		defer reader.Close()

		targetDir := output
		// get the original filepath in order to restore the original repo structure.
		originalFilePath := layer.Annotations[modelspec.AnnotationFilepath]
		if dir := filepath.Dir(originalFilePath); dir != "" {
			targetDir = filepath.Join(targetDir, dir)
		}

		// untar the blob to the target directory.
		if err := archiver.Untar(reader, targetDir); err != nil {
			return fmt.Errorf("failed to untar the blob to output directory: %w", err)
		}
	}

	return nil
}
