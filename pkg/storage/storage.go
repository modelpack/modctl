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

package storage

import (
	"context"
	"io"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

// Option is the option wrapper for modifying the storage options.
type Option func(*Options)

// Options is the options for the storage.
type Options struct {
	// RootDir is the root directory of the storage.
	RootDir string
}

// Storage is an interface for storage which wraps the storage operations.
type Storage interface {
	// PullManifest pulls the manifest from the storage.
	PullManifest(ctx context.Context, repo, reference string) ([]byte, string, error)
	// PushManifest pushes the manifest to the storage.
	PushManifest(ctx context.Context, repo, reference string, body []byte) (string, error)
	// StatManifest stats the manifest in the storage.
	StatManifest(ctx context.Context, repo, digest string) (bool, error)
	// DeleteManifest deletes the manifest from the storage.
	DeleteManifest(ctx context.Context, repo, reference string) error
	// PullBlob pulls the blob from the storage.
	PullBlob(ctx context.Context, repo, digest string) (io.ReadCloser, error)
	// PushBlob pushes the blob to the storage.
	PushBlob(ctx context.Context, repo string, body io.Reader, desc ocispec.Descriptor) (string, int64, error)
	// MountBlob mounts the blob to the storage.
	MountBlob(ctx context.Context, fromRepo, toRepo string, desc ocispec.Descriptor) error
	// StatBlob stats the blob in the storage.
	StatBlob(ctx context.Context, repo, digest string) (bool, error)
	// ListRepositories lists all the repositories in the storage.
	ListRepositories(ctx context.Context) ([]string, error)
	// ListTags lists all the tags in the repository.
	ListTags(ctx context.Context, repo string) ([]string, error)
	// PerformGC performs the garbage collection in the storage to free up the space.
	PerformGC(ctx context.Context, dryRun, removeUntagged bool) error
}

// WithRootDir sets the root directory of the storage.
func WithRootDir(rootDir string) Option {
	return func(o *Options) {
		o.RootDir = rootDir
	}
}
