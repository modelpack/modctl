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

	"github.com/CloudNativeAI/modctl/pkg/oci/build"
	modelspec "github.com/CloudNativeAI/modctl/pkg/oci/spec"
	"github.com/CloudNativeAI/modctl/pkg/storage"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

// NewLicenseProcessor creates a new LICENSE processor.
func NewLicenseProcessor() Processor {
	return &licenseProcessor{}
}

// licenseProcessor is the processor to process the LICENSE file.
type licenseProcessor struct{}

func (p *licenseProcessor) Identify(_ context.Context, path string, info os.FileInfo) bool {
	return info.Name() == "LICENSE" || info.Name() == "LICENSE.txt"
}

func (p *licenseProcessor) Process(ctx context.Context, store storage.Storage, repo, path string, info os.FileInfo) (ocispec.Descriptor, error) {
	desc, err := build.BuildLayer(ctx, store, repo, path)
	if err != nil {
		return ocispec.Descriptor{}, nil
	}

	// add license annotations.
	desc.Annotations = map[string]string{
		modelspec.AnnotationLicense: "true",
	}

	return desc, nil
}
