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

package backend

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	modelspec "github.com/CloudNativeAI/model-spec/specs-go/v1"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/stretchr/testify/assert"

	"github.com/CloudNativeAI/modctl/pkg/config"
	mockstore "github.com/CloudNativeAI/modctl/test/mocks/storage"
)

func TestBackendGetManifest(t *testing.T) {
	ctx := context.Background()
	mockStore := &mockstore.Storage{}
	b := &backend{
		store: mockStore,
	}

	t.Run("LocalStorage_Success", func(t *testing.T) {
		manifest := ocispec.Manifest{Layers: []ocispec.Descriptor{{Digest: "sha256:abc"}}}
		manifestBytes, _ := json.Marshal(manifest)
		mockStore.On("PullManifest", ctx, "localhost/repo", "tag").Return(manifestBytes, "", nil)

		cfg := &config.Attach{OutputRemote: false}
		result, err := b.getManifest(ctx, "localhost/repo:tag", cfg)
		assert.NoError(t, err)
		assert.Equal(t, manifest.Layers, result.Layers)
		mockStore.AssertExpectations(t)
	})

	t.Run("InvalidReference", func(t *testing.T) {
		cfg := &config.Attach{OutputRemote: false}
		_, err := b.getManifest(ctx, "invalid", cfg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse source reference")
	})
}

func TestGetProcessor(t *testing.T) {
	b := &backend{store: &mockstore.Storage{}}
	tests := []struct {
		filepath string
		wantType string
	}{
		{"config.yaml", "modelConfigProcessor"},
		{"model.pth", "modelProcessor"},
		{"script.py", "codeProcessor"},
		{"doc.pdf", "docProcessor"},
		{"unknown.xyz", ""},
	}

	for _, tt := range tests {
		t.Run(tt.filepath, func(t *testing.T) {
			proc := b.getProcessor(tt.filepath)
			if tt.wantType == "" {
				assert.Nil(t, proc)
			} else {
				assert.NotNil(t, proc)
				assert.Contains(t, fmt.Sprintf("%T", proc), tt.wantType)
			}
		})
	}
}

func TestSortLayers(t *testing.T) {
	testCases := []struct {
		name     string
		input    []ocispec.Descriptor
		expected []ocispec.Descriptor
	}{
		{
			name: "mixed media types",
			input: []ocispec.Descriptor{
				{
					MediaType: modelspec.MediaTypeModelDoc,
					Annotations: map[string]string{
						modelspec.AnnotationFilepath: "doc.md",
					},
				},
				{
					MediaType: modelspec.MediaTypeModelWeight,
					Annotations: map[string]string{
						modelspec.AnnotationFilepath: "weight.bin",
					},
				},
				{
					MediaType: modelspec.MediaTypeModelCode,
					Annotations: map[string]string{
						modelspec.AnnotationFilepath: "code.py",
					},
				},
				{
					MediaType: modelspec.MediaTypeModelWeightConfig,
					Annotations: map[string]string{
						modelspec.AnnotationFilepath: "config.json",
					},
				},
			},
			expected: []ocispec.Descriptor{
				{
					MediaType: modelspec.MediaTypeModelWeightConfig,
					Annotations: map[string]string{
						modelspec.AnnotationFilepath: "config.json",
					},
				},
				{
					MediaType: modelspec.MediaTypeModelWeight,
					Annotations: map[string]string{
						modelspec.AnnotationFilepath: "weight.bin",
					},
				},
				{
					MediaType: modelspec.MediaTypeModelCode,
					Annotations: map[string]string{
						modelspec.AnnotationFilepath: "code.py",
					},
				},
				{
					MediaType: modelspec.MediaTypeModelDoc,
					Annotations: map[string]string{
						modelspec.AnnotationFilepath: "doc.md",
					},
				},
			},
		},
		{
			name: "same media type, different filepaths",
			input: []ocispec.Descriptor{
				{
					MediaType: modelspec.MediaTypeModelCode,
					Annotations: map[string]string{
						modelspec.AnnotationFilepath: "z_code.py",
					},
				},
				{
					MediaType: modelspec.MediaTypeModelCode,
					Annotations: map[string]string{
						modelspec.AnnotationFilepath: "a_code.py",
					},
				},
				{
					MediaType: modelspec.MediaTypeModelCode,
					Annotations: map[string]string{
						modelspec.AnnotationFilepath: "m_code.py",
					},
				},
			},
			expected: []ocispec.Descriptor{
				{
					MediaType: modelspec.MediaTypeModelCode,
					Annotations: map[string]string{
						modelspec.AnnotationFilepath: "a_code.py",
					},
				},
				{
					MediaType: modelspec.MediaTypeModelCode,
					Annotations: map[string]string{
						modelspec.AnnotationFilepath: "m_code.py",
					},
				},
				{
					MediaType: modelspec.MediaTypeModelCode,
					Annotations: map[string]string{
						modelspec.AnnotationFilepath: "z_code.py",
					},
				},
			},
		},
		{
			name: "some layers with missing annotations",
			input: []ocispec.Descriptor{
				{
					MediaType: modelspec.MediaTypeModelWeight,
					Annotations: map[string]string{
						modelspec.AnnotationFilepath: "b_weight.bin",
					},
				},
				{
					MediaType: modelspec.MediaTypeModelWeight,
					// No annotations
				},
				{
					MediaType: modelspec.MediaTypeModelWeight,
					Annotations: map[string]string{
						modelspec.AnnotationFilepath: "a_weight.bin",
					},
				},
			},
			expected: []ocispec.Descriptor{
				{
					MediaType: modelspec.MediaTypeModelWeight,
					// No annotations (empty filepath sorts before non-empty)
				},
				{
					MediaType: modelspec.MediaTypeModelWeight,
					Annotations: map[string]string{
						modelspec.AnnotationFilepath: "a_weight.bin",
					},
				},
				{
					MediaType: modelspec.MediaTypeModelWeight,
					Annotations: map[string]string{
						modelspec.AnnotationFilepath: "b_weight.bin",
					},
				},
			},
		},
		{
			name: "all media types in order",
			input: []ocispec.Descriptor{
				{
					MediaType: modelspec.MediaTypeModelWeightConfig,
					Annotations: map[string]string{
						modelspec.AnnotationFilepath: "b_config.json",
					},
				},
				{
					MediaType: modelspec.MediaTypeModelWeight,
					Annotations: map[string]string{
						modelspec.AnnotationFilepath: "weight.bin",
					},
				},
				{
					MediaType: modelspec.MediaTypeModelWeightConfig,
					Annotations: map[string]string{
						modelspec.AnnotationFilepath: "a_config.json",
					},
				},
				{
					MediaType: modelspec.MediaTypeModelCode,
					Annotations: map[string]string{
						modelspec.AnnotationFilepath: "code.py",
					},
				},
				{
					MediaType: modelspec.MediaTypeModelDoc,
					Annotations: map[string]string{
						modelspec.AnnotationFilepath: "doc.md",
					},
				},
			},
			expected: []ocispec.Descriptor{
				{
					MediaType: modelspec.MediaTypeModelWeightConfig,
					Annotations: map[string]string{
						modelspec.AnnotationFilepath: "a_config.json",
					},
				},
				{
					MediaType: modelspec.MediaTypeModelWeightConfig,
					Annotations: map[string]string{
						modelspec.AnnotationFilepath: "b_config.json",
					},
				},
				{
					MediaType: modelspec.MediaTypeModelWeight,
					Annotations: map[string]string{
						modelspec.AnnotationFilepath: "weight.bin",
					},
				},
				{
					MediaType: modelspec.MediaTypeModelCode,
					Annotations: map[string]string{
						modelspec.AnnotationFilepath: "code.py",
					},
				},
				{
					MediaType: modelspec.MediaTypeModelDoc,
					Annotations: map[string]string{
						modelspec.AnnotationFilepath: "doc.md",
					},
				},
			},
		},
		{
			name:     "empty input",
			input:    []ocispec.Descriptor{},
			expected: []ocispec.Descriptor{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sortLayers(tc.input)
			if !reflect.DeepEqual(tc.input, tc.expected) {
				t.Errorf("sortLayers() failed:\nexpected: %+v\ngot: %+v", tc.expected, tc.input)
			}
		})
	}
}
