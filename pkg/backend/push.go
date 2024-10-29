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

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2/content/oci"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
	"oras.land/oras-go/v2/registry/remote/credentials"

	"github.com/CloudNativeAI/modctl/pkg/storage"
)

// Push pushes the image to the registry.
func (b *backend) Push(ctx context.Context, target string, opts ...Option) error {
	// apply options.
	options := &Options{}
	for _, opt := range opts {
		opt(options)
	}

	// parse the repository and tag from the target.
	ref, err := ParseReference(target)
	if err != nil {
		return fmt.Errorf("failed to parse the target: %w", err)
	}

	repo, tag := ref.Repository(), ref.Tag()

	// create the src storage from the image storage path.
	src, err := oci.New(imageStoragePath(repo))
	if err != nil {
		return fmt.Errorf("failed to create source storage: %w", err)
	}

	// create the dst storage from the remote repository.
	dst, err := remote.NewRepository(target)
	if err != nil {
		return fmt.Errorf("failed to create remote repository: %w", err)
	}

	// gets the credentials store.
	credStore, err := credentials.NewStoreFromDocker(credentials.StoreOptions{AllowPlaintextPut: true})
	if err != nil {
		return fmt.Errorf("failed to create credential store: %w", err)
	}

	dst.Client = &auth.Client{
		Cache:      auth.NewCache(),
		Credential: credentials.Credential(credStore),
	}

	if options.plainHTTP {
		dst.PlainHTTP = true
	}

	manifestDesc, err := src.Resolve(ctx, tag)
	if err != nil {
		return fmt.Errorf("failed to resolve the manifest from source: %w", err)
	}

	manifestReader, err := src.Fetch(ctx, manifestDesc)
	if err != nil {
		return fmt.Errorf("failed to fetch the manifest from source: %w", err)
	}

	defer manifestReader.Close()

	var manifest ocispec.Manifest
	if err := json.NewDecoder(manifestReader).Decode(&manifest); err != nil {
		return fmt.Errorf("failed to decode the manifest: %w", err)
	}

	// create the progress bar to track the progress of push.
	pb := NewProgressBar()
	defer pb.Wait()

	// copy the image to the destination, there are three steps:
	// 1. copy the layers.
	// 2. copy the config.
	// 3. copy the manifest.
	// note: the order is important, manifest should be pushed at last.

	// copy the layers.
	// TODO: parallelize the layer copy.
	for _, layer := range manifest.Layers {
		if err := pushIfNotExist(ctx, pb, promptCopyingBlob, src, dst, layer, repo, tag); err != nil {
			return fmt.Errorf("failed to push blob to remote: %w", err)
		}
	}

	// copy the config.
	if err := pushIfNotExist(ctx, pb, promptCopyingConfig, src, dst, manifest.Config, repo, tag); err != nil {
		return fmt.Errorf("failed to push config to remote: %w", err)
	}

	// copy the manifest.
	if err := pushIfNotExist(ctx, pb, promptCopyingManifest, src, dst, manifestDesc, repo, tag); err != nil {
		return fmt.Errorf("failed to push manifest to remote: %w", err)
	}

	return nil
}

// imageStoragePath returns the image storage path.
func imageStoragePath(repo string) string {
	contentDir, _ := storage.GetDefaultContentDir()
	return filepath.Join(contentDir, repo)
}

// pushIfNotExist copies the content from the src storage to the dst storage if the content does not exist.
func pushIfNotExist(ctx context.Context, pb *ProgressBar, prompt string, src *oci.Store, dst *remote.Repository, desc ocispec.Descriptor, repo, tag string) error {
	// check whether the content exists in the destination storage.
	exist, err := dst.Exists(ctx, desc)
	if err != nil {
		return fmt.Errorf("failed to check existence to remote: %w", err)
	}

	if exist {
		pb.PrintMessage(prompt, desc, "skipped: already exists")
		return nil
	}

	// fetch the content from the source storage.
	content, err := src.Fetch(ctx, desc)
	if err != nil {
		return fmt.Errorf("failed to fetch the content from source: %w", err)
	}

	defer content.Close()
	// push the content to the destination, and wrap the content reader for progress bar,
	// manifest should use dst.Manifests().Push, others should use dst.Blobs().Push.
	if desc.MediaType == ocispec.MediaTypeImageManifest {
		if err := dst.Manifests().Push(ctx, desc, pb.Add(prompt, desc, content)); err != nil {
			return err
		}

		// push tag
		if err := dst.Tag(ctx, desc, tag); err != nil {
			return err
		}
	} else {
		if err := dst.Blobs().Push(ctx, desc, pb.Add(prompt, desc, content)); err != nil {
			return err
		}
	}

	return nil
}
