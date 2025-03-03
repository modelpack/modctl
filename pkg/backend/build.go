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
	"fmt"

	"github.com/CloudNativeAI/modctl/pkg/backend/build"
	"github.com/CloudNativeAI/modctl/pkg/backend/processor"
	"github.com/CloudNativeAI/modctl/pkg/modelfile"

	modelspec "github.com/CloudNativeAI/model-spec/specs-go/v1"
	humanize "github.com/dustin/go-humanize"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

// Build builds the user materials into the OCI image which follows the Model Spec.
func (b *backend) Build(ctx context.Context, modelfilePath, workDir, target string, opts ...Option) error {
	// apply options.
	options := &Options{}
	for _, opt := range opts {
		opt(options)
	}

	// parse the repo name and tag name from target.
	ref, err := ParseReference(target)
	if err != nil {
		return fmt.Errorf("failed to parse target: %w", err)
	}

	modelfile, err := modelfile.NewModelfile(modelfilePath)
	if err != nil {
		return fmt.Errorf("failed to parse modelfile: %w", err)
	}

	repo, tag := ref.Repository(), ref.Tag()
	if tag == "" {
		return fmt.Errorf("tag is required")
	}

	layers := []ocispec.Descriptor{}
	layerDescs, err := b.process(ctx, workDir, repo, options, b.getProcessors(modelfile)...)
	if err != nil {
		return fmt.Errorf("failed to process files: %w", err)
	}

	layers = append(layers, layerDescs...)

	// build the image config.
	configDesc, err := build.BuildConfig(ctx, b.store, modelfile, repo)
	if err != nil {
		return fmt.Errorf("failed to build image config: %w", err)
	}

	fmt.Printf("%s => %s (%s)\n", "Built config", configDesc.Digest, humanize.IBytes(uint64(configDesc.Size)))

	// build the image manifest.
	manifestDesc, err := build.BuildManifest(ctx, b.store, repo, tag, layers, configDesc, manifestAnnotation())
	if err != nil {
		return fmt.Errorf("failed to build image manifest: %w", err)
	}

	fmt.Printf("%s => %s (%s)\n", "Built manifest", manifestDesc.Digest, humanize.IBytes(uint64(manifestDesc.Size)))
	return nil
}

func (b *backend) getProcessors(modelfile modelfile.Modelfile) []processor.Processor {
	processors := []processor.Processor{}

	if configs := modelfile.GetConfigs(); len(configs) > 0 {
		processors = append(processors, processor.NewModelConfigProcessor(b.store, modelspec.MediaTypeModelWeightConfig, configs))
	}

	if models := modelfile.GetModels(); len(models) > 0 {
		processors = append(processors, processor.NewModelProcessor(b.store, modelspec.MediaTypeModelWeight, models))
	}

	if codes := modelfile.GetCodes(); len(codes) > 0 {
		processors = append(processors, processor.NewCodeProcessor(b.store, modelspec.MediaTypeModelCode, codes))
	}

	if docs := modelfile.GetDocs(); len(docs) > 0 {
		processors = append(processors, processor.NewDocProcessor(b.store, modelspec.MediaTypeModelDoc, docs))
	}

	return processors
}

// process walks the user work directory and process the identified files.
func (b *backend) process(ctx context.Context, workDir string, repo string, opts *Options, processors ...processor.Processor) ([]ocispec.Descriptor, error) {
	descriptors := []ocispec.Descriptor{}
	for _, p := range processors {
		descs, err := p.Process(ctx, workDir, repo, processor.WithConcurrency(opts.concurrency))
		if err != nil {
			return nil, err
		}

		descriptors = append(descriptors, descs...)
	}

	return descriptors, nil
}

// manifestAnnotation returns the annotations for the manifest.
func manifestAnnotation() map[string]string {
	// placeholder for future expansion of annotations.
	anno := map[string]string{}
	return anno
}
