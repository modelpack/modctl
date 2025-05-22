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
	"os"
	"path/filepath"

	modelspec "github.com/CloudNativeAI/model-spec/specs-go/v1"
	retry "github.com/avast/retry-go/v4"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"

	internalpb "github.com/CloudNativeAI/modctl/internal/pb"
	"github.com/CloudNativeAI/modctl/pkg/backend/build"
	buildconfig "github.com/CloudNativeAI/modctl/pkg/backend/build/config"
	"github.com/CloudNativeAI/modctl/pkg/backend/build/hooks"
	"github.com/CloudNativeAI/modctl/pkg/backend/build/interceptor"
	"github.com/CloudNativeAI/modctl/pkg/backend/processor"
	"github.com/CloudNativeAI/modctl/pkg/config"
	"github.com/CloudNativeAI/modctl/pkg/modelfile"
	"github.com/CloudNativeAI/modctl/pkg/source"
)

const (
	// annotationModelfile is the annotation key for the Modelfile.
	annotationModelfile = "org.cnai.modctl.modelfile"
)

// Build builds the user materials into the model artifact which follows the Model Spec.
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

	sourceInfo, err := getSourceInfo(workDir, cfg)
	if err != nil {
		return fmt.Errorf("failed to get source info: %w", err)
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
	if cfg.Nydusify {
		opts = append(opts, build.WithInterceptor(interceptor.NewNydus()))
	}

	builder, err := build.NewBuilder(outputType, b.store, repo, tag, opts...)
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

	revision := sourceInfo.Commit
	if revision != "" && sourceInfo.Dirty {
		revision += "-dirty"
	}
	// Build the model config.
	modelConfig := &buildconfig.Model{
		Architecture:   modelfile.GetArch(),
		Format:         modelfile.GetFormat(),
		Precision:      modelfile.GetPrecision(),
		Quantization:   modelfile.GetQuantization(),
		ParamSize:      modelfile.GetParamsize(),
		Family:         modelfile.GetFamily(),
		Name:           modelfile.GetName(),
		SourceURL:      sourceInfo.URL,
		SourceRevision: revision,
	}

	var configDesc ocispec.Descriptor
	// Build the model config.
	if err := retry.Do(func() error {
		configDesc, err = builder.BuildConfig(ctx, layers, modelConfig, hooks.NewHooks(
			hooks.WithOnStart(func(name string, size int64, reader io.Reader) io.Reader {
				return pb.Add(internalpb.NormalizePrompt("Building config"), name, size, reader)
			}),
			hooks.WithOnError(func(name string, err error) {
				pb.Abort(name, fmt.Errorf("failed to build config: %w", err))
			}),
			hooks.WithOnComplete(func(name string, desc ocispec.Descriptor) {
				pb.Complete(name, fmt.Sprintf("%s %s", internalpb.NormalizePrompt("Built config"), desc.Digest))
			}),
		))
		return err
	}, append(defaultRetryOpts, retry.Context(ctx))...); err != nil {
		return fmt.Errorf("failed to build model config: %w", err)
	}

	// Build the model manifest.
	if err := retry.Do(func() error {
		_, err = builder.BuildManifest(ctx, layers, configDesc, manifestAnnotation(modelfile), hooks.NewHooks(
			hooks.WithOnStart(func(name string, size int64, reader io.Reader) io.Reader {
				return pb.Add(internalpb.NormalizePrompt("Building manifest"), name, size, reader)
			}),
			hooks.WithOnError(func(name string, err error) {
				pb.Abort(name, fmt.Errorf("failed to build manifest: %w", err))
			}),
			hooks.WithOnComplete(func(name string, desc ocispec.Descriptor) {
				pb.Complete(name, fmt.Sprintf("%s %s", internalpb.NormalizePrompt("Built manifest"), desc.Digest))
			}),
		))
		return err
	}, append(defaultRetryOpts, retry.Context(ctx))...); err != nil {
		return fmt.Errorf("failed to build model manifest: %w", err)
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
func (b *backend) process(ctx context.Context, builder build.Builder, workDir string, pb *internalpb.ProgressBar, cfg *config.Build, processors ...processor.Processor) ([]ocispec.Descriptor, error) {
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
func manifestAnnotation(modelfile modelfile.Modelfile) map[string]string {
	anno := map[string]string{
		annotationModelfile: string(modelfile.Content()),
	}
	return anno
}

// getSourceInfo returns the source information for the build.
func getSourceInfo(workspace string, buildConfig *config.Build) (*source.Info, error) {
	info := &source.Info{
		URL:    buildConfig.SourceURL,
		Commit: buildConfig.SourceRevision,
	}

	// Try to parse the source information if user not specified.
	if info.URL == "" {
		var parser source.Parser

		gitPath := filepath.Join(workspace, ".git")
		if _, err := os.Stat(gitPath); err == nil {
			parser, err = source.NewParser(source.ParserTypeGit)
			if err != nil {
				return nil, err
			}
		}

		zetaPath := filepath.Join(workspace, ".zeta")
		if _, err := os.Stat(zetaPath); err == nil {
			parser, err = source.NewParser(source.ParserTypeZeta)
			if err != nil {
				return nil, err
			}
		}

		// Parse the source information if available.
		if parser != nil {
			parsedInfo, err := parser.Parse(workspace)
			if err != nil {
				return nil, err
			}

			return parsedInfo, nil
		}
	}

	return info, nil
}
