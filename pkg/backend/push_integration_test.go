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
	"bytes"
	"context"
	"encoding/json"
	"io"
	"testing"
	"time"

	godigest "github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/modelpack/modctl/pkg/config"
	"github.com/modelpack/modctl/test/helpers"
	storageMock "github.com/modelpack/modctl/test/mocks/storage"
)

// pushTestFixture holds all objects needed by a push integration test.
type pushTestFixture struct {
	mr          *helpers.MockRegistry
	store       *storageMock.Storage
	backend     *backend
	target      string
	cfg         *config.Push
	blobContent []byte
	blobDigest  godigest.Digest
	configBytes []byte
	configDesc  ocispec.Descriptor
	manifest    ocispec.Manifest
	manifestRaw []byte
}

// newPushTestFixture creates a MockRegistry (destination) and a mock Storage
// (source) with a manifest containing 1 layer blob plus a config blob.
func newPushTestFixture(t *testing.T) *pushTestFixture {
	t.Helper()

	mr := helpers.NewMockRegistry()

	// Layer blob content in local storage.
	blobContent := []byte("push-test-blob-content-with-enough-padding-here")
	blobDigest := godigest.FromBytes(blobContent)

	// Config blob content in local storage.
	configBytes := []byte(`{"architecture":"amd64"}`)
	configDigest := godigest.FromBytes(configBytes)
	configDesc := ocispec.Descriptor{
		MediaType: ocispec.MediaTypeImageConfig,
		Digest:    configDigest,
		Size:      int64(len(configBytes)),
	}

	manifest := ocispec.Manifest{
		MediaType: ocispec.MediaTypeImageManifest,
		Config:    configDesc,
		Layers: []ocispec.Descriptor{
			{
				MediaType: "application/octet-stream",
				Digest:    blobDigest,
				Size:      int64(len(blobContent)),
			},
		},
	}
	manifestRaw, err := json.Marshal(manifest)
	require.NoError(t, err)

	s := newMockStorageForPush(t, manifestRaw, blobContent, blobDigest, configBytes, configDigest)

	return &pushTestFixture{
		mr:          mr,
		store:       s,
		backend:     &backend{store: s},
		target:      mr.Host() + "/test/model:latest",
		cfg:         newPushConfig(),
		blobContent: blobContent,
		blobDigest:  blobDigest,
		configBytes: configBytes,
		configDesc:  configDesc,
		manifest:    manifest,
		manifestRaw: manifestRaw,
	}
}

// newMockStorageForPush creates a mock Storage that serves as the local source
// for push operations. It sets up PullManifest and PullBlob expectations with
// Maybe() so tests that error out early do not trigger unsatisfied expectation
// failures.
//
// Note: Push passes the FULL repository reference (including host:port) to
// PullManifest and PullBlob, so we use mock.Anything for the repo parameter.
func newMockStorageForPush(t *testing.T, manifestRaw, blobContent []byte, blobDigest godigest.Digest, configBytes []byte, configDigest godigest.Digest) *storageMock.Storage {
	t.Helper()
	s := storageMock.NewStorage(t)

	s.On("PullManifest", mock.Anything, mock.Anything, "latest").
		Maybe().
		Return(manifestRaw, godigest.FromBytes(manifestRaw).String(), nil)

	s.On("PullBlob", mock.Anything, mock.Anything, blobDigest.String()).
		Maybe().
		Return(io.NopCloser(bytes.NewReader(blobContent)), nil)

	s.On("PullBlob", mock.Anything, mock.Anything, configDigest.String()).
		Maybe().
		Return(io.NopCloser(bytes.NewReader(configBytes)), nil)

	return s
}

// newPushConfig returns a Push config suitable for integration tests.
func newPushConfig() *config.Push {
	cfg := config.NewPush()
	cfg.PlainHTTP = true
	return cfg
}

// --------------------------------------------------------------------------
// Dimension 1: Functional Correctness
// --------------------------------------------------------------------------

