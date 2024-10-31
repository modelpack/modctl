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

	"github.com/CloudNativeAI/modctl/test/mocks/storage"

	godigest "github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/stretchr/testify/assert"
)

func TestPrune(t *testing.T) {
	mockStore := &storage.Storage{}
	b := &backend{store: mockStore}
	ctx := context.Background()
	repos := []string{"example.com/repo1", "example.com/repo2"}

	mustToJson := func(v any) []byte {
		jsonByte, err := json.Marshal(v)
		assert.NoError(t, err)
		return jsonByte
	}

	indexs := []ocispec.Index{
		{
			Manifests: []ocispec.Descriptor{
				{Digest: godigest.Digest("sha256:1234567890abcdef-repo1")},
			},
		},
		{
			Manifests: []ocispec.Descriptor{
				{Digest: godigest.Digest("sha256:1234567890abcdef-repo2")},
			},
		},
	}
	manifests := []ocispec.Manifest{
		{Layers: []ocispec.Descriptor{{Digest: godigest.Digest("sha256:1234567890abcdef-repo1-blob1")}}},
		{Layers: []ocispec.Descriptor{{Digest: godigest.Digest("sha256:1234567890abcdef-repo2-blob1")}}},
	}
	blobs := []string{
		"sha256:1234567890abcdef-repo1-blob1",
		"sha256:1234567890abcdef-repo1-blob2",
		"sha256:1234567890abcdef-repo2-blob1",
		"sha256:1234567890abcdef-repo2-blob2",
	}

	mockStore.On("ListRepositories", ctx).Return(repos, nil)
	mockStore.On("GetIndex", ctx, repos[0]).Return(mustToJson(indexs[0]), nil)
	mockStore.On("GetIndex", ctx, repos[1]).Return(mustToJson(indexs[1]), nil)
	mockStore.On("PullManifest", ctx, repos[0], indexs[0].Manifests[0].Digest.String()).Return(mustToJson(manifests[0]), indexs[0].Manifests[0].Digest.String(), nil)
	mockStore.On("PullManifest", ctx, repos[1], indexs[1].Manifests[0].Digest.String()).Return(mustToJson(manifests[1]), indexs[1].Manifests[0].Digest.String(), nil)
	mockStore.On("ListBlobs", ctx, repos[0]).Return([]string{blobs[0], blobs[1]}, nil)
	mockStore.On("ListBlobs", ctx, repos[1]).Return([]string{blobs[2], blobs[3]}, nil)
	mockStore.On("CleanupRepo", ctx, repos[0], []string{blobs[1]}, false).Return(1, nil)
	mockStore.On("CleanupRepo", ctx, repos[1], []string{blobs[3]}, false).Return(1, nil)

	prunedBlobs, err := b.Prune(ctx)
	assert.NoError(t, err)
	assert.Len(t, prunedBlobs, 2, "pruned blobs length is not equal to expected length")
	assert.Equal(t, repos[0]+"@"+blobs[1], prunedBlobs[0], "pruned blob is not equal to expected blob")
	assert.Equal(t, repos[1]+"@"+blobs[3], prunedBlobs[1], "pruned blob is not equal to expected blob")
}
