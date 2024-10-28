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

package build

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	modelspec "github.com/CloudNativeAI/modctl/pkg/oci/spec"
	"github.com/CloudNativeAI/modctl/pkg/storage"

	godigest "github.com/opencontainers/go-digest"
	spec "github.com/opencontainers/image-spec/specs-go"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

// DescriptorEmptyJSON is the descriptor of a blob with content of `{}`.
var DescriptorEmptyJSON = ocispec.Descriptor{
	MediaType: ocispec.MediaTypeImageConfig,
	Digest:    `sha256:44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a`,
	Size:      2,
	Data:      []byte(`{}`),
}

// BuildLayer converts the file to the image blob and push it to the storage.
func BuildLayer(ctx context.Context, store storage.Storage, repo, path string) (ocispec.Descriptor, error) {
	reader, err := TarFileToStream(path)
	if err != nil {
		return ocispec.Descriptor{}, fmt.Errorf("failed to tar file: %w", err)
	}

	digest, size, err := store.PushBlob(ctx, repo, reader)
	if err != nil {
		return ocispec.Descriptor{}, fmt.Errorf("failed to push blob to storage: %w", err)
	}

	return ocispec.Descriptor{
		ArtifactType: modelspec.ArtifactTypeModelLayer,
		MediaType:    ocispec.MediaTypeImageLayer,
		Digest:       godigest.Digest(digest),
		Size:         size,
	}, nil
}

// BuildConfig builds the image config and push it to the storage.
func BuildConfig(ctx context.Context, store storage.Storage, repo string) (ocispec.Descriptor, error) {
	// by default using the empty JSON config.
	_, _, err := store.PushBlob(ctx, repo, bytes.NewReader(DescriptorEmptyJSON.Data))
	if err != nil {
		return ocispec.Descriptor{}, fmt.Errorf("failed to push config to storage: %w", err)
	}

	return DescriptorEmptyJSON, nil
}

// BuildManifest builds the manifest and push it to the storage.
func BuildManifest(ctx context.Context, store storage.Storage, repo, reference string, layers []ocispec.Descriptor, config ocispec.Descriptor, annotations map[string]string) (ocispec.Descriptor, error) {
	manifest := &ocispec.Manifest{
		Versioned: spec.Versioned{
			SchemaVersion: 2,
		},
		Annotations:  annotations,
		ArtifactType: modelspec.ArtifactTypeModelManifest,
		Config: ocispec.Descriptor{
			MediaType: config.MediaType,
			Digest:    config.Digest,
			Size:      config.Size,
		},
		MediaType: ocispec.MediaTypeImageManifest,
		Layers:    layers,
	}

	manifestJSON, err := json.Marshal(manifest)
	if err != nil {
		return ocispec.Descriptor{}, fmt.Errorf("failed to marshal manifest: %w", err)
	}

	digestStr, err := store.PushManifest(ctx, repo, reference, manifestJSON)
	if err != nil {
		return ocispec.Descriptor{}, fmt.Errorf("failed to push manifest to storage: %w", err)
	}

	return ocispec.Descriptor{
		MediaType: manifest.MediaType,
		Digest:    godigest.Digest(digestStr),
		Size:      int64(len(manifestJSON)),
	}, nil
}
