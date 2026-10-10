/*
 *     Copyright 2025 The ModelPack Authors
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
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	modelspec "github.com/modelpack/model-spec/specs-go/v1"
	godigest "github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/modelpack/modctl/pkg/config"
)

// recordingFetchHook tracks hook invocations and can request specific layers
// to be skipped by digest.
type recordingFetchHook struct {
	mu          sync.Mutex
	skipDigests map[string]bool
	beforeCount int32
	afterCalls  []afterFetchCall
}

type afterFetchCall struct {
	digest  string
	skipped bool
	err     error
}

func (r *recordingFetchHook) BeforePullLayer(desc ocispec.Descriptor, _ ocispec.Manifest) bool {
	atomic.AddInt32(&r.beforeCount, 1)
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.skipDigests[desc.Digest.String()]
}

func (r *recordingFetchHook) AfterPullLayer(desc ocispec.Descriptor, skipped bool, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.afterCalls = append(r.afterCalls, afterFetchCall{
		digest:  desc.Digest.String(),
		skipped: skipped,
		err:     err,
	})
}

// startFetchTestServer spins up an HTTP server that serves a manifest with
// two layers and tracks how many times each blob is requested.
func startFetchTestServer(t *testing.T) (server *httptest.Server, file1Digest, file2Digest godigest.Digest, blobHits map[string]*int32) {
	t.Helper()

	const (
		file1Content = "file1 content..."
		file2Content = "file2 content..."
	)
	file1Digest = godigest.FromString(file1Content)
	file2Digest = godigest.FromString(file2Content)

	hits := map[string]*int32{
		file1Digest.String(): new(int32),
		file2Digest.String(): new(int32),
	}

	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v2/":
			w.WriteHeader(http.StatusOK)
		case "/v2/test/model/manifests/latest":
			manifest := ocispec.Manifest{
				Layers: []ocispec.Descriptor{
					{
						MediaType: "application/octet-stream.raw",
						Digest:    file1Digest,
						Size:      int64(len(file1Content)),
						Annotations: map[string]string{
							modelspec.AnnotationFilepath: "file1.txt",
						},
					},
					{
						MediaType: "application/octet-stream.raw",
						Digest:    file2Digest,
						Size:      int64(len(file2Content)),
						Annotations: map[string]string{
							modelspec.AnnotationFilepath: "file2.txt",
						},
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			require.NoError(t, json.NewEncoder(w).Encode(manifest))
		case fmt.Sprintf("/v2/test/model/blobs/%s", file1Digest):
			atomic.AddInt32(hits[file1Digest.String()], 1)
			_, err := w.Write([]byte(file1Content))
			require.NoError(t, err)
		case fmt.Sprintf("/v2/test/model/blobs/%s", file2Digest):
			atomic.AddInt32(hits[file2Digest.String()], 1)
			_, err := w.Write([]byte(file2Content))
			require.NoError(t, err)
		default:
			t.Logf("Unexpected request to %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))

	return server, file1Digest, file2Digest, hits
}

// TestFetch_HookSkipShortCircuitsLayer verifies that returning skip=true from
// BeforePullLayer prevents the blob from being downloaded and that
// AfterPullLayer is still invoked with skipped=true.
func TestFetch_HookSkipShortCircuitsLayer(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "fetch-hook-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	server, file1Digest, file2Digest, hits := startFetchTestServer(t)
	defer server.Close()

	hook := &recordingFetchHook{
		skipDigests: map[string]bool{file1Digest.String(): true},
	}

	b := &backend{}
	url := strings.TrimPrefix(server.URL, "http://")
	cfg := &config.Fetch{
		Output:      tempDir,
		Patterns:    []string{"*.txt"},
		PlainHTTP:   true,
		Concurrency: 2,
		Hooks:       hook,
	}

	require.NoError(t, b.Fetch(context.Background(), url+"/test/model:latest", cfg))

	// file1 must NOT have been downloaded; file2 must have been.
	assert.Equal(t, int32(0), atomic.LoadInt32(hits[file1Digest.String()]),
		"skipped layer should not be fetched from remote")
	assert.Equal(t, int32(1), atomic.LoadInt32(hits[file2Digest.String()]),
		"non-skipped layer should be fetched once")

	// BeforePullLayer fires for both layers exactly once (no retries on success).
	assert.Equal(t, int32(2), atomic.LoadInt32(&hook.beforeCount))

	// AfterPullLayer must be invoked for both layers, with proper skipped flag.
	hook.mu.Lock()
	defer hook.mu.Unlock()
	require.Len(t, hook.afterCalls, 2)

	byDigest := map[string]afterFetchCall{}
	for _, c := range hook.afterCalls {
		byDigest[c.digest] = c
	}
	assert.True(t, byDigest[file1Digest.String()].skipped, "file1 should be marked skipped")
	assert.NoError(t, byDigest[file1Digest.String()].err)
	assert.False(t, byDigest[file2Digest.String()].skipped, "file2 should not be marked skipped")
	assert.NoError(t, byDigest[file2Digest.String()].err)
}
