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
	"errors"
	"testing"

	godigest "github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	storageMock "github.com/modelpack/modctl/test/mocks/storage"
)

func TestEstimatePullSize(t *testing.T) {
	ctx := context.Background()
	const repo = "test/model"
	manifest := &ocispec.Manifest{
		Config: ocispec.Descriptor{Digest: godigest.Digest("sha256:cfg"), Size: 10},
		Layers: []ocispec.Descriptor{
			{Digest: godigest.Digest("sha256:l1"), Size: 100},
			{Digest: godigest.Digest("sha256:l2"), Size: 200},
		},
	}

	t.Run("counts config and all layers when nothing exists locally", func(t *testing.T) {
		s := storageMock.NewStorage(t)
		s.On("StatBlob", mock.Anything, repo, mock.Anything).Return(false, nil)

		assert.Equal(t, int64(310), estimatePullSize(ctx, s, repo, manifest, false))
	})

	t.Run("skips blobs that already exist locally", func(t *testing.T) {
		s := storageMock.NewStorage(t)
		s.On("StatBlob", mock.Anything, repo, "sha256:cfg").Return(true, nil)
		s.On("StatBlob", mock.Anything, repo, "sha256:l1").Return(true, nil)
		s.On("StatBlob", mock.Anything, repo, "sha256:l2").Return(false, nil)

		assert.Equal(t, int64(200), estimatePullSize(ctx, s, repo, manifest, false))
	})

	t.Run("repeated pull of the same model needs no space", func(t *testing.T) {
		s := storageMock.NewStorage(t)
		s.On("StatBlob", mock.Anything, repo, mock.Anything).Return(true, nil)

		assert.Equal(t, int64(0), estimatePullSize(ctx, s, repo, manifest, false))
	})

	t.Run("stat error counts the blob", func(t *testing.T) {
		s := storageMock.NewStorage(t)
		s.On("StatBlob", mock.Anything, repo, mock.Anything).Return(false, errors.New("stat failed"))

		assert.Equal(t, int64(310), estimatePullSize(ctx, s, repo, manifest, false))
	})

	t.Run("extract-from-remote counts all layers and never stats", func(t *testing.T) {
		s := storageMock.NewStorage(t)

		assert.Equal(t, int64(300), estimatePullSize(ctx, s, repo, manifest, true))
	})
}
