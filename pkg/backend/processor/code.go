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

package processor

import (
	"context"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/sirupsen/logrus"

	"github.com/CloudNativeAI/modctl/pkg/backend/build"
	"github.com/CloudNativeAI/modctl/pkg/storage"
)

const (
	codeProcessorName = "code"
)

// NewCodeProcessor creates a new code processor.
func NewCodeProcessor(store storage.Storage, mediaType string, patterns []string) Processor {
	return &codeProcessor{
		base: &base{
			name:      codeProcessorName,
			store:     store,
			mediaType: mediaType,
			patterns:  patterns,
		},
	}
}

// codeProcessor is the processor to process the code file.
type codeProcessor struct {
	base *base
}

func (p *codeProcessor) Name() string {
	return codeProcessorName
}

func (p *codeProcessor) Process(ctx context.Context, builder build.Builder, workDir string, opts ...ProcessOption) ([]ocispec.Descriptor, error) {
	logrus.Infof("Processing code file, work dir: %s", workDir)
	return p.base.Process(ctx, builder, workDir, opts...)
}
