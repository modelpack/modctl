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
	"github.com/CloudNativeAI/modctl/pkg/backend/remote"

	modelspec "github.com/CloudNativeAI/model-spec/specs-go/v1"
	godigest "github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

func NewRemoteOutput(cfg *config, repo, tag string) (OutputStrategy, error) {
	remote, err := remote.New(repo, remote.WithPlainHTTP(cfg.plainHTTP), remote.WithInsecure(cfg.insecure))
	if err != nil {
		return nil, fmt.Errorf("failed to create remote repository: %w", err)
	}

	return &remoteOutput{
		cfg:    cfg,
		remote: remote,
		repo:   repo,
		tag:    tag,
	}, nil
}

type remoteOutput struct {
	cfg    *config
	remote *remote.Repository
	repo   string
	tag    string
}

// OutputLayer outputs the layer blob to the remote storage.
func (ro *remoteOutput) OutputLayer(ctx context.Context, mediaType, relPath, digest string, size int64, reader io.Reader, hooks hooks.Hooks) (ocispec.Descriptor, error) {
	desc := ocispec.Descriptor{
		MediaType: mediaType,
		Digest:    godigest.Digest(digest),
		Size:      size,
		Annotations: map[string]string{
			modelspec.AnnotationFilepath: relPath,
		},
	}

	reader = hooks.OnStart(relPath, size, reader)
	exist, err := ro.remote.Blobs().Exists(ctx, desc)
	if err != nil {
		hooks.OnError(relPath, err)
		return ocispec.Descriptor{}, fmt.Errorf("failed to check if blob exists: %w", err)
	}

	if exist {
		hooks.OnComplete(relPath, desc)
		return desc, nil
	}

	if err = ro.remote.Blobs().Push(ctx, desc, reader); err != nil {
		hooks.OnError(relPath, err)
		return ocispec.Descriptor{}, fmt.Errorf("failed to push layer to storage: %w", err)
	}

	hooks.OnComplete(relPath, desc)
	return desc, nil
}

// OutputConfig outputs the config blob to the remote storage.
func (ro *remoteOutput) OutputConfig(ctx context.Context, mediaType, digest string, size int64, reader io.Reader, hooks hooks.Hooks) (ocispec.Descriptor, error) {
	desc := ocispec.Descriptor{
		MediaType: mediaType,
		Digest:    godigest.Digest(digest),
		Size:      size,
	}

	reader = hooks.OnStart(digest, size, reader)
	exist, err := ro.remote.Blobs().Exists(ctx, desc)
	if err != nil {
		hooks.OnError(digest, err)
		return ocispec.Descriptor{}, fmt.Errorf("failed to check if blob exists: %w", err)
	}

	if exist {
		hooks.OnComplete(digest, desc)
		return desc, nil
	}

	if err = ro.remote.Blobs().Push(ctx, desc, reader); err != nil {
		hooks.OnError(digest, err)
		return ocispec.Descriptor{}, fmt.Errorf("failed to push config to storage: %w", err)
	}

	hooks.OnComplete(digest, desc)
	return desc, nil
}

// OutputManifest outputs the manifest blob to the remote storage.
func (ro *remoteOutput) OutputManifest(ctx context.Context, mediaType, digest string, size int64, reader io.Reader, hooks hooks.Hooks) (ocispec.Descriptor, error) {
	desc := ocispec.Descriptor{
		MediaType: mediaType,
		Digest:    godigest.Digest(digest),
		Size:      size,
	}

	reader = hooks.OnStart(digest, size, reader)
	exist, err := ro.remote.Manifests().Exists(ctx, desc)
	if err != nil {
		hooks.OnError(digest, err)
		return ocispec.Descriptor{}, fmt.Errorf("failed to check if blob exists: %w", err)
	}

	if exist {
		hooks.OnComplete(digest, desc)
		return desc, nil
	}

	if err = ro.remote.Manifests().Push(ctx, desc, reader); err != nil {
		hooks.OnError(digest, err)
		return ocispec.Descriptor{}, fmt.Errorf("failed to push manifest to storage: %w", err)
	}

	// Tag the manifest.
	if err = ro.remote.Tag(ctx, desc, ro.tag); err != nil {
		hooks.OnError(digest, err)
		return ocispec.Descriptor{}, fmt.Errorf("failed to tag manifest: %w", err)
	}

	hooks.OnComplete(digest, desc)
	return desc, nil
}