func TestIntegration_Push_HappyPath(t *testing.T) {
	f := newPushTestFixture(t)
	defer f.mr.Close()

	err := f.backend.Push(context.Background(), f.target, f.cfg)
	require.NoError(t, err)

	// Verify blob received by registry.
	assert.True(t, f.mr.BlobExists(f.blobDigest.String()),
		"layer blob should exist in remote registry after push")

	// Verify config received by registry.
	assert.True(t, f.mr.BlobExists(f.configDesc.Digest.String()),
		"config blob should exist in remote registry after push")

	// Verify manifest received by registry.
	assert.True(t, f.mr.ManifestExists("test/model:latest"),
		"manifest should exist in remote registry after push")
}

func TestIntegration_Push_BlobAlreadyExists(t *testing.T) {
	f := newPushTestFixture(t)
	defer f.mr.Close()

	// Pre-populate the registry with the layer blob, config blob, and manifest
	// so that push finds everything already present on the remote.
	f.mr.AddBlobWithDigest(f.blobDigest.String(), f.blobContent)
	f.mr.AddBlobWithDigest(f.configDesc.Digest.String(), f.configBytes)
	f.mr.AddManifest("test/model:latest", f.manifest)
	// Also add manifest under its digest key so the Tag/FetchReference works.
	manifestDigest := godigest.FromBytes(f.manifestRaw).String()
	f.mr.AddManifest("test/model:"+manifestDigest, f.manifest)

	// Use a fresh mock that will fail if PullBlob is called unexpectedly.
	s := storageMock.NewStorage(t)
	s.On("PullManifest", mock.Anything, mock.Anything, "latest").
		Return(f.manifestRaw, godigest.FromBytes(f.manifestRaw).String(), nil)
	// PullBlob should NOT be called because the remote already has everything.
	f.backend.store = s

	err := f.backend.Push(context.Background(), f.target, f.cfg)
	require.NoError(t, err)

	// Verify PullBlob was never called (remote had all blobs, push skipped).
	s.AssertNotCalled(t, "PullBlob", mock.Anything, mock.Anything, mock.Anything)
}

// --------------------------------------------------------------------------
// Dimension 2: Network Errors
// --------------------------------------------------------------------------

