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
	"errors"
	"testing"

	"github.com/CloudNativeAI/modctl/test/mocks/storage"

	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestTag(t *testing.T) {
	tests := []struct {
		name        string
		source      string
		target      string
		setupMocks  func(*storage.Storage)
		expectedErr string
	}{
		{
			name:   "successful tag",
			source: "localhost:5000/repo:tag1",
			target: "localhost:5000/repo:tag2",
			setupMocks: func(s *storage.Storage) {
				manifest := v1.Manifest{
					Config: v1.Descriptor{
						MediaType: "application/vnd.oci.image.config.v1+json",
						Digest:    "sha256:config",
						Size:      100,
					},
					Layers: []v1.Descriptor{
						{
							MediaType: "application/vnd.oci.image.layer.v1.tar+gzip",
							Digest:    "sha256:layer1",
							Size:      200,
						},
						{
							MediaType: "application/vnd.oci.image.layer.v1.tar+gzip",
							Digest:    "sha256:layer2",
							Size:      300,
						},
					},
				}
				manifestBytes, _ := json.Marshal(manifest)
				s.On("PullManifest", mock.Anything, "localhost:5000/repo", "tag1").
					Return(manifestBytes, "sha256:manifest", nil)

				s.On("MountBlob", mock.Anything, "localhost:5000/repo", "localhost:5000/repo", manifest.Config).
					Return(nil)

				for _, layer := range manifest.Layers {
					s.On("MountBlob", mock.Anything, "localhost:5000/repo", "localhost:5000/repo", layer).
						Return(nil)
				}

				s.On("PushManifest", mock.Anything, "localhost:5000/repo", "tag2", manifestBytes).
					Return("sha256:manifest", nil)
			},
			expectedErr: "",
		},
		{
			name:   "invalid source reference",
			source: "invalid-reference",
			target: "localhost:5000/repo:tag2",
			setupMocks: func(s *storage.Storage) {
				// No mocks needed as we expect to fail before hitting the storage
			},
			expectedErr: "failed to parse source",
		},
		{
			name:   "invalid target reference",
			source: "localhost:5000/repo:tag1",
			target: "invalid-reference",
			setupMocks: func(s *storage.Storage) {
				// No mocks needed as we expect to fail before hitting the storage
			},
			expectedErr: "failed to parse target",
		},
		{
			name:   "pull manifest error",
			source: "localhost:5000/repo:tag1",
			target: "localhost:5000/repo:tag2",
			setupMocks: func(s *storage.Storage) {
				s.On("PullManifest", mock.Anything, "localhost:5000/repo", "tag1").
					Return([]byte{}, "", errors.New("manifest not found"))
			},
			expectedErr: "failed to pull manifest",
		},
		{
			name:   "mount blob error",
			source: "localhost:5000/repo:tag1",
			target: "localhost:5000/repo:tag2",
			setupMocks: func(s *storage.Storage) {
				manifest := v1.Manifest{
					Config: v1.Descriptor{
						MediaType: "application/vnd.oci.image.config.v1+json",
						Digest:    "sha256:config",
						Size:      100,
					},
					Layers: []v1.Descriptor{
						{
							MediaType: "application/vnd.oci.image.layer.v1.tar+gzip",
							Digest:    "sha256:layer1",
							Size:      200,
						},
					},
				}
				manifestBytes, _ := json.Marshal(manifest)

				s.On("PullManifest", mock.Anything, "localhost:5000/repo", "tag1").
					Return(manifestBytes, "sha256:manifest", nil)

				s.On("MountBlob", mock.Anything, "localhost:5000/repo", "localhost:5000/repo", manifest.Config).
					Return(errors.New("mount blob failed"))
			},
			expectedErr: "failed to mount blob",
		},
		{
			name:   "push manifest error",
			source: "localhost:5000/repo:tag1",
			target: "localhost:5000/repo:tag2",
			setupMocks: func(s *storage.Storage) {
				manifest := v1.Manifest{
					Config: v1.Descriptor{
						MediaType: "application/vnd.oci.image.config.v1+json",
						Digest:    "sha256:config",
						Size:      100,
					},
					Layers: []v1.Descriptor{},
				}
				manifestBytes, _ := json.Marshal(manifest)
				s.On("PullManifest", mock.Anything, "localhost:5000/repo", "tag1").
					Return(manifestBytes, "sha256:manifest", nil)

				s.On("MountBlob", mock.Anything, "localhost:5000/repo", "localhost:5000/repo", manifest.Config).
					Return(nil)

				s.On("PushManifest", mock.Anything, "localhost:5000/repo", "tag2", manifestBytes).
					Return("", errors.New("push manifest failed"))
			},
			expectedErr: "failed to push manifest",
		},
		{
			name:   "invalid manifest json",
			source: "localhost:5000/repo:tag1",
			target: "localhost:5000/repo:tag2",
			setupMocks: func(s *storage.Storage) {
				// Return invalid JSON as manifest
				s.On("PullManifest", mock.Anything, "localhost:5000/repo", "tag1").
					Return([]byte{123}, "sha256:invalid", nil)
			},
			expectedErr: "failed to unmarshal manifest",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStorage := storage.NewStorage(t)
			tt.setupMocks(mockStorage)

			b := &backend{
				store: mockStorage,
			}

			err := b.Tag(context.Background(), tt.source, tt.target)
			if tt.expectedErr != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
