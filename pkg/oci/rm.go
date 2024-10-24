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

package oci

import (
	"context"
	"fmt"

	"github.com/CloudNativeAI/modctl/pkg/storage"
)

// Remove removes the target from the storage, notice that remove only removes the manifest,
// the blobs may still be used by other manifests, so should use prune to remove the unused blobs.
func Remove(ctx context.Context, target string) (string, error) {
	ref, err := ParseReference(target)
	if err != nil {
		return "", fmt.Errorf("failed to parse target: %w", err)
	}

	store, err := storage.New("")
	if err != nil {
		return "", fmt.Errorf("failed to create storage: %w", err)
	}

	repo, tag := ref.Repository(), ref.Tag()
	_, digest, err := store.PullManifest(ctx, repo, tag)
	if err != nil {
		return "", fmt.Errorf("failed to get manifest: %w", err)
	}

	// remove the manifest by digest.
	if tag != "" {
		if err := store.DeleteManifest(ctx, repo, digest); err != nil {
			return "", fmt.Errorf("failed to delete manifest %s: %w", digest, err)
		}
	}

	return digest, nil
}
