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
	"os"
	pathfilepath "path/filepath"
	"reflect"
	"slices"
	"sort"

	legacymodelspec "github.com/dragonflyoss/model-spec/specs-go/v1"
	modelspec "github.com/modelpack/model-spec/specs-go/v1"
	godigest "github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/sirupsen/logrus"

	internalpb "github.com/modelpack/modctl/internal/pb"
	"github.com/modelpack/modctl/pkg/backend/build"
	buildconfig "github.com/modelpack/modctl/pkg/backend/build/config"
	"github.com/modelpack/modctl/pkg/backend/build/hooks"
	"github.com/modelpack/modctl/pkg/backend/processor"
	"github.com/modelpack/modctl/pkg/backend/remote"
	"github.com/modelpack/modctl/pkg/config"
	"github.com/modelpack/modctl/pkg/modelfile"
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
		legacymodelspec.MediaTypeModelWeightConfig: modelWeightConfigPriority,
		legacymodelspec.MediaTypeModelWeight:       modelWeightPriority,
		legacymodelspec.MediaTypeModelCode:         modelCodePriority,
		legacymodelspec.MediaTypeModelDoc:          modelDocPriority,
	}
)

// Attach attaches user materials into the model artifact which follows the Model Spec.
func (b *backend) Attach(ctx context.Context, filepath string, cfg *config.Attach) error {
	logrus.Infof("attach: starting attach operation for file %s [config: %+v]", filepath, cfg)
	srcManifest, err := b.getManifest(ctx, cfg.Source, cfg.OutputRemote, cfg.PlainHTTP, cfg.Insecure)
	if err != nil {
		return fmt.Errorf("failed to get source manifest: %w", err)
	}

	srcModelConfig, err := b.getModelConfig(ctx, cfg.Source, srcManifest.Config, cfg.OutputRemote, cfg.PlainHTTP, cfg.Insecure)
	if err != nil {
		return fmt.Errorf("failed to get source model config: %w", err)
	}

	logrus.Infof("attach: loaded source model config [%+v]", srcModelConfig)

	proc := b.getProcessor(cfg.DestinationDir, filepath, cfg.Raw)
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

	destPath := filepath
	if cfg.DestinationDir != "" {
		destPath = pathfilepath.Join(cfg.DestinationDir, pathfilepath.Base(filepath))
	}

	layers := srcManifest.Layers
	// If attach a normal file, we need to process it and create a new layer.
	if !cfg.Config {
		var foundLayer *ocispec.Descriptor
		for _, layer := range srcManifest.Layers {
			if anno := layer.Annotations; anno != nil {
				if anno[modelspec.AnnotationFilepath] == destPath || anno[legacymodelspec.AnnotationFilepath] == destPath {
					if !cfg.Force {
						return fmt.Errorf("file %s already exists, please use --force to overwrite if you want to attach it forcibly", destPath)
					}

					foundLayer = &layer
					break
				}
			}
		}

		logrus.Infof("attach: found existing layer for file %s [%+v]", filepath, foundLayer)
		if foundLayer != nil {
			// Remove the found layer from the layers slice as we need to replace it with the new layer.
			for i, layer := range layers {
				if layer.Digest == foundLayer.Digest && layer.MediaType == foundLayer.MediaType {
					layers = slices.Delete(layers, i, i+1)
					break
				}
			}
		}

		newLayers, err := proc.Process(ctx, builder, ".", processor.WithProgressTracker(pb))
		if err != nil {
			return fmt.Errorf("failed to process layers: %w", err)
		}

		// Append the new layers to the original layers.
		layers = append(layers, newLayers...)
		sortLayers(layers)

		logrus.Debugf("attach: generated sorted layers [layers: %+v]", layers)

		diffIDs := []godigest.Digest{}
		for _, layer := range layers {
			diffIDs = append(diffIDs, layer.Digest)
		}
		// Return earlier if the diffID has no changed, which means the artifact has not changed.
		if reflect.DeepEqual(diffIDs, srcModelConfig.ModelFS.DiffIDs) {
			return nil
		}
	}

	var config legacymodelspec.Model
	if !cfg.Config {
		var reasoning bool
		if srcModelConfig.Config.Capabilities != nil && srcModelConfig.Config.Capabilities.Reasoning != nil {
			reasoning = *srcModelConfig.Config.Capabilities.Reasoning
		}

		config, err = build.BuildModelConfig(&buildconfig.Model{
			Architecture:   srcModelConfig.Config.Architecture,
			Format:         srcModelConfig.Config.Format,
			Precision:      srcModelConfig.Config.Precision,
			Quantization:   srcModelConfig.Config.Quantization,
			ParamSize:      srcModelConfig.Config.ParamSize,
			Family:         srcModelConfig.Descriptor.Family,
			Name:           srcModelConfig.Descriptor.Name,
			SourceURL:      srcModelConfig.Descriptor.SourceURL,
			SourceRevision: srcModelConfig.Descriptor.Revision,
			Reasoning:      reasoning,
		}, layers)
		if err != nil {
			return fmt.Errorf("failed to build model config: %w", err)
		}
	} else {
		configFile, err := os.Open(filepath)
		if err != nil {
			return fmt.Errorf("failed to open config file: %w", err)
		}
		defer configFile.Close()

		if err := json.NewDecoder(configFile).Decode(&config); err != nil {
			return fmt.Errorf("failed to decode config file %s: %w", filepath, err)
		}
	}

	logrus.Infof("attach: built model config [%+v]", config)

	configDesc, err := builder.BuildConfig(ctx, config, hooks.NewHooks(
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

	logrus.Infof("attach: successfully attached file %s", filepath)
	return nil
}

func (b *backend) getManifest(ctx context.Context, reference string, fromRemote, plainHTTP, insecure bool) (*ocispec.Manifest, error) {
	ref, err := ParseReference(reference)
	if err != nil {
		return nil, fmt.Errorf("failed to parse source reference: %w", err)
	}

	repo, tag := ref.Repository(), ref.Tag()
	if repo == "" || tag == "" {
		return nil, fmt.Errorf("invalid repository or tag")
	}

	// Fetch from local storage if it is not remote.
	if !fromRemote {
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

	client, err := remote.New(repo, remote.WithPlainHTTP(plainHTTP), remote.WithInsecure(insecure))
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

func (b *backend) getModelConfig(ctx context.Context, reference string, desc ocispec.Descriptor, fromRemote, plainHTTP, insecure bool) (*modelspec.Model, error) {
	ref, err := ParseReference(reference)
	if err != nil {
		return nil, fmt.Errorf("failed to parse reference: %w", err)
	}

	repo := ref.Repository()
	if repo == "" {
		return nil, fmt.Errorf("repository name cannot be empty")
	}

	// Fetch from local storage if it is not remote.
	if !fromRemote {
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

	client, err := remote.New(repo, remote.WithPlainHTTP(plainHTTP), remote.WithInsecure(insecure))
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

func (b *backend) getProcessor(destDir, filepath string, rawMediaType bool) processor.Processor {
	if modelfile.IsFileType(filepath, modelfile.ConfigFilePatterns) {
		mediaType := legacymodelspec.MediaTypeModelWeightConfig
		if rawMediaType {
			mediaType = legacymodelspec.MediaTypeModelWeightConfigRaw
		}
		return processor.NewModelConfigProcessor(b.store, mediaType, []string{filepath}, destDir)
	}

	if modelfile.IsFileType(filepath, modelfile.ModelFilePatterns) {
		mediaType := legacymodelspec.MediaTypeModelWeight
		if rawMediaType {
			mediaType = legacymodelspec.MediaTypeModelWeightRaw
		}
		return processor.NewModelProcessor(b.store, mediaType, []string{filepath}, destDir)
	}

	if modelfile.IsFileType(filepath, modelfile.CodeFilePatterns) {
		mediaType := legacymodelspec.MediaTypeModelCode
		if rawMediaType {
			mediaType = legacymodelspec.MediaTypeModelCodeRaw
		}
		return processor.NewCodeProcessor(b.store, mediaType, []string{filepath}, destDir)
	}

	if modelfile.IsFileType(filepath, modelfile.DocFilePatterns) {
		mediaType := legacymodelspec.MediaTypeModelDoc
		if rawMediaType {
			mediaType = legacymodelspec.MediaTypeModelDocRaw
		}
		return processor.NewDocProcessor(b.store, mediaType, []string{filepath}, destDir)
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
			if layers[i].Annotations[modelspec.AnnotationFilepath] != "" {
				filepathI = layers[i].Annotations[modelspec.AnnotationFilepath]
			} else {
				filepathI = layers[i].Annotations[legacymodelspec.AnnotationFilepath]
			}

		}
		if layers[j].Annotations != nil {
			if layers[j].Annotations[modelspec.AnnotationFilepath] != "" {
				filepathJ = layers[j].Annotations[modelspec.AnnotationFilepath]
			} else {
				filepathJ = layers[j].Annotations[legacymodelspec.AnnotationFilepath]
			}

		}
		return filepathI < filepathJ
	})
}
