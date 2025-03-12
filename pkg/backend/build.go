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
	"io"

	"github.com/CloudNativeAI/modctl/internal/pb"
	internalpb "github.com/CloudNativeAI/modctl/internal/pb"
	"github.com/CloudNativeAI/modctl/pkg/backend/build"
	"github.com/CloudNativeAI/modctl/pkg/backend/build/hooks"
	"github.com/CloudNativeAI/modctl/pkg/backend/processor"
	"github.com/CloudNativeAI/modctl/pkg/config"
	"github.com/CloudNativeAI/modctl/pkg/modelfile"

	modelspec "github.com/CloudNativeAI/model-spec/specs-go/v1"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

// Build builds the user materials into the OCI image which follows the Model Spec.
func (b *backend) Build(ctx context.Context, modelfilePath, workDir, target string, cfg *config.Build) error {
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

	// using the local output by default.
	outputType := build.OutputTypeLocal
	if cfg.OutputRemote {
		outputType = build.OutputTypeRemote
	}

	opts := []build.Option{
		build.WithPlainHTTP(cfg.PlainHTTP),
		build.WithInsecure(cfg.Insecure),
	}
	builder, err := build.NewBuilder(outputType, b.store, modelfile, repo, tag, opts...)
	if err != nil {
		return fmt.Errorf("failed to create builder: %w", err)
	}

	pb := internalpb.NewProgressBar()
	pb.Start()
	defer pb.Stop()

	layers := []ocispec.Descriptor{}
	layerDescs, err := b.process(ctx, builder, workDir, pb, cfg, b.getProcessors(modelfile)...)
	if err != nil {
		return fmt.Errorf("failed to process files: %w", err)
	}

	layers = append(layers, layerDescs...)
	// build the image config.
	configDesc, err := builder.BuildConfig(ctx, hooks.NewHooks(
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
		return fmt.Errorf("failed to build image config: %w", err)
	}

	// build the image manifest.
	_, err = builder.BuildManifest(ctx, layers, configDesc, manifestAnnotation(), hooks.NewHooks(
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
		return fmt.Errorf("failed to build image manifest: %w", err)
	}

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
func (b *backend) process(ctx context.Context, builder build.Builder, workDir string, pb *pb.ProgressBar, cfg *config.Build, processors ...processor.Processor) ([]ocispec.Descriptor, error) {
	descriptors := []ocispec.Descriptor{}
	for _, p := range processors {
		descs, err := p.Process(ctx, builder, workDir, processor.WithConcurrency(cfg.Concurrency), processor.WithProgressTracker(pb))
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
