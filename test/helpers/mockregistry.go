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

// Package helpers provides shared test utilities for integration tests.
package helpers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	godigest "github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

const (
	// manifestMediaType is the OCI image manifest media type.
	manifestMediaType = ocispec.MediaTypeImageManifest
)

// FaultConfig describes fault injection parameters for the mock registry.
// Path-specific faults (via PathFaults) override the global config when the
// request path has a matching suffix.
type FaultConfig struct {
	// LatencyPerRequest is an artificial delay applied before processing.
	LatencyPerRequest time.Duration

	// FailAfterNBytes causes the connection to be force-closed after writing
	// this many bytes of the response body.  0 = disabled.
	FailAfterNBytes int64

	// StatusCodeOverride forces every matching request to return this HTTP
	// status code.  0 = disabled.
	StatusCodeOverride int

	// FailOnNthRequest makes the first N requests (1-based) fail with 500.
	// Request N+1 and onwards succeed normally.  0 = disabled.
	FailOnNthRequest int

	// PathFaults maps a path suffix to a FaultConfig that overrides the global
	// config for requests whose URL path ends with that suffix.
	PathFaults map[string]*FaultConfig
}

// MockRegistry is an in-memory OCI Distribution API server backed by
// httptest.Server.  It is safe for concurrent use.
type MockRegistry struct {
	server *httptest.Server

	mu        sync.RWMutex
	manifests map[string][]byte // key: "repo:ref"
	blobs     map[string][]byte // key: digest string
	// pendingUploads tracks in-progress blob uploads keyed by upload UUID.
	pendingUploads map[string]*bytes.Buffer

	fault        *FaultConfig
	faultMu      sync.Mutex
	requestCount int64 // accessed atomically
	// pathCounts maps a path suffix to an atomic counter stored as *int64.
	pathCounts sync.Map

	// failCounter tracks how many requests have been seen for FailOnNthRequest.
	failCounter int64 // accessed atomically
}

// NewMockRegistry creates and starts a new mock OCI registry server.
func NewMockRegistry() *MockRegistry {
	r := &MockRegistry{
		manifests:      make(map[string][]byte),
		blobs:          make(map[string][]byte),
		pendingUploads: make(map[string]*bytes.Buffer),
	}
	r.server = httptest.NewServer(http.HandlerFunc(r.handleRequest))
	return r
}

// WithFault sets global fault injection config and returns the registry for
// chaining.
func (r *MockRegistry) WithFault(f *FaultConfig) *MockRegistry {
	r.faultMu.Lock()
	r.fault = f
	r.faultMu.Unlock()
	return r
}

// AddManifest serialises manifest and stores it under "repo:ref".
func (r *MockRegistry) AddManifest(ref string, manifest ocispec.Manifest) *MockRegistry {
	data, err := json.Marshal(manifest)
	if err != nil {
		panic(fmt.Sprintf("mockregistry: failed to marshal manifest: %v", err))
	}
	r.mu.Lock()
	r.manifests[ref] = data
	r.mu.Unlock()
	return r
}

// AddBlob stores content and returns its OCI digest string.
func (r *MockRegistry) AddBlob(content []byte) string {
	dgst := godigest.FromBytes(content).String()
	r.mu.Lock()
	r.blobs[dgst] = content
	r.mu.Unlock()
	return dgst
}

// AddBlobWithDigest stores content under a caller-supplied digest string.
// This is useful for corruption tests where the stored digest does not match
// the actual content.
func (r *MockRegistry) AddBlobWithDigest(digest string, content []byte) *MockRegistry {
	r.mu.Lock()
	r.blobs[digest] = content
	r.mu.Unlock()
	return r
}

// Host returns "127.0.0.1:PORT" (no scheme) suitable for use as an OCI
// registry reference host.
func (r *MockRegistry) Host() string {
	return r.server.Listener.Addr().(*net.TCPAddr).String()
}

// Close shuts down the underlying httptest.Server.
func (r *MockRegistry) Close() {
	r.server.Close()
}

// RequestCount returns the total number of requests received.
func (r *MockRegistry) RequestCount() int64 {
	return atomic.LoadInt64(&r.requestCount)
}

// RequestCountByPath returns the number of requests whose URL path ends with
// pathSuffix.
func (r *MockRegistry) RequestCountByPath(pathSuffix string) int64 {
	if v, ok := r.pathCounts.Load(pathSuffix); ok {
		return atomic.LoadInt64(v.(*int64))
	}
	return 0
}

// BlobExists reports whether a blob with the given digest is stored.
func (r *MockRegistry) BlobExists(digest string) bool {
	r.mu.RLock()
	_, ok := r.blobs[digest]
	r.mu.RUnlock()
	return ok
}

