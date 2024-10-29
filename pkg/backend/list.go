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
	"sort"
	"time"

	modelspec "github.com/CloudNativeAI/modctl/pkg/spec"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

// ModelArtifact is the data model to represent the model artifact.
type ModelArtifact struct {
	// Repository is the repository of the model artifact.
	Repository string
	// Tag is the tag of the model artifact.
	Tag string
	// Digest is the digest of the model artifact.
	Digest string
	// Size is the size of the model artifact.
	Size int64
	// CreatedAt is the creation time of the model artifact.
	CreatedAt time.Time
}

// List lists all the model artifacts.
func (b *backend) List(ctx context.Context) ([]*ModelArtifact, error) {
	modelArtifacts := []*ModelArtifact{}

	// list all the repositories.
	repos, err := b.store.ListRepositories(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list repositories: %w", err)
	}

	// list all the tags in the repository.
	for _, repo := range repos {
		tags, err := b.store.ListTags(ctx, repo)
		if err != nil {
			return nil, fmt.Errorf("failed to list tags in repository %s: %w", repo, err)
		}

		// assemble the model artifact.
		for _, tag := range tags {
			modelArtifact, err := b.assembleModelArtifact(ctx, repo, tag)
			if err != nil {
				return nil, fmt.Errorf("failed to assemble model artifact: %w", err)
			}

			modelArtifacts = append(modelArtifacts, modelArtifact)
		}
	}

	sort.Slice(modelArtifacts, func(i, j int) bool {
		return modelArtifacts[i].CreatedAt.After(modelArtifacts[j].CreatedAt)
	})

	return modelArtifacts, nil
}

// assembleModelArtifact assembles the model artifact from the original storage.
func (b *backend) assembleModelArtifact(ctx context.Context, repo, tag string) (*ModelArtifact, error) {
	manifestRaw, digest, err := b.store.PullManifest(ctx, repo, tag)
	if err != nil {
		return nil, fmt.Errorf("failed to pull manifest: %w", err)
	}

	// parse the manifest.
	var manifest ocispec.Manifest
	if err := json.Unmarshal(manifestRaw, &manifest); err != nil {
		return nil, fmt.Errorf("failed to unmarshal manifest: %w", err)
	}

	// calculate the size of the model artifact.
	size := int64(len(manifestRaw)) + manifest.Config.Size
	for _, layer := range manifest.Layers {
		size += layer.Size
	}

	modelArtifact := &ModelArtifact{
		Repository: repo,
		Tag:        tag,
		Digest:     digest,
		Size:       size,
	}

	// parse the creation time from the manifest annotation if existed.
	if createdStr, ok := manifest.Annotations[modelspec.AnnotationCreated]; ok {
		created, err := time.Parse(time.RFC3339, createdStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse created time: %w", err)
		}

		modelArtifact.CreatedAt = created
	}

	return modelArtifact, nil
}
