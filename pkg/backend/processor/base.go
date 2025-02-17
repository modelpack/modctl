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
	"fmt"
	"path/filepath"

	"github.com/CloudNativeAI/modctl/pkg/backend/build"
	"github.com/CloudNativeAI/modctl/pkg/storage"

	humanize "github.com/dustin/go-humanize"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

type base struct {
	// store is the underlying storage backend.
	store storage.Storage
	// mediaType is the media type of the processed content.
	mediaType string
	// patterns is the list of patterns to match.
	patterns []string
}

// Process implements the Processor interface, which can be reused by other processors.
func (b *base) Process(ctx context.Context, workDir, repo string) ([]ocispec.Descriptor, error) {
	absWorkDir, err := filepath.Abs(workDir)
	if err != nil {
		return nil, err
	}

	var matchedPaths []string
	for _, pattern := range b.patterns {
		matches, err := filepath.Glob(filepath.Join(absWorkDir, pattern))
		if err != nil {
			return nil, err
		}

		matchedPaths = append(matchedPaths, matches...)
	}

	var descriptors []ocispec.Descriptor
	for _, path := range matchedPaths {
		desc, err := build.BuildLayer(ctx, b.store, b.mediaType, workDir, repo, path)
		if err != nil {
			return nil, err
		}

		fmt.Printf("%-15s => %s (%s)\n", "Built blob", desc.Digest, humanize.IBytes(uint64(desc.Size)))
		descriptors = append(descriptors, desc)
	}

	return descriptors, nil
}
