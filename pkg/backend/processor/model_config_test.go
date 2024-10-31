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

	modelspec "github.com/CloudNativeAI/modctl/pkg/spec"
	"github.com/CloudNativeAI/modctl/test/mocks/storage"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestModelConfigProcessor_Name(t *testing.T) {
	p := NewModelConfigProcessor([]string{"config*"})
	assert.Equal(t, "model_config", p.Name())
}

func TestModelConfigProcessor_Identify(t *testing.T) {
	p := NewModelConfigProcessor([]string{"config*"})
	mockFS := fstest.MapFS{
		"config-1": &fstest.MapFile{},
		"config-2": &fstest.MapFile{},
		"model":    &fstest.MapFile{},
	}
	info, err := mockFS.Stat("config-1")
	assert.NoError(t, err)
	assert.True(t, p.Identify(context.Background(), "config-1", info))

	info, err = mockFS.Stat("config-2")
	assert.NoError(t, err)
	assert.True(t, p.Identify(context.Background(), "config-2", info))

	info, err = mockFS.Stat("model")
	assert.NoError(t, err)
	assert.False(t, p.Identify(context.Background(), "model", info))
}

func TestModelConfigProcessor_Process(t *testing.T) {
	p := NewModelConfigProcessor([]string{"config*"})
	ctx := context.Background()
	mockStore := &storage.Storage{}
	repo := "test-repo"
	path := "config"
	mockFS := fstest.MapFS{
		"config": &fstest.MapFile{},
	}
	info, err := mockFS.Stat("config")
	assert.NoError(t, err)

	mockStore.On("PushBlob", ctx, repo, mock.Anything).Return("sha256:1234567890abcdef", int64(1024), nil)

	desc, err := p.Process(ctx, mockStore, repo, path, info)
	assert.NoError(t, err)
	assert.NotNil(t, desc)
	assert.Equal(t, "sha256:1234567890abcdef", desc.Digest.String())
	assert.Equal(t, int64(1024), desc.Size)
	assert.Equal(t, "true", desc.Annotations[modelspec.AnnotationConfig])
}
