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
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	internalpb "github.com/CloudNativeAI/modctl/internal/pb"
	"github.com/CloudNativeAI/modctl/pkg/backend/build"
	"github.com/CloudNativeAI/modctl/pkg/backend/build/hooks"
	"github.com/CloudNativeAI/modctl/pkg/storage"

	modelspec "github.com/CloudNativeAI/model-spec/specs-go/v1"
	"github.com/avast/retry-go/v4"
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
func (b *base) Process(ctx context.Context, builder build.Builder, workDir string, opts ...ProcessOption) ([]ocispec.Descriptor, error) {
	processOpts := &processOptions{}
	for _, opt := range opts {
		opt(processOpts)
	}

	absWorkDir, err := filepath.Abs(workDir)
	if err != nil {
		return nil, err
	}

	var matchedPaths []string
	for _, pattern := range b.patterns {
		// Check if the pattern is a specific file path (no wildcards)
		if !strings.ContainsAny(pattern, "*?[]") {
			// For specific file paths, check if the file exists
			var fullPath string
			if filepath.IsAbs(pattern) {
				fullPath = pattern
			} else {
				fullPath = filepath.Join(absWorkDir, pattern)
			}

			if _, err := os.Stat(fullPath); err != nil {
				if os.IsNotExist(err) {
					return nil, fmt.Errorf("file specified in Modelfile does not exist: %s", pattern)
				}
				return nil, fmt.Errorf("failed to check file: %s, error: %w", pattern, err)
			}

			matchedPaths = append(matchedPaths, fullPath)
		} else {
			// For patterns with wildcards, use glob matching
			matches, err := filepath.Glob(filepath.Join(absWorkDir, pattern))
			if err != nil {
				return nil, err
			}

			matchedPaths = append(matchedPaths, matches...)
		}
	}

	sort.Strings(matchedPaths)

	var (
		mu          sync.Mutex
		eg          *errgroup.Group
		descriptors []ocispec.Descriptor
	)

	// Initialize errgroup with a context can be canceled.
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	eg, ctx = errgroup.WithContext(ctx)

	// Set default concurrency limit to 1 if not specified.
	if processOpts.concurrency > 0 {
		eg.SetLimit(processOpts.concurrency)
	} else {
		eg.SetLimit(1)
	}

	// Initialize progress tracker if not provided.
	tracker := processOpts.progressTracker
	if tracker == nil {
		tracker = internalpb.NewProgressBar()
		tracker.Start()
		defer tracker.Stop()
	}

	for _, path := range matchedPaths {
		if ctx.Err() != nil {
			break
		}

		eg.Go(func() error {
			return retry.Do(func() error {
				desc, err := builder.BuildLayer(ctx, b.mediaType, workDir, path, hooks.NewHooks(
					hooks.WithOnStart(func(name string, size int64, reader io.Reader) io.Reader {
						return tracker.Add(internalpb.NormalizePrompt("Building layer"), name, size, reader)
					}),
					hooks.WithOnError(func(name string, err error) {
						tracker.Abort(name, fmt.Errorf("failed to build layer: %w", err))
					}),
					hooks.WithOnComplete(func(name string, desc ocispec.Descriptor) {
						tracker.Complete(name, fmt.Sprintf("%s %s", internalpb.NormalizePrompt("Built layer"), desc.Digest))
					}),
				))
				if err != nil {
					cancel()
					return err
				}

				mu.Lock()
				descriptors = append(descriptors, desc)
				mu.Unlock()

				return nil
			}, append(defaultRetryOpts, retry.Context(ctx))...)
		})
	}

	if err := eg.Wait(); err != nil {
		return nil, err
	}

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
