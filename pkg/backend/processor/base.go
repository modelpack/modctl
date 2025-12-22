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

	"github.com/avast/retry-go/v4"
	legacymodelspec "github.com/dragonflyoss/model-spec/specs-go/v1"
	modelspec "github.com/modelpack/model-spec/specs-go/v1"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"

	internalpb "github.com/modelpack/modctl/internal/pb"
	"github.com/modelpack/modctl/pkg/backend/build"
	"github.com/modelpack/modctl/pkg/backend/build/hooks"
	"github.com/modelpack/modctl/pkg/storage"
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
	// destDir is the destination dir for the processed content,
	// which is used to store in the layer filepath annotation,
	// it can be empty and by default is relative path to the workDir.
	destDir string
}

// Process implements the Processor interface, which can be reused by other processors.
func (b *base) Process(ctx context.Context, builder build.Builder, workDir string, opts ...ProcessOption) ([]ocispec.Descriptor, error) {
	logrus.Infof("processor: starting %s processing [mediaType: %s, patterns: %v]", b.name, b.mediaType, b.patterns)

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

	logrus.Infof("processor: processing %s files [count: %d]", b.name, len(matchedPaths))

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
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			if err := retry.Do(func() error {
				logrus.Debugf("processor: processing %s file %s", b.name, path)

				var destPath string
				if b.destDir != "" {
					destPath = filepath.Join(b.destDir, filepath.Base(path))
				}

				desc, err := builder.BuildLayer(ctx, b.mediaType, workDir, path, destPath, hooks.NewHooks(
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
					return fmt.Errorf("processor: failed to build layer for %s file %s: %w", b.name, path, err)
				}

				logrus.Debugf("processor: successfully built %s layer for file %s [digest: %s, size: %d]", b.name, path, desc.Digest, desc.Size)
				mu.Lock()
				descriptors = append(descriptors, desc)
				mu.Unlock()

				return nil
			}, append(defaultRetryOpts, retry.Context(ctx))...); err != nil {
				logrus.Error(err)
				// Cancel manually to abort other tasks because if one fails,
				// we should abort all to avoid useless waiting.
				cancel()
				return err
			}

			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		return nil, err
	}

	logrus.Infof("processor: successfully processed %s files [count: %d]", b.name, len(matchedPaths))

	sort.Slice(descriptors, func(i int, j int) bool {
		// Sort by filepath by default.
		var pathI, pathJ string
		if descriptors[i].Annotations != nil {
			if descriptors[i].Annotations[modelspec.AnnotationFilepath] != "" {
				pathI = descriptors[i].Annotations[modelspec.AnnotationFilepath]
			} else {
				pathI = descriptors[i].Annotations[legacymodelspec.AnnotationFilepath]
			}
		}

		if descriptors[j].Annotations != nil {
			if descriptors[j].Annotations[modelspec.AnnotationFilepath] != "" {
				pathJ = descriptors[j].Annotations[modelspec.AnnotationFilepath]
			} else {
				pathJ = descriptors[j].Annotations[legacymodelspec.AnnotationFilepath]
			}
		}

		return pathI < pathJ
	})

	logrus.Debugf("processor: sorted %s layers [layers: %+v]", b.name, descriptors)

	return descriptors, nil
}
