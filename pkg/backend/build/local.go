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

package build

import (
	"context"
	"fmt"
	"io"

	"github.com/CloudNativeAI/modctl/pkg/backend/build/hooks"
	"github.com/CloudNativeAI/modctl/pkg/storage"
	modelspec "github.com/CloudNativeAI/model-spec/specs-go/v1"

	godigest "github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

func NewLocalOutput(cfg *config, store storage.Storage, repo, tag string) (OutputStrategy, error) {
	return &localOutput{
		cfg:   cfg,
		store: store,
		repo:  repo,
		tag:   tag,
	}, nil
}

type localOutput struct {
	cfg   *config
	store storage.Storage
	repo  string
	tag   string
}

// OutputLayer outputs the layer blob to the local storage.
func (lo *localOutput) OutputLayer(ctx context.Context, mediaType, relPath, digest string, size int64, reader io.Reader, hooks hooks.Hooks) (ocispec.Descriptor, error) {
	reader = hooks.OnStart(relPath, size, reader)
	digest, size, err := lo.store.PushBlob(ctx, lo.repo, reader, ocispec.Descriptor{})
	if err != nil {
		hooks.OnError(relPath, err)
		return ocispec.Descriptor{}, fmt.Errorf("failed to push blob to storage: %w", err)
	}

	desc := ocispec.Descriptor{
		MediaType: mediaType,
		Digest:    godigest.Digest(digest),
		Size:      size,
		Annotations: map[string]string{
			modelspec.AnnotationFilepath: relPath,
		},
	}

	hooks.OnComplete(relPath, desc)
	return desc, nil
}

// OutputConfig outputs the config blob to the storage.
func (lo *localOutput) OutputConfig(ctx context.Context, mediaType, digest string, size int64, reader io.Reader, hooks hooks.Hooks) (ocispec.Descriptor, error) {
	reader = hooks.OnStart(digest, size, reader)
	digest, size, err := lo.store.PushBlob(ctx, lo.repo, reader, ocispec.Descriptor{})
	if err != nil {
		hooks.OnError(digest, err)
		return ocispec.Descriptor{}, fmt.Errorf("failed to push config to storage: %w", err)
	}

	desc := ocispec.Descriptor{
		MediaType: mediaType,
		Size:      size,
		Digest:    godigest.Digest(digest),
	}

	hooks.OnComplete(digest, desc)
	return desc, nil
}

// OutputManifest outputs the manifest blob to the local storage.
func (lo *localOutput) OutputManifest(ctx context.Context, mediaType, digest string, size int64, reader io.Reader, hooks hooks.Hooks) (ocispec.Descriptor, error) {
	reader = hooks.OnStart(digest, size, reader)
	manifestJSON, err := io.ReadAll(reader)
	if err != nil {
		hooks.OnError(digest, err)
		return ocispec.Descriptor{}, fmt.Errorf("failed to read manifest JSON: %w", err)
	}

	digest, err = lo.store.PushManifest(ctx, lo.repo, lo.tag, manifestJSON)
	if err != nil {
		hooks.OnError(digest, err)
		return ocispec.Descriptor{}, fmt.Errorf("failed to push manifest to storage: %w", err)
	}

	desc := ocispec.Descriptor{
		MediaType: mediaType,
		Digest:    godigest.Digest(digest),
		Size:      int64(len(manifestJSON)),
	}

	hooks.OnComplete(digest, desc)
	return desc, nil
}
