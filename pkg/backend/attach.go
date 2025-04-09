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

package backend

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"sort"

	modelspec "github.com/CloudNativeAI/model-spec/specs-go/v1"
	godigest "github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"

	internalpb "github.com/CloudNativeAI/modctl/internal/pb"
	"github.com/CloudNativeAI/modctl/pkg/backend/build"
	buildconfig "github.com/CloudNativeAI/modctl/pkg/backend/build/config"
	"github.com/CloudNativeAI/modctl/pkg/backend/build/hooks"
	"github.com/CloudNativeAI/modctl/pkg/backend/build/interceptor"
	"github.com/CloudNativeAI/modctl/pkg/backend/processor"
	"github.com/CloudNativeAI/modctl/pkg/backend/remote"
	"github.com/CloudNativeAI/modctl/pkg/config"
	"github.com/CloudNativeAI/modctl/pkg/modelfile"
)

const (
	modelWeightConfigPriority = iota
	modelWeightPriority
	modelCodePriority
	modelDocPriority
)

var (
	// mediaTypePriorityMap defines the priority for layer sorting by group.
	mediaTypePriorityMap = map[string]int{
		modelspec.MediaTypeModelWeightConfig: modelWeightConfigPriority,
		modelspec.MediaTypeModelWeight:       modelWeightPriority,
		modelspec.MediaTypeModelCode:         modelCodePriority,
		modelspec.MediaTypeModelDoc:          modelDocPriority,
	}
)

// Attach attaches user materials into the model artifact which follows the Model Spec.
func (b *backend) Attach(ctx context.Context, filepath string, cfg *config.Attach) error {
	srcManifest, err := b.getManifest(ctx, cfg.Source, cfg)
	if err != nil {
		return fmt.Errorf("failed to get source manifest: %w", err)
	}

	srcModelConfig, err := b.getModelConfig(ctx, cfg.Source, srcManifest.Config, cfg)
	if err != nil {
		return fmt.Errorf("failed to get source model config: %w", err)
	}

	var foundLayer *ocispec.Descriptor
	for _, layer := range srcManifest.Layers {
		if anno := layer.Annotations; anno != nil {
			if anno[modelspec.AnnotationFilepath] == filepath {
				if !cfg.Force {
					return fmt.Errorf("file %s already exists, please use --force to overwrite if you want to attach it forcibly", filepath)
				}

				foundLayer = &layer
				break
			}
		}
	}

	layers := srcManifest.Layers
	if foundLayer != nil {
		// Remove the found layer from the layers slice as we need to replace it with the new layer.
		for i, layer := range layers {
			if layer.Digest == foundLayer.Digest && layer.MediaType == foundLayer.MediaType {
				layers = append(layers[:i], layers[i+1:]...)
				break
			}
		}
	}

	proc := b.getProcessor(filepath)
	if proc == nil {
		return fmt.Errorf("failed to get processor for file %s", filepath)
	}

	builder, err := b.getBuilder(cfg.Target, cfg)
	if err != nil {
		return fmt.Errorf("failed to create builder: %w", err)
	}

	pb := internalpb.NewProgressBar()
	pb.Start()
	defer pb.Stop()

	newLayers, err := proc.Process(ctx, builder, ".", processor.WithProgressTracker(pb))
	if err != nil {
		return fmt.Errorf("failed to process layers: %w", err)
	}

	// Append the new layers to the original layers.
	layers = append(layers, newLayers...)
	sortLayers(layers)

	diffIDs := []godigest.Digest{}
	for _, layer := range layers {
		diffIDs = append(diffIDs, layer.Digest)
	}
	// Return earlier if the diffID has no changed, which means the artifact has not changed.
	if reflect.DeepEqual(diffIDs, srcModelConfig.ModelFS.DiffIDs) {
		return nil
	}

	// Build the model config.
	modelConfig := &buildconfig.Model{
		Architecture: srcModelConfig.Config.Architecture,
		Format:       srcModelConfig.Config.Format,
		Precision:    srcModelConfig.Config.Precision,
		Quantization: srcModelConfig.Config.Quantization,
		ParamSize:    srcModelConfig.Config.ParamSize,
		Family:       srcModelConfig.Descriptor.Family,
		Name:         srcModelConfig.Descriptor.Name,
	}
	configDesc, err := builder.BuildConfig(ctx, layers, modelConfig, hooks.NewHooks(
		hooks.WithOnStart(func(name string, size int64, reader io.Reader) io.Reader {
			return pb.Add(internalpb.NormalizePrompt("Building config"), name, size, reader)
		}),
		hooks.WithOnError(func(name string, err error) {
			pb.Complete(name, fmt.Sprintf("Failed to build config: %v", err))
		}),
		hooks.WithOnComplete(func(name string, desc ocispec.Descriptor) {
			pb.Complete(name, fmt.Sprintf("%s %s", internalpb.NormalizePrompt("Built config"), desc.Digest))
		}),
	))
	if err != nil {
		return fmt.Errorf("failed to build model config: %w", err)
	}

	// Build the model manifest.
	_, err = builder.BuildManifest(ctx, layers, configDesc, srcManifest.Annotations, hooks.NewHooks(
		hooks.WithOnStart(func(name string, size int64, reader io.Reader) io.Reader {
			return pb.Add(internalpb.NormalizePrompt("Building manifest"), name, size, reader)
		}),
		hooks.WithOnError(func(name string, err error) {
			pb.Complete(name, fmt.Sprintf("Failed to build manifest: %v", err))
		}),
		hooks.WithOnComplete(func(name string, desc ocispec.Descriptor) {
			pb.Complete(name, fmt.Sprintf("%s %s", internalpb.NormalizePrompt("Built manifest"), desc.Digest))
		}),
	))
	if err != nil {
		return fmt.Errorf("failed to build model manifest: %w", err)
	}

	return nil
}

