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

package helpers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	godigest "github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// baseURL returns the HTTP base URL for the mock registry.
func baseURL(r *MockRegistry) string {
	return "http://" + r.Host()
}

// TestMockRegistry_PingAndBlobRoundTrip verifies the /v2/ ping and blob GET.
func TestMockRegistry_PingAndBlobRoundTrip(t *testing.T) {
	r := NewMockRegistry()
	defer r.Close()

	content := []byte("hello, blob!")
	dgst := r.AddBlob(content)

	// Ping
	resp, err := http.Get(baseURL(r) + "/v2/")
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// GET blob
	resp, err = http.Get(fmt.Sprintf("%s/v2/myrepo/blobs/%s", baseURL(r), dgst))
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Equal(t, content, body)

	// Verify helper
	assert.True(t, r.BlobExists(dgst))
	stored, ok := r.GetBlob(dgst)
	require.True(t, ok)
	assert.Equal(t, content, stored)
}

// TestMockRegistry_ManifestRoundTrip verifies manifest storage and retrieval.
func TestMockRegistry_ManifestRoundTrip(t *testing.T) {
	r := NewMockRegistry()
	defer r.Close()

	content := []byte("layer-content")
	dgst := godigest.FromBytes(content)

	manifest := ocispec.Manifest{
		MediaType: ocispec.MediaTypeImageManifest,
		Layers: []ocispec.Descriptor{
			{
				MediaType: "application/octet-stream",
				Digest:    dgst,
				Size:      int64(len(content)),
			},
		},
	}

	r.AddManifest("myrepo:latest", manifest)
	assert.True(t, r.ManifestExists("myrepo:latest"))

	// HEAD
	req, _ := http.NewRequest(http.MethodHead, baseURL(r)+"/v2/myrepo/manifests/latest", nil)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.NotEmpty(t, resp.Header.Get("Docker-Content-Digest"))

	// GET
	resp, err = http.Get(baseURL(r) + "/v2/myrepo/manifests/latest")
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, manifestMediaType, resp.Header.Get("Content-Type"))

	var got ocispec.Manifest
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&got))
	require.Len(t, got.Layers, 1)
	assert.Equal(t, dgst, got.Layers[0].Digest)
}

// TestMockRegistry_FaultStatusCodeOverride verifies StatusCodeOverride.
func TestMockRegistry_FaultStatusCodeOverride(t *testing.T) {
	r := NewMockRegistry().WithFault(&FaultConfig{StatusCodeOverride: http.StatusServiceUnavailable})
	defer r.Close()

	resp, err := http.Get(baseURL(r) + "/v2/")
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
}

// TestMockRegistry_FaultFailOnNthRequest verifies that the first N requests
// return 500 and request N+1 succeeds.
func TestMockRegistry_FaultFailOnNthRequest(t *testing.T) {
	const n = 3
	r := NewMockRegistry().WithFault(&FaultConfig{FailOnNthRequest: n})
	defer r.Close()

	for i := 1; i <= n; i++ {
		resp, err := http.Get(baseURL(r) + "/v2/")
		require.NoError(t, err)
		resp.Body.Close()
		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode,
			"request %d should fail", i)
	}

	// Request n+1 should succeed.
	resp, err := http.Get(baseURL(r) + "/v2/")
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode, "request %d should succeed", n+1)
}

// TestMockRegistry_FaultLatency verifies LatencyPerRequest injects a delay.
func TestMockRegistry_FaultLatency(t *testing.T) {
	const delay = 100 * time.Millisecond
	r := NewMockRegistry().WithFault(&FaultConfig{LatencyPerRequest: delay})
	defer r.Close()

	start := time.Now()
	resp, err := http.Get(baseURL(r) + "/v2/")
	elapsed := time.Since(start)

	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.GreaterOrEqual(t, elapsed, delay, "response should be delayed by at least %s", delay)
}

// TestMockRegistry_RequestCounting verifies global and per-path counters.
// No fault configuration is needed: every request path is tracked regardless.
func TestMockRegistry_RequestCounting(t *testing.T) {
	r := NewMockRegistry()
	defer r.Close()

	for i := 0; i < 5; i++ {
		resp, err := http.Get(baseURL(r) + "/v2/")
		require.NoError(t, err)
		resp.Body.Close()
	}

	assert.Equal(t, int64(5), r.RequestCount())
	assert.Equal(t, int64(5), r.RequestCountByPath("/v2/"))
}

// TestMockRegistry_BlobUploadRoundTrip verifies POST start + PUT complete.
func TestMockRegistry_BlobUploadRoundTrip(t *testing.T) {
	r := NewMockRegistry()
	defer r.Close()

	content := []byte("upload-me-please")
	dgst := godigest.FromBytes(content).String()

	// POST /v2/myrepo/blobs/uploads/  — start upload
	resp, err := http.Post(baseURL(r)+"/v2/myrepo/blobs/uploads/", "", nil)
	require.NoError(t, err)
	resp.Body.Close()
	require.Equal(t, http.StatusAccepted, resp.StatusCode)

	location := resp.Header.Get("Location")
	require.NotEmpty(t, location, "Location header must be set")

	// PUT <location>?digest=<digest>  — complete upload
	putURL := fmt.Sprintf("%s%s?digest=%s", baseURL(r), location, dgst)
	putReq, err := http.NewRequest(http.MethodPut, putURL, nil)
	require.NoError(t, err)

	// Send body in the PUT (single-shot upload).
	putReq.Body = io.NopCloser(newReader(content))
	putReq.ContentLength = int64(len(content))

	resp, err = http.DefaultClient.Do(putReq)
	require.NoError(t, err)
	resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	// Verify blob is stored.
	assert.True(t, r.BlobExists(dgst))
	stored, ok := r.GetBlob(dgst)
	require.True(t, ok)
	assert.Equal(t, content, stored)
}

// newReader wraps a byte slice in an io.Reader without importing bytes at the
// test-package level (bytes is already imported in the main file).
func newReader(b []byte) io.Reader {
	return &bytesReader{data: b}
}

type bytesReader struct {
	data []byte
	pos  int
}

func (br *bytesReader) Read(p []byte) (int, error) {
	if br.pos >= len(br.data) {
		return 0, io.EOF
	}
	n := copy(p, br.data[br.pos:])
	br.pos += n
	return n, nil
}
