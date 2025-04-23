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

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/sirupsen/logrus"
)

// Tag creates a new tag that refers to the source model artifact.
func (b *backend) Tag(ctx context.Context, source, target string) error {
	logrus.Infof("Tagging model artifact %s as %s", source, target)

	srcRef, err := ParseReference(source)
	if err != nil {
		logrus.Errorf("failed to parse source: %v", err)
		return fmt.Errorf("failed to parse source: %w", err)
	}

	targetRef, err := ParseReference(target)
	if err != nil {
		logrus.Errorf("failed to parse target: %v", err)
		return fmt.Errorf("failed to parse target: %w", err)
	}

	manifestRaw, _, err := b.store.PullManifest(ctx, srcRef.Repository(), srcRef.Tag())
	if err != nil {
		logrus.Errorf("failed to pull manifest: %v", err)
		return fmt.Errorf("failed to pull manifest: %w", err)
	}

	var manifest ocispec.Manifest
	if err := json.Unmarshal(manifestRaw, &manifest); err != nil {
		logrus.Errorf("failed to unmarshal manifest: %v", err)
		return fmt.Errorf("failed to unmarshal manifest: %w", err)
	}

	// mount the blob from source.
	layers := []ocispec.Descriptor{manifest.Config}
	for _, layer := range manifest.Layers {
		layers = append(layers, layer)
	}

	for _, layer := range layers {
		if err := b.store.MountBlob(ctx, srcRef.Repository(), targetRef.Repository(), layer); err != nil {
			logrus.Errorf("failed to mount blob %s: %v", layer.Digest.String(), err)
			return fmt.Errorf("failed to mount blob %s: %w", layer.Digest.String(), err)
		}
	}

	if _, err := b.store.PushManifest(ctx, targetRef.Repository(), targetRef.Tag(), manifestRaw); err != nil {
		logrus.Errorf("failed to push manifest: %v", err)
		return fmt.Errorf("failed to push manifest: %w", err)
	}

	logrus.Infof("Tagged model artifact %s as %s successfully", source, target)
	return nil
}
