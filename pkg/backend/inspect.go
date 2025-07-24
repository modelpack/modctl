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

	legacymodelspec "github.com/dragonflyoss/model-spec/specs-go/v1"
	modelspec "github.com/modelpack/model-spec/specs-go/v1"
	godigest "github.com/opencontainers/go-digest"
	"github.com/sirupsen/logrus"

	"github.com/modelpack/modctl/pkg/config"
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
	// MediaType is the media type of the model artifact layer.
	MediaType string `json:"MediaType"`
	// Digest is the digest of the model artifact layer.
	Digest string `json:"Digest"`
	// Size is the size of the model artifact layer.
	Size int64 `json:"Size"`
	// Filepath is the filepath of the model artifact layer.
	Filepath string `json:"Filepath"`
}

// Inspect inspects the target from the storage.
func (b *backend) Inspect(ctx context.Context, target string, cfg *config.Inspect) (any, error) {
	logrus.Infof("inspect: starting inspect operation for target %s [config: %+v]", target, cfg)
	_, err := ParseReference(target)
	if err != nil {
		return nil, fmt.Errorf("failed to parse target: %w", err)
	}

	manifest, err := b.getManifest(ctx, target, cfg.Remote, cfg.PlainHTTP, cfg.Insecure)
	if err != nil {
		return nil, fmt.Errorf("failed to get manifest: %w", err)
	}

	manifestRaw, err := json.Marshal(manifest)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal manifest: %w", err)
	}

	logrus.Debugf("inspect: loaded manifest for target %s [manifest: %s]", target, string(manifestRaw))

	config, err := b.getModelConfig(ctx, target, manifest.Config, cfg.Remote, cfg.PlainHTTP, cfg.Insecure)
	if err != nil {
		return nil, fmt.Errorf("failed to get config: %w", err)
	}

	logrus.Debugf("inspect: loaded model config for target %s [family: %s, name: %s]", target, config.Descriptor.Family, config.Descriptor.Name)

	if cfg.Config {
		return config, nil
	}

	inspectedModelArtifact := &InspectedModelArtifact{
		ID:           manifest.Config.Digest.String(),
		Digest:       godigest.FromBytes(manifestRaw).String(),
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
		filepath := layer.Annotations[modelspec.AnnotationFilepath]
		if filepath == "" {
			filepath = layer.Annotations[legacymodelspec.AnnotationFilepath]
		}

		inspectedModelArtifact.Layers = append(inspectedModelArtifact.Layers, InspectedModelArtifactLayer{
			MediaType: layer.MediaType,
			Digest:    layer.Digest.String(),
			Size:      layer.Size,
			Filepath:  filepath,
		})
	}

	logrus.Infof("inspect: successfully inspected target %s", target)
	return inspectedModelArtifact, nil
}
