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
	"io"
	"sync/atomic"
	"testing"
	"time"

	godigest "github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/modelpack/modctl/pkg/config"
	storageMock "github.com/modelpack/modctl/test/mocks/storage"

	"github.com/modelpack/modctl/test/helpers"
)

// pullTestFixture holds all objects needed by a pull integration test.
type pullTestFixture struct {
	mr       *helpers.MockRegistry
	store    *storageMock.Storage
	backend  *backend
	target   string
	cfg      *config.Pull
	blobs    [][]byte // raw blob contents
	digests  []string // blob digests (sha256:...)
	manifest ocispec.Manifest
}

// addManifestWithDigestKey registers the manifest under its digest key so
// that oras-go can fetch it by digest (used by pullIfNotExist for the
// manifest copy step). Call this after any AddManifest("repo:tag", ...).
func addManifestWithDigestKey(t *testing.T, mr *helpers.MockRegistry, repo string, manifest ocispec.Manifest) {
	t.Helper()
	manifestBytes, err := json.Marshal(manifest)
	require.NoError(t, err, "marshal manifest for digest key")
	manifestDigest := godigest.FromBytes(manifestBytes).String()
	mr.AddManifest(repo+":"+manifestDigest, manifest)
}

// newPullTestFixture creates a MockRegistry with a manifest containing
// blobCount layers plus a config blob. The storage mock uses Maybe()
// expectations so tests that error out early do not trigger unsatisfied
// expectation failures.
func newPullTestFixture(t *testing.T, blobCount int) *pullTestFixture {
	t.Helper()

	mr := helpers.NewMockRegistry()

	// Create layer blobs.
	blobs := make([][]byte, blobCount)
	digests := make([]string, blobCount)
	layers := make([]ocispec.Descriptor, blobCount)
	for i := 0; i < blobCount; i++ {
		blobs[i] = []byte(fmt.Sprintf("blob-content-%d-padding-to-make-it-longer", i))
		digests[i] = mr.AddBlob(blobs[i])
		layers[i] = ocispec.Descriptor{
			MediaType: "application/octet-stream",
			Digest:    godigest.Digest(digests[i]),
			Size:      int64(len(blobs[i])),
		}
	}

	// Create config blob.
	configContent := []byte(`{"architecture":"amd64"}`)
	configDigest := mr.AddBlob(configContent)
	configDesc := ocispec.Descriptor{
		MediaType: ocispec.MediaTypeImageConfig,
		Digest:    godigest.Digest(configDigest),
		Size:      int64(len(configContent)),
	}

	manifest := ocispec.Manifest{
		MediaType: ocispec.MediaTypeImageManifest,
		Config:    configDesc,
		Layers:    layers,
	}
	mr.AddManifest("test/model:latest", manifest)
	addManifestWithDigestKey(t, mr, "test/model", manifest)

	s := newMockStorageForPull(t)

	return &pullTestFixture{
		mr:       mr,
		store:    s,
		backend:  &backend{store: s},
		target:   mr.Host() + "/test/model:latest",
		cfg:      newPullConfig(),
		blobs:    blobs,
		digests:  digests,
		manifest: manifest,
	}
}

