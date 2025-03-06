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
	docProcessorName = "doc"
)

// NewDocProcessor creates a new doc processor.
func NewDocProcessor(store storage.Storage, mediaType string, patterns []string) Processor {
	return &docProcessor{
		base: &base{
			name:      docProcessorName,
			store:     store,
			mediaType: mediaType,
			patterns:  patterns,
		},
	}
}

// docProcessor is the processor to process the doc file.
type docProcessor struct {
	base *base
}

func (p *docProcessor) Name() string {
	return docProcessorName
}

func (p *docProcessor) Process(ctx context.Context, builder build.Builder, workDir string, opts ...Option) ([]ocispec.Descriptor, error) {
	return p.base.Process(ctx, builder, workDir, opts...)
}
