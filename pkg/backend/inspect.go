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
	"encoding/json"
	"fmt"
	"time"

	modelspec "github.com/CloudNativeAI/model-spec/specs-go/v1"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/sirupsen/logrus"
)

// InspectedModelArtifact is the data structure for model artifact that has been inspected.
type InspectedModelArtifact struct {
	// ID is the image id of the model artifact.
	ID string `json:"Id"`
	// Digest is the digest of the model artifact.
	Digest string `json:"Digest"`
	// Architecture is the architecture of the model.
	Architecture string `json:"Architecture"`
	// CreatedAt is the creation time of the model artifact.
	CreatedAt string `json:"CreatedAt"`
	// Family is the family of the model.
	Family string `json:"Family"`
	// Format is the format of the model.
	Format string `json:"Format"`
	// Name is the name of the model.
	Name string `json:"Name"`
	// ParamSize is the param size of the model.
	ParamSize string `json:"ParamSize"`
	// Precision is the precision of the model.
	Precision string `json:"Precision"`
	// Quantization is the quantization of the model.
	Quantization string `json:"Quantization"`
	// Layers is the layers of the model artifact.
	Layers []InspectedModelArtifactLayer `json:"Layers"`
}

// InspectedModelArtifactLayer is the data structure for model artifact layer that has been inspected.
type InspectedModelArtifactLayer struct {
	// Digest is the digest of the model artifact layer.
	Digest string `json:"Digest"`
	// Size is the size of the model artifact layer.
	Size int64 `json:"Size"`
	// Filepath is the filepath of the model artifact layer.
	Filepath string `json:"Filepath"`
}

// Inspect inspects the target from the storage.
func (b *backend) Inspect(ctx context.Context, target string) (*InspectedModelArtifact, error) {
	logrus.Infof("Inspecting model artifact %s", target)

	ref, err := ParseReference(target)
	if err != nil {
		logrus.Errorf("failed to parse target: %v", err)
		return nil, fmt.Errorf("failed to parse target: %w", err)
	}

	repo, tag := ref.Repository(), ref.Tag()
	manifestRaw, digest, err := b.store.PullManifest(ctx, repo, tag)
	if err != nil {
		logrus.Errorf("failed to get manifest: %v", err)
		return nil, fmt.Errorf("failed to get manifest: %w", err)
	}

	var manifest ocispec.Manifest
	if err := json.Unmarshal(manifestRaw, &manifest); err != nil {
		logrus.Errorf("failed to unmarshal manifest: %v", err)
		return nil, fmt.Errorf("failed to unmarshal manifest: %w", err)
	}

	// fetch and parse the model config.
	configReader, err := b.store.PullBlob(ctx, repo, manifest.Config.Digest.String())
	if err != nil {
		logrus.Errorf("failed to pull config: %v", err)
		return nil, fmt.Errorf("failed to pull config: %w", err)
	}

	defer configReader.Close()
	var config modelspec.Model
	if err := json.NewDecoder(configReader).Decode(&config); err != nil {
		logrus.Errorf("failed to decode config: %v", err)
		return nil, fmt.Errorf("failed to decode config: %w", err)
	}

	inspectedModelArtifact := &InspectedModelArtifact{
		ID:           manifest.Config.Digest.String(),
		Digest:       digest,
		Architecture: config.Config.Architecture,
		Family:       config.Descriptor.Family,
		Format:       config.Config.Format,
		Name:         config.Descriptor.Name,
		ParamSize:    config.Config.ParamSize,
		Precision:    config.Config.Precision,
		Quantization: config.Config.Quantization,
	}

	if config.Descriptor.CreatedAt != nil {
		inspectedModelArtifact.CreatedAt = config.Descriptor.CreatedAt.Format(time.RFC3339)
	}

	for _, layer := range manifest.Layers {
		inspectedModelArtifact.Layers = append(inspectedModelArtifact.Layers, InspectedModelArtifactLayer{
			Digest:   layer.Digest.String(),
			Size:     layer.Size,
			Filepath: layer.Annotations[modelspec.AnnotationFilepath],
		})
	}

	logrus.Infof("Inspected model artifact %s successfully: %#v", target, inspectedModelArtifact)
	return inspectedModelArtifact, nil
}
