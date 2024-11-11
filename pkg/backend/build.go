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
	"os"
	"path/filepath"
	"time"

	"github.com/CloudNativeAI/modctl/pkg/backend/build"
	"github.com/CloudNativeAI/modctl/pkg/backend/processor"
	"github.com/CloudNativeAI/modctl/pkg/modelfile"
	modelspec "github.com/CloudNativeAI/modctl/pkg/spec"

	humanize "github.com/dustin/go-humanize"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

// Build builds the user materials into the OCI image which follows the Model Spec.
func (b *backend) Build(ctx context.Context, modelfilePath, workDir, target string) error {
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
	layers := []ocispec.Descriptor{}
	layerDescs, err := b.process(ctx, workDir, repo, getProcessors(modelfile)...)
	if err != nil {
		return fmt.Errorf("failed to process files: %w", err)
	}

	layers = append(layers, layerDescs...)

	// build the image config.
	configDesc, err := build.BuildConfig(ctx, b.store, repo)
	if err != nil {
		return fmt.Errorf("faile to build image config: %w", err)
	}

	fmt.Printf("%-15s => %s (%s)\n", "Built config", configDesc.Digest, humanize.IBytes(uint64(configDesc.Size)))

	// build the image manifest.
	manifestDesc, err := build.BuildManifest(ctx, b.store, repo, tag, layers, configDesc, manifestAnnotation(modelfile))
	if err != nil {
		return fmt.Errorf("faile to build image manifest: %w", err)
	}

	fmt.Printf("%-15s => %s (%s)\n", "Built manifest", manifestDesc.Digest, humanize.IBytes(uint64(manifestDesc.Size)))
	return nil
}

func defaultProcessors() []processor.Processor {
	return []processor.Processor{
		processor.NewLicenseProcessor(),
		processor.NewReadmeProcessor(),
	}
}

func getProcessors(modelfile modelfile.Modelfile) []processor.Processor {
	processors := defaultProcessors()

	if configs := modelfile.GetConfigs(); len(configs) > 0 {
		processors = append(processors, processor.NewModelConfigProcessor(configs))
	}

	if models := modelfile.GetModels(); len(models) > 0 {
		processors = append(processors, processor.NewModelProcessor(models))
	}

	return processors
}

// process walks the user work directory and process the identified files.
func (b *backend) process(ctx context.Context, workDir string, repo string, processors ...processor.Processor) ([]ocispec.Descriptor, error) {
	layers := []ocispec.Descriptor{}
	// walk the user work directory and handle the default identified files.
	if err := filepath.Walk(workDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("failed to walk directory: %w", err)
		}
		// skip directories.
		if info.IsDir() {
			return nil
		}
		// get absolute path.
		path, err = filepath.Abs(path)
		if err != nil {
			return fmt.Errorf("failed to get absolute path: %w", err)
		}
		// fan-in file processors.
		for _, p := range processors {
			// process the file if it can be recognized.
			if p.Identify(ctx, path, info) {
				desc, err := p.Process(ctx, b.store, repo, path, workDir)
				if err != nil {
					return fmt.Errorf("failed to process file: %w", err)
				}

				fmt.Printf("%-15s => %s (%s)\n", "Built blob", desc.Digest, humanize.IBytes(uint64(desc.Size)))
				layers = append(layers, desc)
			}
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return layers, nil
}

// manifestAnnotation returns the annotations for the manifest.
func manifestAnnotation(modelfile modelfile.Modelfile) map[string]string {
	anno := map[string]string{
		modelspec.AnnotationCreated: time.Now().Format(time.RFC3339),
	}

	if arch := modelfile.GetArch(); arch != "" {
		anno[modelspec.AnnotationArchitecture] = arch
	}

	if family := modelfile.GetFamily(); family != "" {
		anno[modelspec.AnnotationFamily] = family
	}

	if name := modelfile.GetName(); name != "" {
		anno[modelspec.AnnotationName] = name
	}

	if format := modelfile.GetFormat(); format != "" {
		anno[modelspec.AnnotationFormat] = format
	}

	if paramsize := modelfile.GetParamsize(); paramsize != "" {
		anno[modelspec.AnnotationParamSize] = paramsize
	}

	if precision := modelfile.GetPrecision(); precision != "" {
		anno[modelspec.AnnotationPrecision] = precision
	}

	if quantization := modelfile.GetQuantization(); quantization != "" {
		anno[modelspec.AnnotationQuantization] = quantization
	}

	return anno
}
