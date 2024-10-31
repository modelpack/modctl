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
	"testing"
	"time"

	"github.com/CloudNativeAI/modctl/test/mocks/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

func TestList(t *testing.T) {
	mockStore := &storage.Storage{}
	b := &backend{store: mockStore}
	ctx := context.Background()
	repos := []string{"example.com/repo1", "example.com/repo2"}
	tags := []string{"tag1", "tag2"}
	createdAt := time.Now().Format(time.RFC3339)
	manifest := ocispec.Manifest{
		Layers: []ocispec.Descriptor{
			{Size: 1024},
			{Size: 1024},
		},
		Config: ocispec.Descriptor{Size: 1024},
		Annotations: map[string]string{
			"org.cnai.model.created": createdAt,
		},
	}
	manifestRaw, err := json.Marshal(manifest)
	assert.NoError(t, err)

	mockStore.On("ListRepositories", ctx).Return(repos, nil)
	mockStore.On("ListTags", ctx, repos[0]).Return(tags, nil)
	mockStore.On("ListTags", ctx, repos[1]).Return(tags, nil)
	mockStore.On("PullManifest", ctx, mock.Anything, mock.Anything).Return(manifestRaw, "sha256:1234567890abcdef", nil)

	artifacts, err := b.List(ctx)
	assert.NoError(t, err, "list failed")
	assert.Len(t, artifacts, 4, "unexpected number of artifacts")
	assert.Equal(t, repos[0], artifacts[0].Repository, "unexpected repository")
	assert.Equal(t, tags[0], artifacts[0].Tag, "unexpected tag")
	assert.Equal(t, "sha256:1234567890abcdef", artifacts[0].Digest, "unexpected digest")
	assert.Equal(t, int64(3*1024+len(manifestRaw)), artifacts[0].Size, "unexpected size")
	assert.Equal(t, createdAt, artifacts[0].CreatedAt.Format(time.RFC3339), "unexpected created at")
}
