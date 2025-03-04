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
	"os"
	"path/filepath"
	"time"

	"github.com/CloudNativeAI/modctl/pkg/archiver"
	"github.com/CloudNativeAI/modctl/pkg/modelfile"
	"github.com/CloudNativeAI/modctl/pkg/storage"

	modelspec "github.com/CloudNativeAI/model-spec/specs-go/v1"
	godigest "github.com/opencontainers/go-digest"
	spec "github.com/opencontainers/image-spec/specs-go"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

// BuildLayer converts the file to the image blob and push it to the storage.
func BuildLayer(ctx context.Context, store storage.Storage, mediaType, workDir, repo, path string) (ocispec.Descriptor, error) {
	info, err := os.Stat(path)
	if err != nil {
		return ocispec.Descriptor{}, fmt.Errorf("failed to get file info: %w", err)
	}

	if info.IsDir() {
		return ocispec.Descriptor{}, fmt.Errorf("%s is a directory and not supported yet", path)
	}

	workDirPath, err := filepath.Abs(workDir)
	if err != nil {
		return ocispec.Descriptor{}, fmt.Errorf("failed to get absolute path of workDir: %w", err)
	}

	reader, err := archiver.Tar(path, workDirPath)
	if err != nil {
		return ocispec.Descriptor{}, fmt.Errorf("failed to tar file: %w", err)
	}

	digest, size, err := store.PushBlob(ctx, repo, reader, ocispec.Descriptor{})
	if err != nil {
		return ocispec.Descriptor{}, fmt.Errorf("failed to push blob to storage: %w", err)
	}

	// Gets the relative path of the file as annotation.
	//nolint:typecheck
	relPath, err := filepath.Rel(workDirPath, path)
	if err != nil {
		return ocispec.Descriptor{}, fmt.Errorf("failed to get relative path: %w", err)
	}

	return ocispec.Descriptor{
		MediaType: mediaType,
		Digest:    godigest.Digest(digest),
		Size:      size,
		Annotations: map[string]string{
			modelspec.AnnotationFilepath: relPath,
		},
	}, nil
}

// buildModelConfig builds the model config.
func buildModelConfig(modelfile modelfile.Modelfile) (*modelspec.Model, error) {
	config := modelspec.ModelConfig{
		Architecture: modelfile.GetArch(),
		Format:       modelfile.GetFormat(),
		Precision:    modelfile.GetPrecision(),
		Quantization: modelfile.GetQuantization(),
		ParamSize:    modelfile.GetParamsize(),
	}

	createdAt := time.Now()
	descriptor := modelspec.ModelDescriptor{
		CreatedAt: &createdAt,
		Family:    modelfile.GetFamily(),
		Name:      modelfile.GetName(),
	}

	fs := modelspec.ModelFS{
		Type: "layers",
	}

	return &modelspec.Model{
		Config:     config,
		Descriptor: descriptor,
		ModelFS:    fs,
	}, nil
}

// BuildConfig builds the image config and push it to the storage.
func BuildConfig(ctx context.Context, store storage.Storage, modelfile modelfile.Modelfile, repo string) (ocispec.Descriptor, error) {
	config, err := buildModelConfig(modelfile)
	if err != nil {
		return ocispec.Descriptor{}, fmt.Errorf("failed to build model config: %w", err)
	}

	configJSON, err := json.Marshal(config)
	if err != nil {
		return ocispec.Descriptor{}, fmt.Errorf("failed to marshal config: %w", err)
	}

	digest, size, err := store.PushBlob(ctx, repo, bytes.NewReader(configJSON), ocispec.Descriptor{})
	if err != nil {
		return ocispec.Descriptor{}, fmt.Errorf("failed to push config to storage: %w", err)
	}

	return ocispec.Descriptor{
		MediaType: modelspec.MediaTypeModelConfig,
		Size:      size,
		Digest:    godigest.Digest(digest),
	}, nil
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
