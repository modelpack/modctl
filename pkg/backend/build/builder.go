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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/CloudNativeAI/modctl/pkg/archiver"
	"github.com/CloudNativeAI/modctl/pkg/backend/build/hooks"
	"github.com/CloudNativeAI/modctl/pkg/modelfile"
	"github.com/CloudNativeAI/modctl/pkg/storage"

	modelspec "github.com/CloudNativeAI/model-spec/specs-go/v1"
	sha256 "github.com/minio/sha256-simd"
	spec "github.com/opencontainers/image-spec/specs-go"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

// tarHeaderSize is the size of a tar header.
// TODO: the real size should be calculated based on the actual stream,
// now we use a fixed size in order to avoid extra read costs.
const tarHeaderSize = 512

// OutputType defines the type of output to generate.
type OutputType string

const (
	// OutputTypeLocal indicates that the output should be stored locally in modctl local storage.
	OutputTypeLocal OutputType = "local"
	// OutputTypeRemote indicates that the output should be pushed to a remote registry directly.
	OutputTypeRemote OutputType = "remote"
)

// Builder is an interface for building artifacts.
type Builder interface {
	// BuildLayer builds the layer blob from the given file path.
	BuildLayer(ctx context.Context, mediaType, workDir, path string, hooks hooks.Hooks) (ocispec.Descriptor, error)

	// BuildConfig builds the config blob of the artifact.
	BuildConfig(ctx context.Context, hooks hooks.Hooks) (ocispec.Descriptor, error)

	// BuildManifest builds the manifest blob of the artifact.
	BuildManifest(ctx context.Context, layers []ocispec.Descriptor, config ocispec.Descriptor, annotations map[string]string, hooks hooks.Hooks) (ocispec.Descriptor, error)
}

type OutputStrategy interface {
	// OutputLayer outputs the layer blob to the storage (local or remote).
	OutputLayer(ctx context.Context, mediaType, workDir, relPath string, size int64, reader io.Reader, hooks hooks.Hooks) (ocispec.Descriptor, error)

	// OutputConfig outputs the config blob to the storage (local or remote).
	OutputConfig(ctx context.Context, mediaType, digest string, size int64, reader io.Reader, hooks hooks.Hooks) (ocispec.Descriptor, error)

	// OutputManifest outputs the manifest blob to the storage (local or remote).
	OutputManifest(ctx context.Context, mediaType, digest string, size int64, reader io.Reader, hooks hooks.Hooks) (ocispec.Descriptor, error)
}

// NewBuilder creates a new builder instance.
func NewBuilder(outputType OutputType, store storage.Storage, modelfile modelfile.Modelfile, repo, tag string, opts ...Option) (Builder, error) {
	cfg := &config{}
	for _, opt := range opts {
		opt(cfg)
	}

	var (
		strategy OutputStrategy
		err      error
	)
	switch outputType {
	case OutputTypeLocal:
		strategy, err = NewLocalOutput(cfg, store, repo, tag)
	case OutputTypeRemote:
		strategy, err = NewRemoteOutput(cfg, repo, tag)
	default:
		return nil, fmt.Errorf("unsupported output type: %s", outputType)
	}

	if err != nil {
		return nil, err
	}

	return &abstractBuilder{
		store:     store,
		modelfile: modelfile,
		repo:      repo,
		tag:       tag,
		strategy:  strategy,
	}, nil
}

// abstractBuilder is an abstract implementation of the Builder interface.
type abstractBuilder struct {
	store     storage.Storage
	modelfile modelfile.Modelfile
	repo      string
	tag       string
	// strategy is the output strategy used to output the blob.
	strategy OutputStrategy
}

func (ab *abstractBuilder) BuildLayer(ctx context.Context, mediaType, workDir, path string, hooks hooks.Hooks) (ocispec.Descriptor, error) {
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

	// Gets the relative path of the file as annotation.
	//nolint:typecheck
	relPath, err := filepath.Rel(workDirPath, path)
	if err != nil {
		return ocispec.Descriptor{}, fmt.Errorf("failed to get relative path: %w", err)
	}

	reader, err := archiver.Tar(path, workDirPath)
	if err != nil {
		return ocispec.Descriptor{}, fmt.Errorf("failed to tar file: %w", err)
	}

	return ab.strategy.OutputLayer(ctx, mediaType, workDir, relPath, info.Size()+tarHeaderSize, reader, hooks)
}

func (ab *abstractBuilder) BuildConfig(ctx context.Context, hooks hooks.Hooks) (ocispec.Descriptor, error) {
	config, err := buildModelConfig(ab.modelfile)
	if err != nil {
		return ocispec.Descriptor{}, fmt.Errorf("failed to build model config: %w", err)
	}

	configJSON, err := json.Marshal(config)
	if err != nil {
		return ocispec.Descriptor{}, fmt.Errorf("failed to marshal config: %w", err)
	}

	digest := fmt.Sprintf("sha256:%x", sha256.Sum256(configJSON))
	return ab.strategy.OutputConfig(ctx, modelspec.MediaTypeModelConfig, digest, int64(len(configJSON)), bytes.NewReader(configJSON), hooks)
}

func (ab *abstractBuilder) BuildManifest(ctx context.Context, layers []ocispec.Descriptor, config ocispec.Descriptor, annotations map[string]string, hooks hooks.Hooks) (ocispec.Descriptor, error) {
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

	digest := fmt.Sprintf("sha256:%x", sha256.Sum256(manifestJSON))
	return ab.strategy.OutputManifest(ctx, manifest.MediaType, digest, int64(len(manifestJSON)), bytes.NewReader(manifestJSON), hooks)
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