// GetBlob returns the content for the given digest, and whether it was found.
func (r *MockRegistry) GetBlob(digest string) ([]byte, bool) {
	r.mu.RLock()
	data, ok := r.blobs[digest]
	r.mu.RUnlock()
	return data, ok
}

// ManifestExists reports whether a manifest stored under ref exists.
func (r *MockRegistry) ManifestExists(ref string) bool {
	r.mu.RLock()
	_, ok := r.manifests[ref]
	r.mu.RUnlock()
	return ok
}

// -----------------------------------------------------------------------
// Internal helpers
// -----------------------------------------------------------------------

// effectiveFault returns the FaultConfig to apply for path, considering
// per-path overrides.  Returns nil if no fault is configured.
func (r *MockRegistry) effectiveFault(path string) *FaultConfig {
	r.faultMu.Lock()
	f := r.fault
	r.faultMu.Unlock()

	if f == nil {
		return nil
	}
	for suffix, pf := range f.PathFaults {
		if strings.HasSuffix(path, suffix) {
			return pf
		}
	}
	return f
}

// incrementPathCount bumps all registered path-suffix counters that match.
func (r *MockRegistry) incrementPathCount(path string) {
	r.faultMu.Lock()
	f := r.fault
	r.faultMu.Unlock()

	if f == nil {
		return
	}
	for suffix := range f.PathFaults {
		if strings.HasSuffix(path, suffix) {
			r.bumpPathCounter(suffix)
		}
	}
}

func (r *MockRegistry) bumpPathCounter(suffix string) {
	var zero int64
	actual, _ := r.pathCounts.LoadOrStore(suffix, &zero)
	atomic.AddInt64(actual.(*int64), 1)
}

// handleRequest is the top-level HTTP handler.
func (r *MockRegistry) handleRequest(w http.ResponseWriter, req *http.Request) {
	atomic.AddInt64(&r.requestCount, 1)
	r.incrementPathCount(req.URL.Path)

	f := r.effectiveFault(req.URL.Path)

	// Apply latency.
	if f != nil && f.LatencyPerRequest > 0 {
		time.Sleep(f.LatencyPerRequest)
	}

	// Apply FailOnNthRequest.
	if f != nil && f.FailOnNthRequest > 0 {
		n := atomic.AddInt64(&r.failCounter, 1)
		if n <= int64(f.FailOnNthRequest) {
			http.Error(w, "fault: fail on nth request", http.StatusInternalServerError)
			return
		}
	}

	// Apply StatusCodeOverride.
	if f != nil && f.StatusCodeOverride != 0 {
		w.WriteHeader(f.StatusCodeOverride)
		return
	}

	r.route(w, req, f)
}

// route dispatches to the appropriate OCI endpoint handler.
func (r *MockRegistry) route(w http.ResponseWriter, req *http.Request, f *FaultConfig) {
	path := req.URL.Path

	// GET /v2/ — registry ping
	if path == "/v2/" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// /v2/<name>/manifests/<ref>
	if idx := strings.Index(path, "/manifests/"); idx != -1 {
		prefix := path[:idx]              // /v2/<name>
		ref := path[idx+len("/manifests/"):]
		name := strings.TrimPrefix(prefix, "/v2/")
		r.handleManifest(w, req, name, ref, f)
		return
	}

	// /v2/<name>/blobs/uploads/<uuid> or /v2/<name>/blobs/uploads/
	if strings.Contains(path, "/blobs/uploads") {
		r.handleBlobUpload(w, req, path, f)
		return
	}

	// /v2/<name>/blobs/<digest>
	if idx := strings.Index(path, "/blobs/"); idx != -1 {
		digest := path[idx+len("/blobs/"):]
		r.handleBlob(w, req, digest, f)
		return
	}

	http.Error(w, "not found", http.StatusNotFound)
}

// handleManifest handles HEAD/GET/PUT on /v2/<name>/manifests/<ref>.
func (r *MockRegistry) handleManifest(w http.ResponseWriter, req *http.Request, name, ref string, f *FaultConfig) {
	key := name + ":" + ref

	switch req.Method {
	case http.MethodHead, http.MethodGet:
		r.mu.RLock()
		data, ok := r.manifests[key]
		r.mu.RUnlock()

		if !ok {
			http.Error(w, "manifest not found", http.StatusNotFound)
			return
		}

		dgst := godigest.FromBytes(data).String()
		w.Header().Set("Content-Type", manifestMediaType)
		w.Header().Set("Docker-Content-Digest", dgst)
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(data)))
		if req.Method == http.MethodHead {
			w.WriteHeader(http.StatusOK)
			return
		}
		writeBody(w, data, f)

	case http.MethodPut:
		buf := new(bytes.Buffer)
		if _, err := buf.ReadFrom(req.Body); err != nil {
			http.Error(w, "read error", http.StatusInternalServerError)
			return
		}
		data := buf.Bytes()
		dgst := godigest.FromBytes(data).String()

		r.mu.Lock()
		r.manifests[key] = data
		r.mu.Unlock()

		w.Header().Set("Docker-Content-Digest", dgst)
		w.WriteHeader(http.StatusCreated)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleBlob handles HEAD/GET on /v2/<name>/blobs/<digest>.
