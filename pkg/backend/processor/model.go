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

	"github.com/modelpack/modctl/pkg/backend/build"
	"github.com/modelpack/modctl/pkg/storage"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

const (
	modelProcessorName = "model"
)

// NewModelProcessor creates a new model processor.
func NewModelProcessor(store storage.Storage, mediaType string, patterns []string) Processor {
	return &modelProcessor{
		base: &base{
			name:      modelProcessorName,
			store:     store,
			mediaType: mediaType,
			patterns:  patterns,
		},
	}
}

// modelProcessor is the processor to process the model file.
type modelProcessor struct {
	base *base
}

func (p *modelProcessor) Name() string {
	return modelProcessorName
}

func (p *modelProcessor) Process(ctx context.Context, builder build.Builder, workDir string, opts ...ProcessOption) ([]ocispec.Descriptor, error) {
	return p.base.Process(ctx, builder, workDir, opts...)
}
