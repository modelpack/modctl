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

package processor

import (
	"context"
	"testing"
	"testing/fstest"

	"github.com/CloudNativeAI/modctl/test/mocks/storage"
	modelspec "github.com/CloudNativeAI/model-spec/specs-go/v1"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestModelProcessor_Name(t *testing.T) {
	p := NewModelProcessor([]string{"model*"})
	assert.Equal(t, "model", p.Name())
}

func TestModelProcessor_Identify(t *testing.T) {
	p := NewModelProcessor([]string{"model*"})
	mockFS := fstest.MapFS{
		"model-1": &fstest.MapFile{},
		"model-2": &fstest.MapFile{},
		"config":  &fstest.MapFile{},
	}
	info, err := mockFS.Stat("model-1")
	assert.NoError(t, err)
	assert.True(t, p.Identify(context.Background(), "model-1", info))

	info, err = mockFS.Stat("model-2")
	assert.NoError(t, err)
	assert.True(t, p.Identify(context.Background(), "model-2", info))

	info, err = mockFS.Stat("config")
	assert.NoError(t, err)
	assert.False(t, p.Identify(context.Background(), "config", info))
}

func TestModelProcessor_Process(t *testing.T) {
	p := NewModelProcessor([]string{"model*"})
	ctx := context.Background()
	mockStore := &storage.Storage{}
	repo := "test-repo"
	path := "/tmp/model"

	mockStore.On("PushBlob", ctx, repo, mock.Anything).Return("sha256:1234567890abcdef", int64(1024), nil)

	desc, err := p.Process(ctx, mockStore, repo, path, "/tmp")
	assert.NoError(t, err)
	assert.NotNil(t, desc)
	assert.Equal(t, "sha256:1234567890abcdef", desc.Digest.String())
	assert.Equal(t, int64(1024), desc.Size)
	assert.Equal(t, "model", desc.Annotations[modelspec.AnnotationFilepath])
}