func (r *MockRegistry) handleBlob(w http.ResponseWriter, req *http.Request, digest string, f *FaultConfig) {
	switch req.Method {
	case http.MethodHead, http.MethodGet:
		r.mu.RLock()
		data, ok := r.blobs[digest]
		r.mu.RUnlock()

		if !ok {
			http.Error(w, "blob not found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Docker-Content-Digest", digest)
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(data)))
		if req.Method == http.MethodHead {
			w.WriteHeader(http.StatusOK)
			return
		}
		writeBody(w, data, f)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleBlobUpload handles the multi-step blob upload protocol.
func (r *MockRegistry) handleBlobUpload(w http.ResponseWriter, req *http.Request, path string, f *FaultConfig) {
	// POST /v2/<name>/blobs/uploads/ — initiate upload
	if req.Method == http.MethodPost && strings.HasSuffix(path, "/blobs/uploads/") {
		uuid := godigest.FromString(fmt.Sprintf("%d", time.Now().UnixNano())).Hex()[:16]
		r.mu.Lock()
		r.pendingUploads[uuid] = new(bytes.Buffer)
		r.mu.Unlock()

		// Extract name from path: /v2/<name>/blobs/uploads/
		name := strings.TrimPrefix(strings.TrimSuffix(path, "/blobs/uploads/"), "/v2/")
		location := fmt.Sprintf("/v2/%s/blobs/uploads/%s", name, uuid)
		w.Header().Set("Location", location)
		w.Header().Set("Docker-Upload-UUID", uuid)
		w.WriteHeader(http.StatusAccepted)
		return
	}

	// PATCH /v2/<name>/blobs/uploads/<uuid> — chunked upload
	if req.Method == http.MethodPatch {
		uuid := uploadUUID(path)
		r.mu.Lock()
		buf, ok := r.pendingUploads[uuid]
		if !ok {
			r.mu.Unlock()
			http.Error(w, "upload not found", http.StatusNotFound)
			return
		}
		if _, err := buf.ReadFrom(req.Body); err != nil {
			r.mu.Unlock()
			http.Error(w, "read error", http.StatusInternalServerError)
			return
		}
		r.mu.Unlock()
		w.Header().Set("Location", path)
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// PUT /v2/<name>/blobs/uploads/<uuid>?digest=<digest> — complete upload
	if req.Method == http.MethodPut {
		uuid := uploadUUID(path)
		dgst := req.URL.Query().Get("digest")
		if dgst == "" {
			http.Error(w, "missing digest", http.StatusBadRequest)
			return
		}

		r.mu.Lock()
		buf, ok := r.pendingUploads[uuid]
		if !ok {
			r.mu.Unlock()
			http.Error(w, "upload not found", http.StatusNotFound)
			return
		}
		// Append any final body bytes.
		if req.Body != nil {
			if _, err := buf.ReadFrom(req.Body); err != nil {
				r.mu.Unlock()
				http.Error(w, "read error", http.StatusInternalServerError)
				return
			}
		}
		data := buf.Bytes()
		delete(r.pendingUploads, uuid)
		r.blobs[dgst] = append([]byte(nil), data...)
		r.mu.Unlock()

		w.Header().Set("Docker-Content-Digest", dgst)
		w.WriteHeader(http.StatusCreated)
		return
	}

	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
}

// uploadUUID extracts the UUID from a blob upload path such as
// /v2/<name>/blobs/uploads/<uuid>.
func uploadUUID(path string) string {
	parts := strings.Split(path, "/blobs/uploads/")
	if len(parts) < 2 {
		return ""
	}
	return parts[1]
}

// writeBody writes data to w, honouring FailAfterNBytes by hijacking the
// connection mid-write when the limit is reached.
func writeBody(w http.ResponseWriter, data []byte, f *FaultConfig) {
	if f == nil || f.FailAfterNBytes <= 0 {
		_, _ = w.Write(data)
		return
	}

	limit := f.FailAfterNBytes
	if limit > int64(len(data)) {
		limit = int64(len(data))
	}

	// Write the permitted bytes first.
	_, _ = w.Write(data[:limit])

	// Hijack the connection and close it abruptly.
	hj, ok := w.(http.Hijacker)
	if !ok {
		return
	}
	conn, _, err := hj.Hijack()
	if err != nil {
		return
	}
	_ = conn.Close()
}
