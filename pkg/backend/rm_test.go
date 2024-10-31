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
	"testing"

	"github.com/CloudNativeAI/modctl/test/mocks/storage"

	"github.com/stretchr/testify/assert"
)

func TestRemove(t *testing.T) {
	mockStore := &storage.Storage{}
	b := &backend{store: mockStore}
	ctx := context.Background()
	target := "example.com/repo:tag"
	ref, err := ParseReference("example.com/repo:tag")
	assert.NoError(t, err)
	digest := "sha256:1234567890abcdef"

	mockStore.On("PullManifest", ctx, ref.Repository(), ref.Tag()).Return(nil, digest, nil)
	mockStore.On("DeleteManifest", ctx, ref.Repository(), digest).Return(nil)

	result, err := b.Remove(ctx, target)
	assert.NoError(t, err)
	assert.Equal(t, digest, result)

	mockStore.AssertExpectations(t)
}
