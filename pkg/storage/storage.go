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
	// GetIndex gets the content of index.json in repo from the storage.
	GetIndex(ctx context.Context, repo string) ([]byte, error)
	// PullManifest pulls the manifest from the storage.
	PullManifest(ctx context.Context, repo, reference string) ([]byte, string, error)
	// PushManifest pushes the manifest to the storage.
	PushManifest(ctx context.Context, repo, reference string, body []byte) (string, error)
	// DeleteManifest deletes the manifest from the storage.
	DeleteManifest(ctx context.Context, repo, reference string) error
	// PushBlob pushes the blob to the storage.
	PushBlob(ctx context.Context, repo string, body io.Reader) (string, int64, error)
	// StatBlob stats the blob in the storage.
	StatBlob(ctx context.Context, repo, digest string) (bool, error)
	// DeleteBlob deletes the blob from the storage.
	DeleteBlob(ctx context.Context, repo, digest string) error
	// ListBlobs lists all the blobs in the repository.
	ListBlobs(ctx context.Context, repo string) ([]string, error)
	// ListRepositories lists all the repositories in the storage.
	ListRepositories(ctx context.Context) ([]string, error)
	// CleanupRepo cleans up the repository in the storage.
	CleanupRepo(ctx context.Context, repo string, blobs []string, removeRepo bool) (int, error)
	// ListTags lists all the tags in the repository.
	ListTags(ctx context.Context, repo string) ([]string, error)
}

// WithRootDir sets the root directory of the storage.
func WithRootDir(rootDir string) Option {
	return func(o *Options) {
		o.RootDir = rootDir
	}
}
