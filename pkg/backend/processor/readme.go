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

	"github.com/CloudNativeAI/modctl/pkg/backend/build"
	modelspec "github.com/CloudNativeAI/modctl/pkg/spec"
	"github.com/CloudNativeAI/modctl/pkg/storage"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

// NewReadmeProcessor creates a new README processor.
func NewReadmeProcessor() Processor {
	return &readmeProcessor{}
}

// readmeProcessor is the processor to process the README file.
type readmeProcessor struct{}

func (p *readmeProcessor) Name() string {
	return "readme"
}

func (p *readmeProcessor) Identify(_ context.Context, path string, info os.FileInfo) bool {
	return info.Name() == "README.md" || info.Name() == "README"
}

func (p *readmeProcessor) Process(ctx context.Context, store storage.Storage, repo, path, workDir string) (ocispec.Descriptor, error) {
	desc, err := build.BuildLayer(ctx, store, repo, path, workDir)
	if err != nil {
		return ocispec.Descriptor{}, nil
	}

	// add readme annotations.
	if desc.Annotations == nil {
		desc.Annotations = map[string]string{}
	}

	desc.Annotations[modelspec.AnnotationReadme] = "true"
	return desc, nil
}
