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

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

// Prune prunes the unused blobs and clean up the storage.
func (b *backend) Prune(ctx context.Context) ([]string, error) {
	// list all repositories.
	repos, err := b.store.ListRepositories(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list repositories: %w", err)
	}

	prunedBlobs := make([]string, 0)
	for _, repo := range repos {
		pruned, err := b.pruneRepo(ctx, repo)
		if err != nil {
			return nil, fmt.Errorf("failed to prune repo %s: %w", repo, err)
		}

		for _, blob := range pruned {
			prunedBlobs = append(prunedBlobs, fmt.Sprintf("%s@%s", repo, blob))
		}
	}

	return prunedBlobs, nil
}

// pruneRepo prunes the unused blobs in the repository, and clean up the repository
// if no blobs left.
func (b *backend) pruneRepo(ctx context.Context, repo string) ([]string, error) {
	// get index.json from the repository.
	indexContent, err := b.store.GetIndex(ctx, repo)
	if err != nil {
		return nil, fmt.Errorf("failed to get index.json in repo %s: %w", repo, err)
	}

	var index ocispec.Index
	if err := json.Unmarshal(indexContent, &index); err != nil {
		return nil, fmt.Errorf("failed to unmarshal index.json in repo %s: %w", repo, err)
	}

	refedBlobs, err := b.refedBlobs(ctx, repo, index)
	if err != nil {
		return nil, fmt.Errorf("failed to get refed blobs in repo %s: %w", repo, err)
	}

	allBlobs, err := b.store.ListBlobs(ctx, repo)
	if err != nil {
		return nil, fmt.Errorf("failed to list blobs in repo %s: %w", repo, err)
	}

	pruneBlobs := make([]string, 0)
	for _, blob := range allBlobs {
		if _, ok := refedBlobs[blob]; !ok {
			pruneBlobs = append(pruneBlobs, blob)
		}
	}

	// cleanup the repo.
	removeRepo := false // len(pruneBlobs) > 0 && len(pruneBlobs) == len(allBlobs)
	_, err = b.store.CleanupRepo(ctx, repo, pruneBlobs, removeRepo)
	if err != nil {
		return nil, fmt.Errorf("failed to cleanup repo %s: %w", repo, err)
	}

	return pruneBlobs, nil
}

// refedBlobs returns the blobs that are referenced by the index.
func (b *backend) refedBlobs(ctx context.Context, repo string, index ocispec.Index) (map[string]bool, error) {
	refed := make(map[string]bool)
	for _, desc := range index.Manifests {
		manifestContent, _, err := b.store.PullManifest(ctx, repo, desc.Digest.String())
		if err != nil {
			return nil, fmt.Errorf("failed to get manifest %s in repo %s: %w", desc.Digest, repo, err)
		}

		var manifest ocispec.Manifest
		if err := json.Unmarshal(manifestContent, &manifest); err != nil {
			return nil, fmt.Errorf("failed to unmarshal manifest %s in repo %s: %w", desc.Digest, repo, err)
		}

		refed[desc.Digest.String()] = true
		refed[manifest.Config.Digest.String()] = true
		for _, layer := range manifest.Layers {
			refed[layer.Digest.String()] = true
		}
	}

	return refed, nil
}
