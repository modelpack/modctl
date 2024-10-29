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

package processor

import (
	"context"
	"os"
	"regexp"

	"github.com/CloudNativeAI/modctl/pkg/backend/build"
	modelspec "github.com/CloudNativeAI/modctl/pkg/spec"
	"github.com/CloudNativeAI/modctl/pkg/storage"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

// NewModelConfigProcessor creates a new model config processor.
func NewModelConfigProcessor(configs []string) Processor {
	return &modelConfigProcessor{
		configs: configs,
	}
}

// modelConfigProcessor is the processor to process the model config file.
type modelConfigProcessor struct {
	// configs is the list of regular expressions to match the model config file.
	configs []string
}

func (p *modelConfigProcessor) Identify(_ context.Context, path string, info os.FileInfo) bool {
	for _, config := range p.configs {
		if matched, _ := regexp.MatchString(config, info.Name()); matched {
			return true
		}
	}

	return false
}

func (p *modelConfigProcessor) Process(ctx context.Context, store storage.Storage, repo, path string, info os.FileInfo) (ocispec.Descriptor, error) {
	desc, err := build.BuildLayer(ctx, store, repo, path)
	if err != nil {
		return ocispec.Descriptor{}, nil
	}

	// add config annotations.
	desc.Annotations = map[string]string{
		modelspec.AnnotationConfig: "true",
	}

	return desc, nil
}