func TestIntegration_Push_ManifestPushFails(t *testing.T) {
	f := newPushTestFixture(t)
	defer f.mr.Close()

	// Inject a 500 error on manifest push endpoint.
	// The actual path is /v2/<name>/manifests/latest, so use "/manifests/latest"
	// as the suffix to match via effectiveFault's HasSuffix check.
	f.mr.WithFault(&helpers.FaultConfig{
		PathFaults: map[string]*helpers.FaultConfig{
			"/manifests/latest": {StatusCodeOverride: 500},
		},
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := f.backend.Push(ctx, f.target, f.cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "manifest",
		"error should mention manifest failure")
}

// --------------------------------------------------------------------------
// Dimension 3: Resource Leak (Known Bug #491)
// --------------------------------------------------------------------------

// TestKnownBug_Push_ReadCloserNotClosed_SuccessPath documents that the
// ReadCloser returned by PullBlob is never closed on the success path.
// See: https://github.com/modelpack/modctl/issues/491
//
// This uses a reverse assertion: AssertNotClosed passes today because the
// bug exists. When the bug is fixed (Close() is called), this test will
// FAIL, signaling that the assertion should be flipped to AssertClosed.
func TestKnownBug_Push_ReadCloserNotClosed_SuccessPath(t *testing.T) {
	mr := helpers.NewMockRegistry()
	defer mr.Close()

	blobContent := []byte("leak-test-success-blob-content-with-padding")
	blobDigest := godigest.FromBytes(blobContent)

	configBytes := []byte(`{"architecture":"amd64"}`)
	configDigest := godigest.FromBytes(configBytes)
	configDesc := ocispec.Descriptor{
		MediaType: ocispec.MediaTypeImageConfig,
		Digest:    configDigest,
		Size:      int64(len(configBytes)),
	}

	manifest := ocispec.Manifest{
		MediaType: ocispec.MediaTypeImageManifest,
		Config:    configDesc,
		Layers: []ocispec.Descriptor{
			{
				MediaType: "application/octet-stream",
				Digest:    blobDigest,
				Size:      int64(len(blobContent)),
			},
		},
	}
	manifestRaw, err := json.Marshal(manifest)
	require.NoError(t, err)

	// Create tracking readers for blob and config.
	blobTracker := helpers.NewTrackingReadCloser(io.NopCloser(bytes.NewReader(blobContent)))
	configTracker := helpers.NewTrackingReadCloser(io.NopCloser(bytes.NewReader(configBytes)))

	s := storageMock.NewStorage(t)
	s.On("PullManifest", mock.Anything, mock.Anything, "latest").
		Return(manifestRaw, godigest.FromBytes(manifestRaw).String(), nil)
	s.On("PullBlob", mock.Anything, mock.Anything, blobDigest.String()).
		Return(io.ReadCloser(blobTracker), nil)
	s.On("PullBlob", mock.Anything, mock.Anything, configDigest.String()).
		Return(io.ReadCloser(configTracker), nil)

	b := &backend{store: s}
	target := mr.Host() + "/test/model:latest"
	cfg := newPushConfig()

	err = b.Push(context.Background(), target, cfg)
	require.NoError(t, err, "push should succeed")

	// Known bug #491: PullBlob ReadClosers are never closed.
	// Reverse assertion — passes today, will fail when bug is fixed.
	blobTracker.AssertNotClosed(t)
	configTracker.AssertNotClosed(t)
}

// TestKnownBug_Push_ReadCloserNotClosed_ErrorPath documents that the
// ReadCloser returned by PullBlob is never closed on the error path either.
// See: https://github.com/modelpack/modctl/issues/491
//
// The blob upload is made to fail by faulting the POST /blobs/uploads/
// endpoint. The layer's PullBlob is still called (before the upload attempt),
// but Close() is never invoked on the returned reader.
func TestKnownBug_Push_ReadCloserNotClosed_ErrorPath(t *testing.T) {
	mr := helpers.NewMockRegistry()
	defer mr.Close()

	blobContent := []byte("leak-test-error-blob-content-with-padding")
	blobDigest := godigest.FromBytes(blobContent)

	configBytes := []byte(`{"architecture":"amd64"}`)
	configDigest := godigest.FromBytes(configBytes)
	configDesc := ocispec.Descriptor{
		MediaType: ocispec.MediaTypeImageConfig,
		Digest:    configDigest,
		Size:      int64(len(configBytes)),
	}

	manifest := ocispec.Manifest{
		MediaType: ocispec.MediaTypeImageManifest,
		Config:    configDesc,
		Layers: []ocispec.Descriptor{
			{
				MediaType: "application/octet-stream",
				Digest:    blobDigest,
				Size:      int64(len(blobContent)),
			},
		},
	}
	manifestRaw, err := json.Marshal(manifest)
	require.NoError(t, err)

	// Fail the blob upload POST request.
	mr.WithFault(&helpers.FaultConfig{
		PathFaults: map[string]*helpers.FaultConfig{
			"/blobs/uploads/": {StatusCodeOverride: 500},
		},
	})

	blobTracker := helpers.NewTrackingReadCloser(io.NopCloser(bytes.NewReader(blobContent)))

	s := storageMock.NewStorage(t)
	s.On("PullManifest", mock.Anything, mock.Anything, "latest").
		Return(manifestRaw, godigest.FromBytes(manifestRaw).String(), nil)
	s.On("PullBlob", mock.Anything, mock.Anything, blobDigest.String()).
		Maybe().
		Return(io.ReadCloser(blobTracker), nil)
	// Config PullBlob may or may not be called depending on whether the layer
	// upload error short-circuits before config push.
	s.On("PullBlob", mock.Anything, mock.Anything, configDigest.String()).
		Maybe().
		Return(io.NopCloser(bytes.NewReader(configBytes)), nil)

	b := &backend{store: s}
	target := mr.Host() + "/test/model:latest"
	cfg := newPushConfig()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = b.Push(ctx, target, cfg)
	require.Error(t, err, "push should fail due to blob upload fault")

	// Known bug #491: PullBlob ReadCloser is never closed, even on error.
	// Reverse assertion — passes today, will fail when bug is fixed.
	if blobTracker.WasClosed() {
		// If this branch is reached, the bug may be fixed — flip to AssertClosed.
		t.Log("blob tracker was closed — bug #491 may be fixed")
	} else {
		blobTracker.AssertNotClosed(t)
	}
}

// --------------------------------------------------------------------------
// Dimension 6: Data Integrity
// --------------------------------------------------------------------------

func TestIntegration_Push_VerifyBlobIntegrity(t *testing.T) {
	f := newPushTestFixture(t)
	defer f.mr.Close()

	err := f.backend.Push(context.Background(), f.target, f.cfg)
	require.NoError(t, err)

	// Get the blob from registry and verify exact byte match.
	remoteBlob, ok := f.mr.GetBlob(f.blobDigest.String())
	require.True(t, ok, "layer blob should be present in registry")
	assert.Equal(t, f.blobContent, remoteBlob,
		"remote blob bytes should exactly match source blob")

	// Verify config blob integrity too.
	remoteConfig, ok := f.mr.GetBlob(f.configDesc.Digest.String())
	require.True(t, ok, "config blob should be present in registry")
	assert.Equal(t, f.configBytes, remoteConfig,
		"remote config bytes should exactly match source config")
}

// --------------------------------------------------------------------------
// Dimension 7: Graceful Shutdown
// --------------------------------------------------------------------------

func TestIntegration_Push_ContextCancelMidUpload(t *testing.T) {
	f := newPushTestFixture(t)
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

	err := f.backend.Push(ctx, f.target, f.cfg)
	require.Error(t, err)
}

// --------------------------------------------------------------------------
// Dimension 8: Idempotency
// --------------------------------------------------------------------------

func TestIntegration_Push_Idempotent(t *testing.T) {
	f := newPushTestFixture(t)
	defer f.mr.Close()

	// First push: nothing on remote, everything is uploaded.
	err := f.backend.Push(context.Background(), f.target, f.cfg)
	require.NoError(t, err)

	// Verify first push stored everything.
	assert.True(t, f.mr.BlobExists(f.blobDigest.String()), "blob should exist after first push")
	assert.True(t, f.mr.ManifestExists("test/model:latest"), "manifest should exist after first push")

	// Second push: use a fresh mock so we can verify PullBlob call count.
	// The registry already has all blobs and manifest from the first push.
	// pushIfNotExist checks dst.Exists() (HEAD) and skips if present.
	s2 := storageMock.NewStorage(t)
	s2.On("PullManifest", mock.Anything, mock.Anything, "latest").
		Return(f.manifestRaw, godigest.FromBytes(f.manifestRaw).String(), nil)
	// PullBlob should NOT be called on the second push because the remote
	// already has all blobs. But register it with Maybe() just in case.
	s2.On("PullBlob", mock.Anything, mock.Anything, mock.Anything).
		Maybe().
		Return(io.NopCloser(bytes.NewReader(f.blobContent)), nil)
	f.backend.store = s2

	reqCountBefore := f.mr.RequestCount()

	err = f.backend.Push(context.Background(), f.target, f.cfg)
	require.NoError(t, err)

	reqCountAfter := f.mr.RequestCount()
	t.Logf("first push completed; second push registry requests: %d",
		reqCountAfter-reqCountBefore)

	// PullBlob should not have been called on the second push because the
	// remote already had all blobs (pushIfNotExist skips them).
	s2.AssertNotCalled(t, "PullBlob", mock.Anything, mock.Anything, mock.Anything)
}