func (b *backend) getManifest(ctx context.Context, reference string, cfg *config.Attach) (*ocispec.Manifest, error) {
	ref, err := ParseReference(reference)
	if err != nil {
		return nil, fmt.Errorf("failed to parse source reference: %w", err)
	}

	repo, tag := ref.Repository(), ref.Tag()
	if repo == "" || tag == "" {
		return nil, fmt.Errorf("invalid repository or tag")
	}

	// Fetch from local storage if it is not remote.
	if !cfg.OutputRemote {
		manifestRaw, _, err := b.store.PullManifest(ctx, repo, tag)
		if err != nil {
			return nil, fmt.Errorf("failed to pull manifest: %w", err)
		}

		var manifest ocispec.Manifest
		if err := json.Unmarshal(manifestRaw, &manifest); err != nil {
			return nil, fmt.Errorf("failed to unmarshal manifest: %w", err)
		}

		return &manifest, nil
	}

	client, err := remote.New(repo, remote.WithPlainHTTP(cfg.PlainHTTP), remote.WithInsecure(cfg.Insecure))
	if err != nil {
		return nil, fmt.Errorf("failed to create remote client: %w", err)
	}

	_, manifestReader, err := client.Manifests().FetchReference(ctx, reference)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch manifest: %w", err)
	}
	defer manifestReader.Close()

	var manifest ocispec.Manifest
	if err := json.NewDecoder(manifestReader).Decode(&manifest); err != nil {
		return nil, fmt.Errorf("failed to decode manifest: %w", err)
	}

	return &manifest, nil
}

func (b *backend) getModelConfig(ctx context.Context, reference string, desc ocispec.Descriptor, cfg *config.Attach) (*modelspec.Model, error) {
	ref, err := ParseReference(reference)
	if err != nil {
		return nil, fmt.Errorf("failed to parse reference: %w", err)
	}

	repo := ref.Repository()
	if repo == "" {
		return nil, fmt.Errorf("repository name cannot be empty")
	}

	// Fetch from local storage if it is not remote.
	if !cfg.OutputRemote {
		reader, err := b.store.PullBlob(ctx, repo, desc.Digest.String())
		if err != nil {
			return nil, fmt.Errorf("failed to pull blob: %w", err)
		}
		defer reader.Close()

		var model modelspec.Model
		if err := json.NewDecoder(reader).Decode(&model); err != nil {
			return nil, fmt.Errorf("failed to decode model config: %w", err)
		}

		return &model, nil
	}

	client, err := remote.New(repo, remote.WithPlainHTTP(cfg.PlainHTTP), remote.WithInsecure(cfg.Insecure))
	if err != nil {
		return nil, fmt.Errorf("failed to create remote client: %w", err)
	}

	reader, err := client.Blobs().Fetch(ctx, desc)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch blob: %w", err)
	}
	defer reader.Close()

	var model modelspec.Model
	if err := json.NewDecoder(reader).Decode(&model); err != nil {
		return nil, fmt.Errorf("failed to decode model: %w", err)
	}

	return &model, nil
}

func (b *backend) getProcessor(filepath string) processor.Processor {
	if modelfile.IsFileType(filepath, modelfile.ConfigFilePatterns) {
		return processor.NewModelConfigProcessor(b.store, modelspec.MediaTypeModelWeightConfig, []string{filepath})
	}

	if modelfile.IsFileType(filepath, modelfile.ModelFilePatterns) {
		return processor.NewModelProcessor(b.store, modelspec.MediaTypeModelWeight, []string{filepath})
	}

	if modelfile.IsFileType(filepath, modelfile.CodeFilePatterns) {
		return processor.NewCodeProcessor(b.store, modelspec.MediaTypeModelCode, []string{filepath})
	}

	if modelfile.IsFileType(filepath, modelfile.DocFilePatterns) {
		return processor.NewDocProcessor(b.store, modelspec.MediaTypeModelDoc, []string{filepath})
	}

	return nil
}

func (b *backend) getBuilder(reference string, cfg *config.Attach) (build.Builder, error) {
	ref, err := ParseReference(reference)
	if err != nil {
		return nil, fmt.Errorf("failed to parse target reference: %w", err)
	}

	repo, tag := ref.Repository(), ref.Tag()
	if repo == "" || tag == "" {
		return nil, fmt.Errorf("invalid repository or tag")
	}

	outputType := build.OutputTypeLocal
	if cfg.OutputRemote {
		outputType = build.OutputTypeRemote
	}

	opts := []build.Option{
		build.WithPlainHTTP(cfg.PlainHTTP),
		build.WithInsecure(cfg.Insecure),
	}
	if cfg.Nydusify {
		opts = append(opts, build.WithInterceptor(interceptor.NewNydus()))
	}

	builder, err := build.NewBuilder(outputType, b.store, repo, tag, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create builder: %w", err)
	}

	return builder, nil
}

// sortLayers sorts the layers group by mediaType and sort by the filepath.
func sortLayers(layers []ocispec.Descriptor) {
	sort.SliceStable(layers, func(i, j int) bool {
		priorityI := mediaTypePriorityMap[layers[i].MediaType]
		priorityJ := mediaTypePriorityMap[layers[j].MediaType]

		if priorityI != priorityJ {
			return priorityI < priorityJ
		}

		// Sort by the filepath if the priority is same.
		var filepathI, filepathJ string
		if layers[i].Annotations != nil {
			filepathI = layers[i].Annotations[modelspec.AnnotationFilepath]
		}
		if layers[j].Annotations != nil {
			filepathJ = layers[j].Annotations[modelspec.AnnotationFilepath]
		}
		return filepathI < filepathJ
	})
}
