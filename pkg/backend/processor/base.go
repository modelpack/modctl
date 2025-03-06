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
	"sort"
	"sync"
	"sync/atomic"

	"github.com/CloudNativeAI/modctl/pkg/backend/build"
	"github.com/CloudNativeAI/modctl/pkg/storage"

	modelspec "github.com/CloudNativeAI/model-spec/specs-go/v1"
	"github.com/chelnak/ysmrr"
	humanize "github.com/dustin/go-humanize"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"golang.org/x/sync/errgroup"
)

type base struct {
	// name is the name of the processor.
	name string
	// store is the underlying storage backend.
	store storage.Storage
	// mediaType is the media type of the processed content.
	mediaType string
	// patterns is the list of patterns to match.
	patterns []string
}

// Process implements the Processor interface, which can be reused by other processors.
func (b *base) Process(ctx context.Context, builder build.Builder, workDir string, opts ...Option) ([]ocispec.Descriptor, error) {
	baseOpts := &options{}
	for _, opt := range opts {
		opt(baseOpts)
	}

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

	sort.Strings(matchedPaths)

	var (
		idx         atomic.Int64
		mu          sync.Mutex
		eg          errgroup.Group
		descriptors []ocispec.Descriptor
	)

	// Set default concurrency limit to 1 if not specified.
	if baseOpts.concurrency > 0 {
		eg.SetLimit(baseOpts.concurrency)
	} else {
		eg.SetLimit(1)
	}

	total := int64(len(matchedPaths))
	sm := ysmrr.NewSpinnerManager()
	sm.Start()

	for _, path := range matchedPaths {
		eg.Go(func() error {
			relPath, err := filepath.Rel(absWorkDir, path)
			if err != nil {
				return err
			}

			blobMsg := fmt.Sprintf("blob [%s %d/%d]", b.name, idx.Add(1), total)
			sp := sm.AddSpinner(fmt.Sprintf("Building %s => %s", blobMsg, relPath))

			desc, err := builder.BuildLayer(ctx, b.mediaType, workDir, path)
			if err != nil {
				sp.ErrorWithMessagef("Failed to build blob %s: %v", relPath, err)
				return err
			}

			sp.CompleteWithMessagef("%s => %s (%s)", fmt.Sprintf("Built %s", blobMsg), desc.Digest, humanize.IBytes(uint64(desc.Size)))

			mu.Lock()
			descriptors = append(descriptors, desc)
			mu.Unlock()

			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		return nil, err
	}

	sm.Stop()

	sort.Slice(descriptors, func(i int, j int) bool {
		// Sort by filepath by default.
		var pathI, pathJ string
		if descriptors[i].Annotations != nil {
			pathI = descriptors[i].Annotations[modelspec.AnnotationFilepath]
		}

		if descriptors[j].Annotations != nil {
			pathJ = descriptors[j].Annotations[modelspec.AnnotationFilepath]
		}

		return pathI < pathJ
	})

	return descriptors, nil
}
