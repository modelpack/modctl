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

func TestLicenseProcessor_Name(t *testing.T) {
	p := NewLicenseProcessor()
	assert.Equal(t, "license", p.Name())
}

func TestLicenseProcessor_Identify(t *testing.T) {
	p := NewLicenseProcessor()
	mockFS := fstest.MapFS{
		"LICENSE":     &fstest.MapFile{},
		"LICENSE.txt": &fstest.MapFile{},
		"README.md":   &fstest.MapFile{},
	}
	info, err := mockFS.Stat("LICENSE")
	assert.NoError(t, err)
	assert.True(t, p.Identify(context.Background(), "LICENSE", info))

	info, err = mockFS.Stat("LICENSE.txt")
	assert.NoError(t, err)
	assert.True(t, p.Identify(context.Background(), "LICENSE.txt", info))

	info, err = mockFS.Stat("README.md")
	assert.NoError(t, err)
	assert.False(t, p.Identify(context.Background(), "README.md", info))
}

func TestLicenseProcessor_Process(t *testing.T) {
	p := NewLicenseProcessor()
	ctx := context.Background()
	mockStore := &storage.Storage{}
	repo := "test-repo"
	path := "/tmp/LICENSE"

	mockStore.On("PushBlob", ctx, repo, mock.Anything).Return("sha256:1234567890abcdef", int64(1024), nil)

	desc, err := p.Process(ctx, mockStore, repo, path, "/tmp")
	assert.NoError(t, err)
	assert.NotNil(t, desc)
	assert.Equal(t, "sha256:1234567890abcdef", desc.Digest.String())
	assert.Equal(t, int64(1024), desc.Size)
	assert.Equal(t, "true", desc.Annotations[modelspec.AnnotationLicense])
	assert.Equal(t, "LICENSE", desc.Annotations[modelspec.AnnotationFilepath])
}
