/*
 *     Copyright 2026 The ModelPack Authors
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

package distribution

import (
	"bytes"
	"context"
	"errors"
	"testing"

	distribution "github.com/distribution/distribution/v3"
	godigest "github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/stretchr/testify/require"
)

func TestPushBlobPropagatesCommitError(t *testing.T) {
	storage, err := NewStorage(t.TempDir())
	require.NoError(t, err)

	_, _, err = storage.PushBlob(context.Background(), "example.com/test", bytes.NewBufferString("blob"), ocispec.Descriptor{
		Digest: godigest.FromString("different blob"),
		Size:   4,
	})

	var invalidDigest distribution.ErrBlobInvalidDigest
	require.True(t, errors.As(err, &invalidDigest))
}