// newMockStorageForPull creates a mock Storage with Maybe() expectations
// that accept all Stat/Push calls. Using Maybe() ensures tests that error
// out early (network errors, context cancellation) do not fail from
// unsatisfied expectations.
func newMockStorageForPull(t *testing.T) *storageMock.Storage {
	t.Helper()
	s := storageMock.NewStorage(t)
	s.On("StatBlob", mock.Anything, mock.Anything, mock.Anything).
		Maybe().Return(false, nil)
	s.On("PushBlob", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Maybe().
		Run(func(args mock.Arguments) {
			// Drain the reader so the pull pipeline completes and digest
			// validation sees all bytes.
			r := args.Get(2).(io.Reader)
			_, _ = io.Copy(io.Discard, r)
		}).
		Return("", int64(0), nil)
	s.On("StatManifest", mock.Anything, mock.Anything, mock.Anything).
		Maybe().Return(false, nil)
	s.On("PushManifest", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Maybe().Return("", nil)
	return s
}

// newPullConfig returns a Pull config suitable for integration tests
// (PlainHTTP, progress disabled, reasonable concurrency).
func newPullConfig() *config.Pull {
	cfg := config.NewPull()
	cfg.PlainHTTP = true
	cfg.DisableProgress = true
	cfg.Concurrency = 5
	return cfg
}

// --------------------------------------------------------------------------
// Dimension 1: Functional Correctness
// --------------------------------------------------------------------------

func TestIntegration_Pull_HappyPath(t *testing.T) {
	f := newPullTestFixture(t, 2)
	defer f.mr.Close()

	err := f.backend.Pull(context.Background(), f.target, f.cfg)
	require.NoError(t, err)

	// Verify PushBlob was called for each layer (2) + config (1) = 3 times.
	f.store.AssertNumberOfCalls(t, "PushBlob", 3)
	// Verify PushManifest was called once for the manifest itself.
	f.store.AssertNumberOfCalls(t, "PushManifest", 1)
}

func TestIntegration_Pull_BlobAlreadyExists(t *testing.T) {
	f := newPullTestFixture(t, 2)
	defer f.mr.Close()

	// Replace the storage mock: StatBlob returns true (blob exists locally).
	s := storageMock.NewStorage(t)
	s.On("StatBlob", mock.Anything, mock.Anything, mock.Anything).
		Return(true, nil)
	s.On("StatManifest", mock.Anything, mock.Anything, mock.Anything).
		Return(false, nil)
	s.On("PushManifest", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Maybe().Return("", nil)
	f.backend.store = s

	err := f.backend.Pull(context.Background(), f.target, f.cfg)
	require.NoError(t, err)

	// PushBlob should never be called since all blobs already exist.
	s.AssertNotCalled(t, "PushBlob", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	// PushManifest is called once (manifest is not a blob, checked via StatManifest).
	s.AssertNumberOfCalls(t, "PushManifest", 1)
}

func TestIntegration_Pull_ConcurrentLayers(t *testing.T) {
	const blobCount = 5
	f := newPullTestFixture(t, blobCount)
	defer f.mr.Close()

	err := f.backend.Pull(context.Background(), f.target, f.cfg)
	require.NoError(t, err)

	// 5 layer blobs + 1 config blob = 6 PushBlob calls.
	f.store.AssertNumberOfCalls(t, "PushBlob", blobCount+1)
	f.store.AssertNumberOfCalls(t, "PushManifest", 1)
}

// --------------------------------------------------------------------------
// Dimension 2: Network Errors
// --------------------------------------------------------------------------

func TestIntegration_Pull_ContextTimeout(t *testing.T) {
	f := newPullTestFixture(t, 1)
	defer f.mr.Close()

	// Add 1s latency per request; context expires in 200ms.
	f.mr.WithFault(&helpers.FaultConfig{
		LatencyPerRequest: 1 * time.Second,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	err := f.backend.Pull(ctx, f.target, f.cfg)
	require.Error(t, err)
	assert.ErrorIs(t, ctx.Err(), context.DeadlineExceeded)
}

func TestIntegration_Pull_PartialResponse(t *testing.T) {
	f := newPullTestFixture(t, 1)
	defer f.mr.Close()

	// FailAfterNBytes on blob — truncate after 5 bytes.
	blobPath := fmt.Sprintf("/blobs/%s", f.digests[0])
	f.mr.WithFault(&helpers.FaultConfig{
		PathFaults: map[string]*helpers.FaultConfig{
			blobPath: {FailAfterNBytes: 5},
		},
	})

	// Short timeout: first attempt fails fast, retry backoff (5s) is
	// interrupted by context cancellation.
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := f.backend.Pull(ctx, f.target, f.cfg)
	require.Error(t, err)
}

func TestIntegration_Pull_ManifestOK_BlobFails(t *testing.T) {
	f := newPullTestFixture(t, 2)
	defer f.mr.Close()

	// Make the second blob fail with 500.
	blobPath := fmt.Sprintf("/blobs/%s", f.digests[1])
	f.mr.WithFault(&helpers.FaultConfig{
		PathFaults: map[string]*helpers.FaultConfig{
			blobPath: {StatusCodeOverride: 500},
		},
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := f.backend.Pull(ctx, f.target, f.cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to pull blob to local")
}

// --------------------------------------------------------------------------
// Dimension 4: Concurrency Safety
// --------------------------------------------------------------------------

func TestIntegration_Pull_ConcurrentPartialFailure(t *testing.T) {
	const blobCount = 5
	f := newPullTestFixture(t, blobCount)
	defer f.mr.Close()

	// Make 2 of the 5 blobs fail with 500.
	pathFaults := make(map[string]*helpers.FaultConfig)
	for i := 0; i < 2; i++ {
		blobPath := fmt.Sprintf("/blobs/%s", f.digests[i])
		pathFaults[blobPath] = &helpers.FaultConfig{StatusCodeOverride: 500}
	}
	f.mr.WithFault(&helpers.FaultConfig{
		PathFaults: pathFaults,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := f.backend.Pull(ctx, f.target, f.cfg)
	require.Error(t, err)
}

// --------------------------------------------------------------------------
// Dimension 6: Data Integrity
// --------------------------------------------------------------------------

func TestIntegration_Pull_TruncatedBlob(t *testing.T) {
	f := newPullTestFixture(t, 1)
	defer f.mr.Close()

	// Truncate blob after 5 bytes — should cause either read error or
	// digest validation failure.
	blobPath := fmt.Sprintf("/blobs/%s", f.digests[0])
	f.mr.WithFault(&helpers.FaultConfig{
		PathFaults: map[string]*helpers.FaultConfig{
			blobPath: {FailAfterNBytes: 5},
		},
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := f.backend.Pull(ctx, f.target, f.cfg)
	require.Error(t, err)
}

func TestIntegration_Pull_CorruptedBlob(t *testing.T) {
	// Build a fixture with 1 blob, then swap it for corrupt content.
	f := newPullTestFixture(t, 1)
	defer f.mr.Close()

	// Replace the real blob content with garbage under the same digest.
	corruptContent := []byte("this-is-totally-wrong-content-that-does-not-match-digest")
	f.mr.AddBlobWithDigest(f.digests[0], corruptContent)

	// Update the layer descriptor size to match the corrupt content length,
	// otherwise oras-go may reject the response before digest validation.
	// Rebuild the manifest with the new size.
	f.manifest.Layers[0].Size = int64(len(corruptContent))
	f.mr.AddManifest("test/model:latest", f.manifest)
	addManifestWithDigestKey(t, f.mr, "test/model", f.manifest)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := f.backend.Pull(ctx, f.target, f.cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "digest")
}

// --------------------------------------------------------------------------
// Dimension 7: Graceful Shutdown
// --------------------------------------------------------------------------

func TestIntegration_Pull_ContextCancelMidDownload(t *testing.T) {
	f := newPullTestFixture(t, 3)
	defer f.mr.Close()

	// Add latency so we can cancel mid-flight.
	f.mr.WithFault(&helpers.FaultConfig{
		LatencyPerRequest: 1 * time.Second,
	})

	ctx, cancel := context.WithCancel(context.Background())
	// Cancel after 200ms.
	go func() {
		time.Sleep(200 * time.Millisecond)
		cancel()
	}()

	err := f.backend.Pull(ctx, f.target, f.cfg)
	require.Error(t, err)
}

// --------------------------------------------------------------------------
// Dimension 8: Idempotency
// --------------------------------------------------------------------------

func TestIntegration_Pull_Idempotent(t *testing.T) {
	f := newPullTestFixture(t, 2)
	defer f.mr.Close()

	// First pull: everything is new.
	err := f.backend.Pull(context.Background(), f.target, f.cfg)
	require.NoError(t, err)

	reqCountAfterFirst := f.mr.RequestCount()

	// Second pull: all blobs and manifest exist locally.
	s2 := storageMock.NewStorage(t)
	s2.On("StatBlob", mock.Anything, mock.Anything, mock.Anything).
		Return(true, nil)
	s2.On("StatManifest", mock.Anything, mock.Anything, mock.Anything).
		Return(true, nil)
	f.backend.store = s2

	err = f.backend.Pull(context.Background(), f.target, f.cfg)
	require.NoError(t, err)

	reqCountAfterSecond := f.mr.RequestCount()
	delta := reqCountAfterSecond - reqCountAfterFirst
	t.Logf("first pull requests: %d, second pull delta: %d", reqCountAfterFirst, delta)

	// Both pulls still fetch from the remote (the Pull function always
	// fetches the manifest, config, and layers from the registry before
	// checking local existence). The key difference is that the second
	// pull should NOT write anything to local storage.
	s2.AssertNotCalled(t, "PushBlob", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	s2.AssertNotCalled(t, "PushManifest", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

// --------------------------------------------------------------------------
// Additional: concurrency tracking test
// --------------------------------------------------------------------------

func TestIntegration_Pull_ConcurrentLayers_AllStored(t *testing.T) {
	const blobCount = 5
	mr := helpers.NewMockRegistry()
	defer mr.Close()

	// Create blobs.
	digests := make([]string, blobCount)
	layers := make([]ocispec.Descriptor, blobCount)
	for i := 0; i < blobCount; i++ {
		content := []byte(fmt.Sprintf("concurrent-blob-%d-with-enough-padding", i))
		digests[i] = mr.AddBlob(content)
		layers[i] = ocispec.Descriptor{
			MediaType: "application/octet-stream",
			Digest:    godigest.Digest(digests[i]),
			Size:      int64(len(content)),
		}
	}

	configContent := []byte(`{"architecture":"arm64"}`)
	configDigest := mr.AddBlob(configContent)
	manifest := ocispec.Manifest{
		MediaType: ocispec.MediaTypeImageManifest,
		Config: ocispec.Descriptor{
			MediaType: ocispec.MediaTypeImageConfig,
			Digest:    godigest.Digest(configDigest),
			Size:      int64(len(configContent)),
		},
		Layers: layers,
	}
	mr.AddManifest("test/model:latest", manifest)
	addManifestWithDigestKey(t, mr, "test/model", manifest)

	// Track which digests were pushed via an atomic counter.
	var pushCount atomic.Int32
	s := storageMock.NewStorage(t)
	s.On("StatBlob", mock.Anything, mock.Anything, mock.Anything).
		Maybe().Return(false, nil)
	s.On("PushBlob", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Maybe().
		Run(func(args mock.Arguments) {
			r := args.Get(2).(io.Reader)
			_, _ = io.Copy(io.Discard, r)
			pushCount.Add(1)
		}).
		Return("", int64(0), nil)
	s.On("StatManifest", mock.Anything, mock.Anything, mock.Anything).
		Maybe().Return(false, nil)
	s.On("PushManifest", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Maybe().Return("", nil)

	b := &backend{store: s}
	cfg := newPullConfig()
	target := mr.Host() + "/test/model:latest"

	err := b.Pull(context.Background(), target, cfg)
	require.NoError(t, err)

	// 5 layers + 1 config = 6
	assert.Equal(t, int32(blobCount+1), pushCount.Load(), "all blobs should be pushed to storage")
}
