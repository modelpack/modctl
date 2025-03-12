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

	"github.com/CloudNativeAI/modctl/pkg/backend/build"
	"github.com/CloudNativeAI/modctl/pkg/storage"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

const (
	modelConfigProcessorName = "config"
)

// NewModelConfigProcessor creates a new model config processor.
func NewModelConfigProcessor(store storage.Storage, mediaType string, patterns []string) Processor {
	return &modelConfigProcessor{
		base: &base{
			name:      modelConfigProcessorName,
			store:     store,
			mediaType: mediaType,
			patterns:  patterns,
		},
	}
}

// modelConfigProcessor is the processor to process the model config file.
type modelConfigProcessor struct {
	base *base
}

func (p *modelConfigProcessor) Name() string {
	return modelConfigProcessorName
}

func (p *modelConfigProcessor) Process(ctx context.Context, builder build.Builder, workDir string, opts ...ProcessOption) ([]ocispec.Descriptor, error) {
	return p.base.Process(ctx, builder, workDir, opts...)
}
