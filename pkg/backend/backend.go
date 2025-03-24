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

package backend

import (
	"context"

	"github.com/CloudNativeAI/modctl/pkg/config"
	"github.com/CloudNativeAI/modctl/pkg/storage"
)

// Backend is the interface to represent the backend.
type Backend interface {
	// Login logs into a registry.
	Login(ctx context.Context, registry, username, password string, cfg *config.Login) error

	// Logout logs out from a registry.
	Logout(ctx context.Context, registry string) error

	// Build builds the user materials into the OCI image which follows the Model Spec.
	Build(ctx context.Context, modelfilePath, workDir, target string, cfg *config.Build) error

	// Pull pulls an artifact from a registry.
	Pull(ctx context.Context, target string, cfg *config.Pull) error

	// Push pushes the image to the registry.
	Push(ctx context.Context, target string, cfg *config.Push) error

	// List lists all the model artifacts.
	List(ctx context.Context) ([]*ModelArtifact, error)

	// Remove deletes the model artifact.
	Remove(ctx context.Context, target string) (string, error)

	// Prune prunes the unused blobs and clean up the storage.
	Prune(ctx context.Context, dryRun, removeUntagged bool) error

	// Inspect inspects the model artifact.
	Inspect(ctx context.Context, target string) (*InspectedModelArtifact, error)

	// Extract extracts the model artifact.
	Extract(ctx context.Context, target string, cfg *config.Extract) error

	// Tag creates a new tag that refers to the source model artifact.
	Tag(ctx context.Context, source, target string) error

	// Nydusify converts the model artifact to nydus format.
	Nydusify(ctx context.Context, target string) (string, error)
}

// backend is the implementation of Backend.
type backend struct {
	store storage.Storage
}

// New creates a new backend.
func New(storageDir string) (Backend, error) {
	store, err := storage.New("", storageDir)
	if err != nil {
		return nil, err
	}

	return &backend{
		store: store,
	}, nil
}
