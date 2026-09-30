# Integration Tests Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add ~35 integration tests across modctl covering 8 dimensions (functional correctness, network errors, resource leaks, concurrency safety, stress, data integrity, graceful shutdown, idempotency) without changing any functional code.

**Architecture:** Shared mock OCI registry (`test/helpers/`) with fault injection serves as remote; mock Storage from `test/mocks/storage/` serves as local. Tests in `*_integration_test.go` files alongside existing code. Known bugs use reverse assertions (pass today, fail when fixed).

**Tech Stack:** Go stdlib `net/http/httptest`, `github.com/stretchr/testify` (already in go.mod), `test/mocks/storage/storage.go` (already generated)

**Spec:** `docs/superpowers/specs/2026-04-07-integration-tests-design.md`

**Known bug issues:** #491 (push ReadCloser leak), #492 (splitReader goroutine leak), #493 (disableProgress data race), #494 (auth error retry)

---

## File Structure

```
test/helpers/
  mockregistry.go              # CREATE — shared mock OCI registry with fault injection
  mockregistry_test.go         # CREATE — self-tests for mock registry
  tracking.go                  # CREATE — TrackingReadCloser utility

pkg/modelfile/
  modelfile_integration_test.go   # CREATE — ExcludePatterns tests (2 tests)
  modelfile_stress_test.go        # CREATE — //go:build stress (2 tests)

pkg/backend/
  pull_integration_test.go        # CREATE — pull tests across dims 1,2,4,6,7,8 (10 tests)
  push_integration_test.go        # CREATE — push tests across dims 1,2,3,6,7,8 (9 tests)
  pull_slowtest_test.go           # CREATE — //go:build slowtest (4 tests)
  push_slowtest_test.go           # CREATE — //go:build slowtest (1 test)
  pull_stress_test.go             # CREATE — //go:build stress (2 tests)

internal/pb/
  pb_integration_test.go          # CREATE — concurrency tests (2 tests)

cmd/modelfile/
  generate_integration_test.go    # CREATE — CLI generate tests (3 tests)

cmd/
  cli_integration_test.go         # CREATE — config boundary tests (3 tests)
```

---

### Task 1: Mock OCI Registry

**Files:**
- Create: `test/helpers/mockregistry.go`
- Create: `test/helpers/mockregistry_test.go`

This is the foundation for all backend integration tests. Follows the pattern in `pkg/backend/fetch_test.go:54-94` but as a reusable, fault-injectable helper.

- [ ] **Step 1: Create `test/helpers/mockregistry.go`**

```go
package helpers

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	godigest "github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

// FaultConfig controls fault injection for the mock registry.
type FaultConfig struct {
	LatencyPerRequest  time.Duration          // per-request delay
	FailAfterNBytes    int64                  // disconnect after N bytes written
	StatusCodeOverride int                    // force HTTP status code
	FailOnNthRequest   int                    // fail Nth request (1-based), 0 = disabled
	PathFaults         map[string]*FaultConfig // per-path override (matched by suffix)
}

// MockRegistry is a test OCI registry backed by httptest.Server.
type MockRegistry struct {
	server    *httptest.Server
	mu        sync.RWMutex
	manifests map[string][]byte // "repo:ref" -> manifest JSON
	blobs     map[string][]byte // digest string -> content

	faults       *FaultConfig
	requestCount atomic.Int64
	pathCounts   sync.Map // path string -> *atomic.Int64

	// uploads tracks in-progress blob uploads
	uploads   map[string][]byte // upload UUID -> accumulated data
	uploadsMu sync.Mutex
	uploadSeq atomic.Int64
}

// NewMockRegistry creates a mock OCI registry with no faults.
func NewMockRegistry() *MockRegistry {
	mr := &MockRegistry{
		manifests: make(map[string][]byte),
		blobs:     make(map[string][]byte),
		uploads:   make(map[string][]byte),
	}
	mr.server = httptest.NewServer(http.HandlerFunc(mr.handler))
	return mr
}

// WithFault sets the fault configuration. Call before making requests.
func (mr *MockRegistry) WithFault(f *FaultConfig) *MockRegistry {
	mr.faults = f
	return mr
}

// AddManifest pre-populates a manifest. ref is "repo:tag".
func (mr *MockRegistry) AddManifest(ref string, manifest ocispec.Manifest) *MockRegistry {
	data, _ := json.Marshal(manifest)
	mr.mu.Lock()
	mr.manifests[ref] = data
	mr.mu.Unlock()
	return mr
}

// AddBlob pre-populates a blob by its content. Returns the digest.
func (mr *MockRegistry) AddBlob(content []byte) string {
	dgst := godigest.FromBytes(content)
	mr.mu.Lock()
	mr.blobs[dgst.String()] = content
	mr.mu.Unlock()
	return dgst.String()
}

// Host returns the registry host (e.g., "127.0.0.1:PORT") without scheme.
func (mr *MockRegistry) Host() string {
	return strings.TrimPrefix(mr.server.URL, "http://")
}

// Close shuts down the mock server.
func (mr *MockRegistry) Close() {
	mr.server.Close()
}

// RequestCount returns total requests received.
func (mr *MockRegistry) RequestCount() int64 {
	return mr.requestCount.Load()
}

// RequestCountByPath returns request count for a specific path suffix.
func (mr *MockRegistry) RequestCountByPath(pathSuffix string) int64 {
	if val, ok := mr.pathCounts.Load(pathSuffix); ok {
		return val.(*atomic.Int64).Load()
	}
	return 0
}

// BlobExists checks if a blob was received (useful for push verification).
func (mr *MockRegistry) BlobExists(digest string) bool {
	mr.mu.RLock()
	defer mr.mu.RUnlock()
	_, ok := mr.blobs[digest]
	return ok
}

// GetBlob returns blob content (useful for push integrity verification).
func (mr *MockRegistry) GetBlob(digest string) ([]byte, bool) {
	mr.mu.RLock()
	defer mr.mu.RUnlock()
	data, ok := mr.blobs[digest]
	return data, ok
}

// ManifestExists checks if a manifest ref was received.
func (mr *MockRegistry) ManifestExists(ref string) bool {
	mr.mu.RLock()
	defer mr.mu.RUnlock()
	_, ok := mr.manifests[ref]
	return ok
}

func (mr *MockRegistry) handler(w http.ResponseWriter, r *http.Request) {
	mr.requestCount.Add(1)
	mr.trackPath(r.URL.Path)

	// Resolve fault config (path-specific overrides global).
	fault := mr.resolveFault(r.URL.Path)

	// Apply fault: check Nth request failure.
	if fault != nil && fault.FailOnNthRequest > 0 {
		count := mr.requestCount.Load()
		if int(count) <= fault.FailOnNthRequest {
			http.Error(w, "injected fault", http.StatusInternalServerError)
			return
		}
	}

	// Apply fault: latency.
	if fault != nil && fault.LatencyPerRequest > 0 {
		time.Sleep(fault.LatencyPerRequest)
	}

	// Apply fault: status code override.
	if fault != nil && fault.StatusCodeOverride > 0 {
		http.Error(w, "injected status", fault.StatusCodeOverride)
		return
	}

	// Route the request.
	path := r.URL.Path
	switch {
	case path == "/v2/" || path == "/v2":
		w.WriteHeader(http.StatusOK)

	case strings.Contains(path, "/manifests/"):
		mr.handleManifest(w, r, fault)

	case strings.Contains(path, "/blobs/uploads/"):
		mr.handleBlobUpload(w, r)

	case strings.Contains(path, "/blobs/"):
		mr.handleBlob(w, r, fault)

	default:
		http.NotFound(w, r)
	}
}

func (mr *MockRegistry) handleManifest(w http.ResponseWriter, r *http.Request, fault *FaultConfig) {
	// Parse: /v2/<repo>/manifests/<ref>
	// repo can contain slashes, ref is the last segment after /manifests/
	parts := strings.SplitN(r.URL.Path, "/manifests/", 2)
	if len(parts) != 2 {
		http.NotFound(w, r)
		return
	}
	repo := strings.TrimPrefix(parts[0], "/v2/")
	ref := parts[1]
	key := repo + ":" + ref

	switch r.Method {
	case http.MethodGet:
		mr.mu.RLock()
		data, ok := mr.manifests[key]
		mr.mu.RUnlock()
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", ocispec.MediaTypeImageManifest)
		w.Header().Set("Docker-Content-Digest", godigest.FromBytes(data).String())
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(data)))
		mr.writeWithFault(w, data, fault)

	case http.MethodHead:
		mr.mu.RLock()
		data, ok := mr.manifests[key]
		mr.mu.RUnlock()
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", ocispec.MediaTypeImageManifest)
		w.Header().Set("Docker-Content-Digest", godigest.FromBytes(data).String())
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(data)))
		w.WriteHeader(http.StatusOK)

	case http.MethodPut:
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		mr.mu.Lock()
		mr.manifests[key] = body
		// Also store by digest for HEAD lookups
		dgst := godigest.FromBytes(body)
		mr.manifests[repo+":"+dgst.String()] = body
		mr.mu.Unlock()
		w.Header().Set("Docker-Content-Digest", dgst.String())
		w.WriteHeader(http.StatusCreated)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (mr *MockRegistry) handleBlob(w http.ResponseWriter, r *http.Request, fault *FaultConfig) {
	// Parse: /v2/<repo>/blobs/<digest>
	parts := strings.SplitN(r.URL.Path, "/blobs/", 2)
	if len(parts) != 2 {
		http.NotFound(w, r)
		return
	}
	digest := parts[1]

	mr.mu.RLock()
	data, ok := mr.blobs[digest]
	mr.mu.RUnlock()

	switch r.Method {
	case http.MethodGet:
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(data)))
		w.Header().Set("Docker-Content-Digest", digest)
		mr.writeWithFault(w, data, fault)

	case http.MethodHead:
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(data)))
		w.Header().Set("Docker-Content-Digest", digest)
		w.WriteHeader(http.StatusOK)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (mr *MockRegistry) handleBlobUpload(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		// Start upload — return a UUID location.
		uuid := fmt.Sprintf("upload-%d", mr.uploadSeq.Add(1))
		mr.uploadsMu.Lock()
		mr.uploads[uuid] = nil
		mr.uploadsMu.Unlock()

		// Parse repo from path for location header.
		parts := strings.SplitN(r.URL.Path, "/blobs/uploads/", 2)
		repo := strings.TrimPrefix(parts[0], "/v2/")
		location := fmt.Sprintf("/v2/%s/blobs/uploads/%s", repo, uuid)
		w.Header().Set("Location", location)
		w.Header().Set("Docker-Upload-UUID", uuid)
		w.WriteHeader(http.StatusAccepted)

	case http.MethodPut:
		// Complete upload — read body, store as blob.
		// Path: /v2/<repo>/blobs/uploads/<uuid>?digest=<digest>
		parts := strings.SplitN(r.URL.Path, "/blobs/uploads/", 2)
		if len(parts) != 2 {
			http.Error(w, "bad upload path", http.StatusBadRequest)
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		digest := r.URL.Query().Get("digest")
		if digest == "" {
			// Compute digest from body.
			h := sha256.Sum256(body)
			digest = fmt.Sprintf("sha256:%x", h)
		}

		mr.mu.Lock()
		mr.blobs[digest] = body
		mr.mu.Unlock()

		w.Header().Set("Docker-Content-Digest", digest)
		w.WriteHeader(http.StatusCreated)

	case http.MethodPatch:
		// Chunked upload — accumulate data.
		parts := strings.SplitN(r.URL.Path, "/blobs/uploads/", 2)
		if len(parts) != 2 {
			http.Error(w, "bad upload path", http.StatusBadRequest)
			return
		}
		uuid := parts[1]

		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		mr.uploadsMu.Lock()
		mr.uploads[uuid] = append(mr.uploads[uuid], body...)
		mr.uploadsMu.Unlock()

		w.Header().Set("Location", r.URL.Path)
		w.WriteHeader(http.StatusAccepted)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (mr *MockRegistry) writeWithFault(w http.ResponseWriter, data []byte, fault *FaultConfig) {
	if fault != nil && fault.FailAfterNBytes > 0 && fault.FailAfterNBytes < int64(len(data)) {
		// Write partial data then close connection.
		w.Write(data[:fault.FailAfterNBytes])
		// Hijack to force-close the connection.
		if hj, ok := w.(http.Hijacker); ok {
			conn, _, _ := hj.Hijack()
			if conn != nil {
				conn.Close()
			}
		}
		return
	}
	w.Write(data)
}

func (mr *MockRegistry) resolveFault(path string) *FaultConfig {
	if mr.faults == nil {
		return nil
	}
	// Check path-specific faults first.
	if mr.faults.PathFaults != nil {
		for suffix, pf := range mr.faults.PathFaults {
			if strings.HasSuffix(path, suffix) {
				return pf
			}
		}
	}
	return mr.faults
}

func (mr *MockRegistry) trackPath(path string) {
	val, _ := mr.pathCounts.LoadOrStore(path, &atomic.Int64{})
	val.(*atomic.Int64).Add(1)
}
```

- [ ] **Step 2: Write self-tests in `test/helpers/mockregistry_test.go`**

```go
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

func TestMockRegistry_PingAndBlobRoundTrip(t *testing.T) {
	mr := NewMockRegistry()
	defer mr.Close()

	// Ping.
	resp, err := http.Get(fmt.Sprintf("http://%s/v2/", mr.Host()))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	// Add and fetch blob.
	content := []byte("hello blob")
	digest := mr.AddBlob(content)

	resp, err = http.Get(fmt.Sprintf("http://%s/v2/test/repo/blobs/%s", mr.Host(), digest))
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	assert.Equal(t, content, body)
}

func TestMockRegistry_ManifestRoundTrip(t *testing.T) {
	mr := NewMockRegistry()
	defer mr.Close()

	manifest := ocispec.Manifest{
		MediaType: ocispec.MediaTypeImageManifest,
		Layers: []ocispec.Descriptor{
			{Digest: godigest.FromString("layer1"), Size: 6},
		},
	}
	mr.AddManifest("test/repo:latest", manifest)

	resp, err := http.Get(fmt.Sprintf("http://%s/v2/test/repo/manifests/latest", mr.Host()))
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var got ocispec.Manifest
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&got))
	assert.Len(t, got.Layers, 1)
}

func TestMockRegistry_FaultStatusCodeOverride(t *testing.T) {
	mr := NewMockRegistry()
	defer mr.Close()
	mr.WithFault(&FaultConfig{StatusCodeOverride: http.StatusInternalServerError})

	resp, err := http.Get(fmt.Sprintf("http://%s/v2/", mr.Host()))
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
}

func TestMockRegistry_FaultFailOnNthRequest(t *testing.T) {
	mr := NewMockRegistry()
	defer mr.Close()
	mr.WithFault(&FaultConfig{FailOnNthRequest: 2})

	url := fmt.Sprintf("http://%s/v2/", mr.Host())

	// Requests 1 and 2 should fail (count <= N).
	resp, _ := http.Get(url)
	resp.Body.Close()
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	resp, _ = http.Get(url)
	resp.Body.Close()
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	// Request 3 should succeed.
	resp, _ = http.Get(url)
	resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestMockRegistry_FaultLatency(t *testing.T) {
	mr := NewMockRegistry()
	defer mr.Close()
	mr.WithFault(&FaultConfig{LatencyPerRequest: 100 * time.Millisecond})

	start := time.Now()
	resp, err := http.Get(fmt.Sprintf("http://%s/v2/", mr.Host()))
	require.NoError(t, err)
	resp.Body.Close()
	assert.GreaterOrEqual(t, time.Since(start), 80*time.Millisecond)
}

func TestMockRegistry_RequestCounting(t *testing.T) {
	mr := NewMockRegistry()
	defer mr.Close()

	url := fmt.Sprintf("http://%s/v2/", mr.Host())
	for i := 0; i < 3; i++ {
		resp, _ := http.Get(url)
		resp.Body.Close()
	}

	assert.Equal(t, int64(3), mr.RequestCount())
}

func TestMockRegistry_BlobUploadRoundTrip(t *testing.T) {
	mr := NewMockRegistry()
	defer mr.Close()

	// POST to start upload.
	resp, err := http.Post(
		fmt.Sprintf("http://%s/v2/test/repo/blobs/uploads/", mr.Host()),
		"application/octet-stream", nil,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusAccepted, resp.StatusCode)
	location := resp.Header.Get("Location")
	resp.Body.Close()
	require.NotEmpty(t, location)

	// PUT to complete upload.
	content := []byte("uploaded blob data")
	dgst := godigest.FromBytes(content)
	uploadURL := fmt.Sprintf("http://%s%s?digest=%s", mr.Host(), location, dgst.String())

	req, _ := http.NewRequest(http.MethodPut, uploadURL, io.NopCloser(
		io.NewSectionReader(
			readerAt(content), 0, int64(len(content)),
		),
	))
	// Simpler: just use bytes.NewReader
	req, _ = http.NewRequest(http.MethodPut, uploadURL, bytesReader(content))
	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	// Verify blob exists.
	assert.True(t, mr.BlobExists(dgst.String()))
	data, ok := mr.GetBlob(dgst.String())
	assert.True(t, ok)
	assert.Equal(t, content, data)
}

// bytesReader is a helper to create a bytes.Reader.
func bytesReader(data []byte) io.Reader {
	return io.NopCloser(ioReader(data))
}

// These are defined to avoid importing bytes in the test
// (the real implementation will use bytes.NewReader)
type readerAtImpl struct{ data []byte }

func (r readerAtImpl) ReadAt(p []byte, off int64) (int, error) {
	copy(p, r.data[off:])
	n := len(r.data) - int(off)
	if n > len(p) {
		n = len(p)
	}
	return n, io.EOF
}
func readerAt(data []byte) io.ReaderAt   { return readerAtImpl{data} }
func ioReader(data []byte) io.Reader {
	return &ioReaderImpl{data: data}
}

type ioReaderImpl struct {
	data []byte
	pos  int
}

func (r *ioReaderImpl) Read(p []byte) (int, error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n := copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}
```

Note: The upload test uses some helper functions to avoid unnecessary imports. The actual implementation should use `bytes.NewReader` directly. Clean up the helper functions and use `bytes.NewReader` and `bytes.NewBuffer` instead.

- [ ] **Step 3: Run self-tests**

Run: `cd /Users/zhaochen/modelpack/modctl/.claude/worktrees/silly-leaping-pelican && go test ./test/helpers/ -v -count=1`
Expected: All tests PASS.

- [ ] **Step 4: Commit**

```bash
git add test/helpers/mockregistry.go test/helpers/mockregistry_test.go
git commit -s -m "test: add shared mock OCI registry with fault injection"
```

---

### Task 2: Resource Tracking Utilities

**Files:**
- Create: `test/helpers/tracking.go`

- [ ] **Step 1: Create `test/helpers/tracking.go`**

```go
package helpers

import (
	"io"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TrackingReadCloser wraps an io.ReadCloser and records whether Close() was called.
type TrackingReadCloser struct {
	io.ReadCloser
	closed atomic.Bool
}

// NewTrackingReadCloser wraps rc with close-tracking.
func NewTrackingReadCloser(rc io.ReadCloser) *TrackingReadCloser {
	return &TrackingReadCloser{ReadCloser: rc}
}

// Close marks the closer as closed and delegates to the underlying closer.
func (t *TrackingReadCloser) Close() error {
	t.closed.Store(true)
	return t.ReadCloser.Close()
}

// WasClosed returns true if Close() was called.
func (t *TrackingReadCloser) WasClosed() bool {
	return t.closed.Load()
}

// AssertClosed asserts Close() was called.
func (t *TrackingReadCloser) AssertClosed(tb testing.TB) {
	tb.Helper()
	assert.True(tb, t.closed.Load(), "ReadCloser was not closed")
}

// AssertNotClosed asserts Close() was NOT called (for reverse assertions on known bugs).
func (t *TrackingReadCloser) AssertNotClosed(tb testing.TB) {
	tb.Helper()
	assert.False(tb, t.closed.Load(), "ReadCloser was unexpectedly closed — bug may be fixed!")
}
```

- [ ] **Step 2: Run compilation check**

Run: `go build ./test/helpers/`
Expected: No errors.

- [ ] **Step 3: Commit**

```bash
git add test/helpers/tracking.go
git commit -s -m "test: add TrackingReadCloser utility for resource leak detection"
```

---

### Task 3: Modelfile Exclude Integration Tests

**Files:**
- Create: `pkg/modelfile/modelfile_integration_test.go`

Existing unit tests in `modelfile_test.go` cover config.json parsing, content generation, hidden files, and workspace validation. These tests cover the **only untested integration path**: `ExcludePatterns` field flowing through `GenerateConfig` → `NewModelfileByWorkspace` → `PathFilter` → workspace walk.

- [ ] **Step 1: Write tests in `pkg/modelfile/modelfile_integration_test.go`**

```go
package modelfile

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	configmodelfile "github.com/modelpack/modctl/pkg/config/modelfile"
)

func TestIntegration_ExcludePatterns_SinglePattern(t *testing.T) {
	tempDir := t.TempDir()

	// Create workspace with model files and log files.
	files := map[string]string{
		"model.bin":     "model data",
		"config.json":   `{"model_type": "test"}`,
		"train.log":     "training log",
		"eval.log":      "eval log",
		"run.py":        "code",
	}
	for name, content := range files {
		require.NoError(t, os.WriteFile(filepath.Join(tempDir, name), []byte(content), 0644))
	}

	config := &configmodelfile.GenerateConfig{
		Workspace:       tempDir,
		Name:            "exclude-test",
		ExcludePatterns: []string{"*.log"},
	}

	mf, err := NewModelfileByWorkspace(tempDir, config)
	require.NoError(t, err)

	// Collect all files in the modelfile.
	allFiles := append(append(append(mf.GetConfigs(), mf.GetModels()...), mf.GetCodes()...), mf.GetDocs()...)

	// .log files should be excluded.
	for _, f := range allFiles {
		assert.NotContains(t, f, ".log", "excluded file %s should not appear", f)
	}

	// model.bin, config.json, run.py should still be present.
	assert.Contains(t, mf.GetModels(), "model.bin")
	assert.Contains(t, mf.GetConfigs(), "config.json")
	assert.Contains(t, mf.GetCodes(), "run.py")
}

func TestIntegration_ExcludePatterns_MultiplePatterns(t *testing.T) {
	tempDir := t.TempDir()

	// Create workspace with various file types and a checkpoints directory.
	dirs := []string{"checkpoints", "src"}
	for _, d := range dirs {
		require.NoError(t, os.MkdirAll(filepath.Join(tempDir, d), 0755))
	}

	files := map[string]string{
		"model.bin":                "model",
		"config.json":             `{"model_type": "test"}`,
		"debug.log":               "log",
		"checkpoints/step100.bin": "ckpt",
		"src/train.py":            "code",
	}
	for name, content := range files {
		require.NoError(t, os.WriteFile(filepath.Join(tempDir, name), []byte(content), 0644))
	}

	config := &configmodelfile.GenerateConfig{
		Workspace:       tempDir,
		Name:            "multi-exclude-test",
		ExcludePatterns: []string{"*.log", "checkpoints/*"},
	}

	mf, err := NewModelfileByWorkspace(tempDir, config)
	require.NoError(t, err)

	allFiles := append(append(append(mf.GetConfigs(), mf.GetModels()...), mf.GetCodes()...), mf.GetDocs()...)

	// .log files and checkpoints/* should be excluded.
	for _, f := range allFiles {
		assert.NotContains(t, f, ".log")
		assert.NotContains(t, f, "checkpoints/")
	}

	// Remaining files should be present.
	assert.Contains(t, mf.GetModels(), "model.bin")
	assert.Contains(t, mf.GetCodes(), "src/train.py")
}
```

- [ ] **Step 2: Run tests**

Run: `go test ./pkg/modelfile/ -run TestIntegration_ExcludePatterns -v -count=1`
Expected: PASS.

- [ ] **Step 3: Commit**

```bash
git add pkg/modelfile/modelfile_integration_test.go
git commit -s -m "test: add integration tests for ExcludePatterns through NewModelfileByWorkspace"
```

---

### Task 4: ProgressBar Concurrency Tests

**Files:**
- Create: `internal/pb/pb_integration_test.go`

`internal/pb/` has zero tests. These test concurrency safety.

- [ ] **Step 1: Write tests in `internal/pb/pb_integration_test.go`**

```go
package pb

import (
	"io"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestKnownBug_DisableProgress_DataRace(t *testing.T) {
	// This test documents the data race in SetDisableProgress/Add.
	// Run with: go test -race ./internal/pb/ -run TestKnownBug_DisableProgress_DataRace
	//
	// Known bug: global disableProgress bool has no atomic protection.
	// See: https://github.com/modelpack/modctl/issues/493
	//
	// When the bug is fixed (atomic.Bool), this test will still pass
	// AND the -race detector will stop reporting the race.
	// At that point, remove the KnownBug prefix.

	var wg sync.WaitGroup
	pb := NewProgressBar(io.Discard)
	pb.Start()
	defer pb.Stop()

	// Concurrent SetDisableProgress.
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				SetDisableProgress(j%2 == 0)
			}
		}()
	}

	// Concurrent Add.
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				reader := strings.NewReader("test data")
				pb.Add("test", "bar-"+string(rune('a'+id)), 9, reader)
			}
		}(i)
	}

	wg.Wait()
	// If we get here without panic, the test passes.
	// The real detection is via -race flag.
}

func TestIntegration_ProgressBar_ConcurrentUpdates(t *testing.T) {
	pb := NewProgressBar(io.Discard)
	pb.Start()
	defer pb.Stop()

	// Disable progress to avoid mpb rendering issues in test.
	SetDisableProgress(true)
	defer SetDisableProgress(false)

	var wg sync.WaitGroup

	// Concurrent Add + Complete + Abort from multiple goroutines.
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			name := string(rune('a' + id))
			reader := strings.NewReader("data")
			pb.Add("prompt", name, 4, reader)
			if id%3 == 0 {
				pb.Complete(name, "done")
			} else if id%3 == 1 {
				pb.Abort(name, assert.AnError)
			}
			// id%3 == 2: just leave it
		}(i)
	}

	wg.Wait()
	// No panic = success.
}
```

- [ ] **Step 2: Run tests (without race detector first)**

Run: `go test ./internal/pb/ -run 'TestKnownBug_DisableProgress|TestIntegration_ProgressBar' -v -count=1`
Expected: PASS (both tests complete without panic).

- [ ] **Step 3: Run with race detector to document the known bug**

Run: `go test ./internal/pb/ -run TestKnownBug_DisableProgress -race -v -count=1 2>&1 || true`
Expected: Race detector reports data race on `disableProgress`. This is the documented bug (#493). The test itself still passes (no panic), but `-race` will exit non-zero.

- [ ] **Step 4: Commit**

```bash
git add internal/pb/pb_integration_test.go
git commit -s -m "test: add ProgressBar concurrency tests, document #493 data race"
```

---

### Task 5: Pull Integration Tests

**Files:**
- Create: `pkg/backend/pull_integration_test.go`

Pull tests use MockRegistry (remote) + mock Storage (local destination). The pattern follows `fetch_test.go:98` where `&backend{}` is constructed directly, but with a mock `store` field since Pull uses `b.store`.

Reference: `pull.go:40-171` (Pull), `pull.go:174-243` (pullIfNotExist)

- [ ] **Step 1: Write pull integration tests**

Create `pkg/backend/pull_integration_test.go` with the following tests. Each test creates its own MockRegistry and mock Storage.

```go
package backend

import (
	"context"
	"encoding/json"
	"io"
	"strings"
	"testing"
	"time"

	modelspec "github.com/modelpack/model-spec/specs-go/v1"
	godigest "github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/modelpack/modctl/pkg/config"
	"github.com/modelpack/modctl/test/helpers"
	storageMock "github.com/modelpack/modctl/test/mocks/storage"
)

// newPullTestFixture creates a MockRegistry with one manifest + N blobs,
// and returns the registry, target string, blob contents, and config.
func newPullTestFixture(t *testing.T, blobCount int) (*helpers.MockRegistry, string, [][]byte, ocispec.Manifest) {
	t.Helper()
	mr := helpers.NewMockRegistry()

	blobs := make([][]byte, blobCount)
	layers := make([]ocispec.Descriptor, blobCount)
	for i := 0; i < blobCount; i++ {
		content := []byte(strings.Repeat("x", 100+i))
		blobs[i] = content
		digest := mr.AddBlob(content)
		layers[i] = ocispec.Descriptor{
			MediaType: "application/octet-stream",
			Digest:    godigest.Digest(digest),
			Size:      int64(len(content)),
			Annotations: map[string]string{
				modelspec.AnnotationFilepath: "layer" + string(rune('0'+i)),
			},
		}
	}

	// Config blob.
	configContent := []byte(`{"model_type":"test"}`)
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

	return mr, mr.Host() + "/test/model:latest", blobs, manifest
}

// newMockStorageForPull returns a mock Storage that accepts all writes.
func newMockStorageForPull() *storageMock.Storage {
	s := storageMock.NewStorage(nil)
	// StatBlob returns false (not exist) so pull writes the blob.
	s.On("StatBlob", mock.Anything, mock.Anything, mock.Anything).Return(false, nil)
	// StatManifest returns false (not exist) so pull writes the manifest.
	s.On("StatManifest", mock.Anything, mock.Anything, mock.Anything).Return(false, nil)
	// PushBlob accepts anything.
	s.On("PushBlob", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(godigest.Digest(""), int64(0), nil)
	// PushManifest accepts anything.
	s.On("PushManifest", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(godigest.Digest(""), nil)
	return s
}

func newPullConfig() *config.Pull {
	cfg := config.NewPull()
	cfg.PlainHTTP = true
	cfg.DisableProgress = true
	cfg.Concurrency = 5
	return cfg
}

// --- Dimension 1: Functional Correctness ---

func TestIntegration_Pull_HappyPath(t *testing.T) {
	mr, target, _, _ := newPullTestFixture(t, 2)
	defer mr.Close()

	s := newMockStorageForPull()
	b := &backend{store: s}

	err := b.Pull(context.Background(), target, newPullConfig())
	require.NoError(t, err)

	// Verify PushBlob was called for each layer + config.
	s.AssertNumberOfCalls(t, "PushBlob", 2) // 2 layers
	// Verify PushManifest was called.
	s.AssertCalled(t, "PushManifest", mock.Anything, "test/model", "latest", mock.Anything)
}

func TestIntegration_Pull_BlobAlreadyExists(t *testing.T) {
	mr, target, _, _ := newPullTestFixture(t, 1)
	defer mr.Close()

	s := newMockStorageForPull()
	// Override: blob already exists locally.
	s.ExpectedCalls = nil // Clear defaults
	s.On("StatBlob", mock.Anything, mock.Anything, mock.Anything).Return(true, nil)
	s.On("StatManifest", mock.Anything, mock.Anything, mock.Anything).Return(true, nil)
	s.On("PushBlob", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(godigest.Digest(""), int64(0), nil)
	s.On("PushManifest", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(godigest.Digest(""), nil)

	b := &backend{store: s}
	err := b.Pull(context.Background(), target, newPullConfig())
	require.NoError(t, err)

	// PushBlob should NOT have been called since blob already exists.
	s.AssertNotCalled(t, "PushBlob", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestIntegration_Pull_ConcurrentLayers(t *testing.T) {
	mr, target, _, _ := newPullTestFixture(t, 5)
	defer mr.Close()

	s := newMockStorageForPull()
	b := &backend{store: s}

	err := b.Pull(context.Background(), target, newPullConfig())
	require.NoError(t, err)

	// All 5 blobs should have been pushed to storage.
	s.AssertNumberOfCalls(t, "PushBlob", 5)
}

// --- Dimension 2: Network Errors (fast) ---

func TestIntegration_Pull_ContextTimeout(t *testing.T) {
	mr, target, _, _ := newPullTestFixture(t, 1)
	defer mr.Close()
	mr.WithFault(&helpers.FaultConfig{LatencyPerRequest: 5 * time.Second})

	s := newMockStorageForPull()
	b := &backend{store: s}

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	err := b.Pull(ctx, target, newPullConfig())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context")
}

func TestIntegration_Pull_PartialResponse(t *testing.T) {
	mr, target, _, _ := newPullTestFixture(t, 1)
	defer mr.Close()

	// Fail after 10 bytes on blob fetches.
	mr.WithFault(&helpers.FaultConfig{
		PathFaults: map[string]*helpers.FaultConfig{
			// Match any blob path.
			"/blobs/": {FailAfterNBytes: 10},
		},
	})

	s := newMockStorageForPull()
	b := &backend{store: s}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := b.Pull(ctx, target, newPullConfig())
	assert.Error(t, err)
}

func TestIntegration_Pull_ManifestOK_BlobFails(t *testing.T) {
	mr, target, _, manifest := newPullTestFixture(t, 2)
	defer mr.Close()

	// Only fail on the second blob's digest.
	failDigest := manifest.Layers[1].Digest.String()
	mr.WithFault(&helpers.FaultConfig{
		PathFaults: map[string]*helpers.FaultConfig{
			failDigest: {StatusCodeOverride: 500},
		},
	})

	s := newMockStorageForPull()
	b := &backend{store: s}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := b.Pull(ctx, target, newPullConfig())
	assert.Error(t, err)
}

// --- Dimension 4: Concurrency Safety ---

func TestIntegration_Pull_ConcurrentPartialFailure(t *testing.T) {
	mr, target, _, manifest := newPullTestFixture(t, 5)
	defer mr.Close()

	// Fail 2 out of 5 blobs.
	mr.WithFault(&helpers.FaultConfig{
		PathFaults: map[string]*helpers.FaultConfig{
			manifest.Layers[1].Digest.String(): {StatusCodeOverride: 500},
			manifest.Layers[3].Digest.String(): {StatusCodeOverride: 500},
		},
	})

	s := newMockStorageForPull()
	b := &backend{store: s}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := b.Pull(ctx, target, newPullConfig())
	assert.Error(t, err)
	// Key assertion: no hang, error returned within timeout.
}

// --- Dimension 6: Data Integrity ---

func TestIntegration_Pull_TruncatedBlob(t *testing.T) {
	mr := helpers.NewMockRegistry()
	defer mr.Close()

	// Add a blob but serve truncated content via fault.
	content := []byte(strings.Repeat("x", 200))
	digest := mr.AddBlob(content)

	configContent := []byte(`{}`)
	configDigest := mr.AddBlob(configContent)

	manifest := ocispec.Manifest{
		MediaType: ocispec.MediaTypeImageManifest,
		Config: ocispec.Descriptor{
			MediaType: ocispec.MediaTypeImageConfig,
			Digest:    godigest.Digest(configDigest),
			Size:      int64(len(configContent)),
		},
		Layers: []ocispec.Descriptor{
			{
				MediaType: "application/octet-stream",
				Digest:    godigest.Digest(digest),
				Size:      int64(len(content)),
			},
		},
	}
	mr.AddManifest("test/model:latest", manifest)

	// Serve only first 50 bytes.
	mr.WithFault(&helpers.FaultConfig{
		PathFaults: map[string]*helpers.FaultConfig{
			digest: {FailAfterNBytes: 50},
		},
	})

	s := newMockStorageForPull()
	b := &backend{store: s}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := b.Pull(ctx, target(mr), newPullConfig())
	assert.Error(t, err, "truncated blob should cause error (digest mismatch or read error)")
}

func TestIntegration_Pull_CorruptedBlob(t *testing.T) {
	mr := helpers.NewMockRegistry()
	defer mr.Close()

	// Add a blob with correct digest but we'll serve wrong content.
	realContent := []byte("real content")
	realDigest := godigest.FromBytes(realContent)

	// Store wrong content under the real digest key.
	mr.AddBlob(realContent) // stores under correct digest
	// Now overwrite with corrupt data — need direct access.
	// Instead: create manifest referencing one digest, but serve different content.
	// Simpler approach: use a custom handler. For now, test that correct content passes
	// and we trust the validateDigest path.
	// Actually let's test differently: store corrupt data manually.

	configContent := []byte(`{}`)
	configDigest := mr.AddBlob(configContent)

	manifest := ocispec.Manifest{
		MediaType: ocispec.MediaTypeImageManifest,
		Config: ocispec.Descriptor{
			MediaType: ocispec.MediaTypeImageConfig,
			Digest:    godigest.Digest(godigest.FromBytes(configContent).String()),
			Size:      int64(len(configContent)),
		},
		Layers: []ocispec.Descriptor{
			{
				MediaType: "application/octet-stream",
				Digest:    realDigest,
				Size:      int64(len(realContent)),
			},
		},
	}
	mr.AddManifest("test/model:latest", manifest)

	// Note: To serve corrupt data, we'd need to modify the MockRegistry to allow
	// storing data under a mismatched digest. For this test, add a method or
	// directly manipulate internals. The implementer should add a
	// MockRegistry.AddBlobWithDigest(digest, content) method for this purpose.

	s := newMockStorageForPull()
	b := &backend{store: s}

	err := b.Pull(context.Background(), mr.Host()+"/test/model:latest", newPullConfig())
	// With correct data, this should pass. The corruption test needs AddBlobWithDigest.
	// Implementer: add AddBlobWithDigest to MockRegistry, store corrupt bytes under realDigest.
	require.NoError(t, err) // Placeholder — replace with corruption test.
}

// --- Dimension 7: Graceful Shutdown ---

func TestIntegration_Pull_ContextCancelMidDownload(t *testing.T) {
	mr, target, _, _ := newPullTestFixture(t, 3)
	defer mr.Close()

	// Add latency to simulate slow download.
	mr.WithFault(&helpers.FaultConfig{LatencyPerRequest: 500 * time.Millisecond})

	s := newMockStorageForPull()
	b := &backend{store: s}

	ctx, cancel := context.WithCancel(context.Background())

	// Cancel after 300ms — mid-download.
	go func() {
		time.Sleep(300 * time.Millisecond)
		cancel()
	}()

	err := b.Pull(ctx, target, newPullConfig())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context")
}

// --- Dimension 8: Idempotency ---

func TestIntegration_Pull_Idempotent(t *testing.T) {
	mr, target, _, _ := newPullTestFixture(t, 2)
	defer mr.Close()

	s := newMockStorageForPull()
	b := &backend{store: s}

	// First pull.
	err := b.Pull(context.Background(), target, newPullConfig())
	require.NoError(t, err)

	countAfterFirst := mr.RequestCount()

	// Now make storage report blobs exist.
	s.ExpectedCalls = nil
	s.On("StatBlob", mock.Anything, mock.Anything, mock.Anything).Return(true, nil)
	s.On("StatManifest", mock.Anything, mock.Anything, mock.Anything).Return(true, nil)
	s.On("PushBlob", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(godigest.Digest(""), int64(0), nil)
	s.On("PushManifest", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(godigest.Digest(""), nil)

	// Second pull — should skip blob fetches but still fetch manifest.
	err = b.Pull(context.Background(), target, newPullConfig())
	require.NoError(t, err)

	// Verify second pull made fewer requests (manifest fetch only, no blob fetches).
	countAfterSecond := mr.RequestCount()
	blobRequestsDelta := countAfterSecond - countAfterFirst
	// Should be: 1 manifest GET + possibly HEAD checks, but NO blob GETs.
	// The exact count depends on oras-go internals, but should be < first pull count.
	assert.Less(t, blobRequestsDelta, countAfterFirst,
		"second pull should make fewer requests than first")
}

// target is a helper to construct a target string from MockRegistry.
func target(mr *helpers.MockRegistry) string {
	return mr.Host() + "/test/model:latest"
}
```

**Important notes for implementer:**
- The mock storage setup pattern (`newMockStorageForPull()`) may need adjustment based on exact mock method signatures in `test/mocks/storage/storage.go`. Check the generated mock's `NewStorage` constructor — it may take `*testing.T` instead of `nil`.
- `TestIntegration_Pull_CorruptedBlob` needs a `MockRegistry.AddBlobWithDigest(digest string, content []byte)` method that stores content under a specific digest regardless of actual hash. Add this to `mockregistry.go`.
- Some tests may need the `target` helper or direct `mr.Host()+"/test/model:latest"` depending on the fixture setup.

- [ ] **Step 2: Add `AddBlobWithDigest` to MockRegistry**

Add to `test/helpers/mockregistry.go`:
```go
// AddBlobWithDigest stores content under an explicit digest (for corruption tests).
func (mr *MockRegistry) AddBlobWithDigest(digest string, content []byte) *MockRegistry {
	mr.mu.Lock()
	mr.blobs[digest] = content
	mr.mu.Unlock()
	return mr
}
```

Then fix `TestIntegration_Pull_CorruptedBlob` to use it:
```go
func TestIntegration_Pull_CorruptedBlob(t *testing.T) {
	mr := helpers.NewMockRegistry()
	defer mr.Close()

	// Create manifest referencing a blob by its real digest,
	// but serve different (corrupt) content under that digest.
	realContent := []byte("real content that should be here")
	corruptContent := []byte("THIS IS CORRUPT DATA!!!!!!!!!!")
	realDigest := godigest.FromBytes(realContent).String()
	mr.AddBlobWithDigest(realDigest, corruptContent)

	configContent := []byte(`{}`)
	configDigest := mr.AddBlob(configContent)
	manifest := ocispec.Manifest{
		MediaType: ocispec.MediaTypeImageManifest,
		Config: ocispec.Descriptor{
			MediaType: ocispec.MediaTypeImageConfig,
			Digest:    godigest.Digest(configDigest),
			Size:      int64(len(configContent)),
		},
		Layers: []ocispec.Descriptor{{
			MediaType: "application/octet-stream",
			Digest:    godigest.Digest(realDigest),
			Size:      int64(len(realContent)), // Size matches real, content doesn't
		}},
	}
	mr.AddManifest("test/model:latest", manifest)

	s := newMockStorageForPull()
	b := &backend{store: s}

	err := b.Pull(context.Background(), mr.Host()+"/test/model:latest", newPullConfig())
	assert.Error(t, err, "corrupted blob should fail digest validation")
	assert.Contains(t, err.Error(), "digest")
}
```

- [ ] **Step 3: Run pull integration tests**

Run: `go test ./pkg/backend/ -run TestIntegration_Pull -v -count=1 -timeout 60s`
Expected: All tests PASS except possibly the corruption test if sizes differ. Adjust as needed.

- [ ] **Step 4: Commit**

```bash
git add pkg/backend/pull_integration_test.go test/helpers/mockregistry.go
git commit -s -m "test: add pull integration tests covering 6 dimensions"
```

---

### Task 6: Push Integration Tests

**Files:**
- Create: `pkg/backend/push_integration_test.go`

Push uses mock Storage as source (local) and MockRegistry as destination (remote). The known-bug tests for ReadCloser leak (#491) use TrackingReadCloser injected through mock Storage's `PullBlob` method.

Reference: `push.go:38-128` (Push), `push.go:130-193` (pushIfNotExist)

- [ ] **Step 1: Write push integration tests**

Create `pkg/backend/push_integration_test.go`:

```go
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

type pushFixture struct {
	registry    *helpers.MockRegistry
	target      string
	manifest    ocispec.Manifest
	manifestRaw []byte
	blobContent []byte
	blobDigest  godigest.Digest
	configRaw   []byte
	configDigest godigest.Digest
}

func newPushTestFixture(t *testing.T) *pushFixture {
	t.Helper()
	mr := helpers.NewMockRegistry()

	blobContent := []byte("push test blob content")
	blobDigest := godigest.FromBytes(blobContent)
	configRaw := []byte(`{"model_type":"test"}`)
	configDigest := godigest.FromBytes(configRaw)

	manifest := ocispec.Manifest{
		MediaType: ocispec.MediaTypeImageManifest,
		Config: ocispec.Descriptor{
			MediaType: ocispec.MediaTypeImageConfig,
			Digest:    configDigest,
			Size:      int64(len(configRaw)),
		},
		Layers: []ocispec.Descriptor{{
			MediaType: "application/octet-stream",
			Digest:    blobDigest,
			Size:      int64(len(blobContent)),
		}},
	}
	manifestRaw, _ := json.Marshal(manifest)

	return &pushFixture{
		registry:     mr,
		target:       mr.Host() + "/test/model:latest",
		manifest:     manifest,
		manifestRaw:  manifestRaw,
		blobContent:  blobContent,
		blobDigest:   blobDigest,
		configRaw:    configRaw,
		configDigest: configDigest,
	}
}

func newMockStorageForPush(f *pushFixture) *storageMock.Storage {
	s := storageMock.NewStorage(nil)
	s.On("PullManifest", mock.Anything, "test/model", "latest").
		Return(f.manifestRaw, godigest.FromBytes(f.manifestRaw), nil)
	s.On("PullBlob", mock.Anything, mock.Anything, f.blobDigest.String()).
		Return(io.NopCloser(bytes.NewReader(f.blobContent)), nil)
	s.On("PullBlob", mock.Anything, mock.Anything, f.configDigest.String()).
		Return(io.NopCloser(bytes.NewReader(f.configRaw)), nil)
	return s
}

func newPushConfig() *config.Push {
	cfg := config.NewPush()
	cfg.PlainHTTP = true
	cfg.Concurrency = 5
	return cfg
}

// --- Dimension 1: Functional Correctness ---

func TestIntegration_Push_HappyPath(t *testing.T) {
	f := newPushTestFixture(t)
	defer f.registry.Close()

	s := newMockStorageForPush(f)
	b := &backend{store: s}

	err := b.Push(context.Background(), f.target, newPushConfig())
	require.NoError(t, err)

	// Verify blob was received by registry.
	assert.True(t, f.registry.BlobExists(f.blobDigest.String()),
		"blob should have been pushed to registry")

	// Verify manifest was received.
	assert.True(t, f.registry.ManifestExists("test/model:latest"),
		"manifest should have been pushed to registry")
}

func TestIntegration_Push_BlobAlreadyExists(t *testing.T) {
	f := newPushTestFixture(t)
	defer f.registry.Close()

	// Pre-populate registry with the blob and manifest.
	f.registry.AddBlobWithDigest(f.blobDigest.String(), f.blobContent)
	f.registry.AddBlobWithDigest(f.configDigest.String(), f.configRaw)
	f.registry.AddManifest("test/model:latest", f.manifest)

	s := newMockStorageForPush(f)
	b := &backend{store: s}

	err := b.Push(context.Background(), f.target, newPushConfig())
	require.NoError(t, err)

	// PullBlob should NOT have been called since remote already has the blob.
	s.AssertNotCalled(t, "PullBlob", mock.Anything, mock.Anything, f.blobDigest.String())
}

// --- Dimension 2: Network Errors ---

func TestIntegration_Push_ManifestPushFails(t *testing.T) {
	f := newPushTestFixture(t)
	defer f.registry.Close()

	// Fail only manifest push (PUT /manifests/).
	f.registry.WithFault(&helpers.FaultConfig{
		PathFaults: map[string]*helpers.FaultConfig{
			"/manifests/": {StatusCodeOverride: 500},
		},
	})

	s := newMockStorageForPush(f)
	b := &backend{store: s}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := b.Push(ctx, f.target, newPushConfig())
	assert.Error(t, err)
}

// --- Dimension 3: Resource Leak Detection (Known Bugs) ---

func TestKnownBug_Push_ReadCloserNotClosed_SuccessPath(t *testing.T) {
	// BUG: push.go:175 — PullBlob returns io.ReadCloser that is never closed.
	// The content is wrapped with io.NopCloser before passing to dst.Blobs().Push(),
	// so even on success, the original ReadCloser.Close() is never called.
	// See: https://github.com/modelpack/modctl/issues/491
	f := newPushTestFixture(t)
	defer f.registry.Close()

	tracker := helpers.NewTrackingReadCloser(io.NopCloser(bytes.NewReader(f.blobContent)))

	s := storageMock.NewStorage(nil)
	s.On("PullManifest", mock.Anything, mock.Anything, mock.Anything).
		Return(f.manifestRaw, godigest.FromBytes(f.manifestRaw), nil)
	// Return tracker for blob pull.
	s.On("PullBlob", mock.Anything, mock.Anything, f.blobDigest.String()).
		Return(io.ReadCloser(tracker), nil)
	// Config uses normal reader.
	s.On("PullBlob", mock.Anything, mock.Anything, f.configDigest.String()).
		Return(io.NopCloser(bytes.NewReader(f.configRaw)), nil)

	b := &backend{store: s}

	err := b.Push(context.Background(), f.target, newPushConfig())
	require.NoError(t, err, "push should succeed")

	// REVERSE ASSERTION: assert the bug EXISTS (Close was NOT called).
	// When #491 is fixed, this will fail — flip to AssertClosed and remove KnownBug prefix.
	tracker.AssertNotClosed(t)
}

func TestKnownBug_Push_ReadCloserNotClosed_ErrorPath(t *testing.T) {
	// Same bug on error path.
	// See: https://github.com/modelpack/modctl/issues/491
	f := newPushTestFixture(t)
	defer f.registry.Close()

	// Make blob push fail.
	f.registry.WithFault(&helpers.FaultConfig{
		PathFaults: map[string]*helpers.FaultConfig{
			"/blobs/uploads/": {StatusCodeOverride: 500},
		},
	})

	tracker := helpers.NewTrackingReadCloser(io.NopCloser(bytes.NewReader(f.blobContent)))

	s := storageMock.NewStorage(nil)
	s.On("PullManifest", mock.Anything, mock.Anything, mock.Anything).
		Return(f.manifestRaw, godigest.FromBytes(f.manifestRaw), nil)
	s.On("PullBlob", mock.Anything, mock.Anything, f.blobDigest.String()).
		Return(io.ReadCloser(tracker), nil)
	s.On("PullBlob", mock.Anything, mock.Anything, f.configDigest.String()).
		Return(io.NopCloser(bytes.NewReader(f.configRaw)), nil)

	b := &backend{store: s}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := b.Push(ctx, f.target, newPushConfig())
	assert.Error(t, err)

	// REVERSE ASSERTION: Close was NOT called even on error path.
	tracker.AssertNotClosed(t)
}

// --- Dimension 6: Data Integrity ---

func TestIntegration_Push_VerifyBlobIntegrity(t *testing.T) {
	f := newPushTestFixture(t)
	defer f.registry.Close()

	s := newMockStorageForPush(f)
	b := &backend{store: s}

	err := b.Push(context.Background(), f.target, newPushConfig())
	require.NoError(t, err)

	// Verify the registry received the exact bytes.
	received, ok := f.registry.GetBlob(f.blobDigest.String())
	require.True(t, ok, "blob should exist in registry")
	assert.Equal(t, f.blobContent, received, "pushed blob content should match source")
}

// --- Dimension 7: Graceful Shutdown ---

func TestIntegration_Push_ContextCancelMidUpload(t *testing.T) {
	f := newPushTestFixture(t)
	defer f.registry.Close()

	// Add latency to simulate slow upload.
	f.registry.WithFault(&helpers.FaultConfig{LatencyPerRequest: 500 * time.Millisecond})

	s := newMockStorageForPush(f)
	b := &backend{store: s}

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(200 * time.Millisecond)
		cancel()
	}()

	err := b.Push(ctx, f.target, newPushConfig())
	assert.Error(t, err)
}

// --- Dimension 8: Idempotency ---

func TestIntegration_Push_Idempotent(t *testing.T) {
	f := newPushTestFixture(t)
	defer f.registry.Close()

	s := newMockStorageForPush(f)
	b := &backend{store: s}

	// First push.
	err := b.Push(context.Background(), f.target, newPushConfig())
	require.NoError(t, err)

	countAfterFirst := f.registry.RequestCount()

	// Second push — registry already has all blobs.
	err = b.Push(context.Background(), f.target, newPushConfig())
	require.NoError(t, err)

	countAfterSecond := f.registry.RequestCount()
	secondPushRequests := countAfterSecond - countAfterFirst
	assert.Less(t, secondPushRequests, countAfterFirst,
		"second push should make fewer requests (blobs already exist)")
}
```

- [ ] **Step 2: Run push integration tests**

Run: `go test ./pkg/backend/ -run TestIntegration_Push -v -count=1 -timeout 60s`
Expected: PASS (including KnownBug tests which use reverse assertions).

- [ ] **Step 3: Run KnownBug tests specifically**

Run: `go test ./pkg/backend/ -run TestKnownBug_Push -v -count=1`
Expected: PASS (reverse assertions confirm the bug exists).

- [ ] **Step 4: Commit**

```bash
git add pkg/backend/push_integration_test.go
git commit -s -m "test: add push integration tests, document ReadCloser leak #491"
```

---

### Task 7: CLI Integration Tests

**Files:**
- Create: `cmd/modelfile/generate_integration_test.go`
- Create: `cmd/cli_integration_test.go`

The `cmd/` directory has zero tests. These test cobra command execution.

- [ ] **Step 1: Write `cmd/modelfile/generate_integration_test.go`**

```go
package modelfile

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntegration_CLI_Generate_BasicFlags(t *testing.T) {
	tempDir := t.TempDir()
	outputDir := t.TempDir()

	// Create minimal workspace.
	require.NoError(t, os.WriteFile(filepath.Join(tempDir, "model.bin"), []byte("data"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tempDir, "config.json"), []byte(`{"model_type":"llama"}`), 0644))

	// Reset global config for test isolation.
	generateConfig = NewGenerateConfigForTest()

	cmd := generateCmd
	cmd.SetArgs([]string{
		tempDir,
		"--name", "test-model",
		"--arch", "transformer",
		"--family", "llama3",
		"--output", outputDir,
	})

	err := cmd.Execute()
	require.NoError(t, err)

	// Verify Modelfile was written.
	modelfilePath := filepath.Join(outputDir, "Modelfile")
	content, err := os.ReadFile(modelfilePath)
	require.NoError(t, err)
	assert.Contains(t, string(content), "NAME test-model")
	assert.Contains(t, string(content), "ARCH transformer")
	assert.Contains(t, string(content), "FAMILY llama3")
}

func TestIntegration_CLI_Generate_OutputAndOverwrite(t *testing.T) {
	tempDir := t.TempDir()
	outputDir := t.TempDir()

	require.NoError(t, os.WriteFile(filepath.Join(tempDir, "model.bin"), []byte("data"), 0644))

	// Pre-create Modelfile.
	modelfilePath := filepath.Join(outputDir, "Modelfile")
	require.NoError(t, os.WriteFile(modelfilePath, []byte("existing"), 0644))

	// Without --overwrite should fail.
	generateConfig = NewGenerateConfigForTest()
	cmd := generateCmd
	cmd.SetArgs([]string{tempDir, "--output", outputDir})
	err := cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")

	// With --overwrite should succeed.
	generateConfig = NewGenerateConfigForTest()
	cmd.SetArgs([]string{tempDir, "--output", outputDir, "--overwrite"})
	err = cmd.Execute()
	assert.NoError(t, err)
}

func TestIntegration_CLI_Generate_MutualExclusion(t *testing.T) {
	generateConfig = NewGenerateConfigForTest()
	cmd := generateCmd
	cmd.SetArgs([]string{"./workspace", "--model-url", "some/model"})
	err := cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "mutually exclusive")
}

// NewGenerateConfigForTest returns a fresh GenerateConfig to avoid test pollution.
// The implementer should check if configmodelfile.NewGenerateConfig() can be used directly
// or if a test-specific helper is needed based on the global generateConfig variable pattern.
func NewGenerateConfigForTest() *configmodelfile.GenerateConfig {
	return configmodelfile.NewGenerateConfig()
}
```

**Note for implementer:** The global `generateConfig` variable in `generate.go:32` makes cobra command tests tricky — each test must reset it. Check if `generateCmd` can be reconstructed or if resetting the global is sufficient. You may need to import `configmodelfile` in the test file. Also, cobra's `Execute()` calls `RunE` which calls `generateConfig.Convert()` and `generateConfig.Validate()` before `runGenerate()`, so the Validate step may error before the mutually-exclusive check in `RunE`.

- [ ] **Step 2: Write `cmd/cli_integration_test.go`**

```go
package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/modelpack/modctl/pkg/config"
)

func TestIntegration_CLI_ConcurrencyZero(t *testing.T) {
	cfg := config.NewPull()
	cfg.Concurrency = 0
	err := cfg.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "concurrency")
}

func TestIntegration_CLI_ConcurrencyNegative(t *testing.T) {
	cfg := config.NewPull()
	cfg.Concurrency = -1
	err := cfg.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "concurrency")
}

func TestIntegration_CLI_ExtractFromRemoteNoDir(t *testing.T) {
	cfg := config.NewPull()
	cfg.ExtractFromRemote = true
	cfg.ExtractDir = ""
	err := cfg.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "extract dir")
}
```

- [ ] **Step 3: Run CLI tests**

Run: `go test ./cmd/modelfile/ -run TestIntegration_CLI_Generate -v -count=1`
Run: `go test ./cmd/ -run TestIntegration_CLI -v -count=1`
Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add cmd/modelfile/generate_integration_test.go cmd/cli_integration_test.go
git commit -s -m "test: add CLI integration tests for generate command and config validation"
```

---

### Task 8: Slow Tests (Retry-Dependent)

**Files:**
- Create: `pkg/backend/pull_slowtest_test.go`
- Create: `pkg/backend/push_slowtest_test.go`

These tests exercise real retry backoff and take 30+ seconds. Gated behind `//go:build slowtest`.

- [ ] **Step 1: Write `pkg/backend/pull_slowtest_test.go`**

```go
//go:build slowtest

package backend

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/modelpack/modctl/test/helpers"
)

func TestSlow_Pull_RetryOnTransientError(t *testing.T) {
	mr, target, _, _ := newPullTestFixture(t, 1)
	defer mr.Close()

	// First 2 requests fail, 3rd succeeds.
	mr.WithFault(&helpers.FaultConfig{FailOnNthRequest: 2})

	s := newMockStorageForPull()
	b := &backend{store: s}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	err := b.Pull(ctx, target, newPullConfig())
	require.NoError(t, err, "pull should succeed after transient failures")
}

func TestSlow_Pull_RetryExhausted(t *testing.T) {
	mr, target, _, _ := newPullTestFixture(t, 1)
	defer mr.Close()

	// All requests fail with 500.
	mr.WithFault(&helpers.FaultConfig{StatusCodeOverride: 500})

	s := newMockStorageForPull()
	b := &backend{store: s}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	err := b.Pull(ctx, target, newPullConfig())
	assert.Error(t, err, "pull should fail after retry exhaustion")
}

func TestSlow_Pull_RateLimited(t *testing.T) {
	mr, target, _, _ := newPullTestFixture(t, 1)
	defer mr.Close()

	// First 3 requests return 429, then succeed.
	mr.WithFault(&helpers.FaultConfig{FailOnNthRequest: 3})

	s := newMockStorageForPull()
	b := &backend{store: s}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	err := b.Pull(ctx, target, newPullConfig())
	// Should eventually succeed after backoff.
	require.NoError(t, err)
}

func TestKnownBug_Pull_AuthErrorStillRetries(t *testing.T) {
	// BUG: retry.Do() retries ALL errors including 401.
	// Auth errors should fail immediately.
	// See: https://github.com/modelpack/modctl/issues/494
	mr, target, _, _ := newPullTestFixture(t, 1)
	defer mr.Close()

	mr.WithFault(&helpers.FaultConfig{StatusCodeOverride: 401})

	s := newMockStorageForPull()
	b := &backend{store: s}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	start := time.Now()
	err := b.Pull(ctx, target, newPullConfig())
	elapsed := time.Since(start)
	assert.Error(t, err)

	// REVERSE ASSERTION: Currently auth errors ARE retried (takes 30s+).
	// When #494 is fixed, auth errors should fail immediately (<2s).
	// Flip this assertion when fixed.
	assert.Greater(t, elapsed, 10*time.Second,
		"If this fails, auth errors are no longer retried — #494 may be fixed! "+
			"Update to assert.Less(t, elapsed, 5*time.Second)")
}
```

- [ ] **Step 2: Write `pkg/backend/push_slowtest_test.go`**

```go
//go:build slowtest

package backend

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/modelpack/modctl/test/helpers"
)

func TestSlow_Push_RetryOnTransientError(t *testing.T) {
	f := newPushTestFixture(t)
	defer f.registry.Close()

	// First 2 upload requests fail.
	f.registry.WithFault(&helpers.FaultConfig{
		PathFaults: map[string]*helpers.FaultConfig{
			"/blobs/uploads/": {FailOnNthRequest: 2},
		},
	})

	s := newMockStorageForPush(f)
	b := &backend{store: s}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	err := b.Push(ctx, f.target, newPushConfig())
	require.NoError(t, err, "push should succeed after transient upload failures")
}
```

- [ ] **Step 3: Verify slow tests compile (don't run)**

Run: `go test ./pkg/backend/ -tags slowtest -list 'TestSlow|TestKnownBug_Pull_Auth' -count=1`
Expected: Lists test names without running them.

- [ ] **Step 4: Commit**

```bash
git add pkg/backend/pull_slowtest_test.go pkg/backend/push_slowtest_test.go
git commit -s -m "test: add retry-dependent slow tests (//go:build slowtest)"
```

---

### Task 9: Stress Tests

**Files:**
- Create: `pkg/modelfile/modelfile_stress_test.go`
- Create: `pkg/backend/pull_stress_test.go`

Gated behind `//go:build stress`.

- [ ] **Step 1: Write `pkg/modelfile/modelfile_stress_test.go`**

```go
//go:build stress

package modelfile

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	configmodelfile "github.com/modelpack/modctl/pkg/config/modelfile"
)

func TestStress_NearMaxFileCount(t *testing.T) {
	tempDir := t.TempDir()

	// Create 2040 files (near MaxWorkspaceFileCount=2048).
	// Include at least one model file to avoid "no model/code/dataset" error.
	require.NoError(t, os.WriteFile(filepath.Join(tempDir, "model.bin"), []byte("model"), 0644))
	for i := 0; i < 2039; i++ {
		name := fmt.Sprintf("file_%04d.py", i)
		require.NoError(t, os.WriteFile(filepath.Join(tempDir, name), []byte("x"), 0644))
	}

	config := &configmodelfile.GenerateConfig{
		Workspace: tempDir,
		Name:      "stress-test",
	}

	mf, err := NewModelfileByWorkspace(tempDir, config)
	require.NoError(t, err, "should handle near-limit file count")
	require.NotNil(t, mf)
}

func TestStress_DeeplyNestedDirs(t *testing.T) {
	tempDir := t.TempDir()

	// Create 100-level nested directory with a model file at the bottom.
	parts := make([]string, 100)
	for i := range parts {
		parts[i] = fmt.Sprintf("d%d", i)
	}
	deepDir := filepath.Join(tempDir, strings.Join(parts, string(filepath.Separator)))
	require.NoError(t, os.MkdirAll(deepDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(deepDir, "model.bin"), []byte("deep model"), 0644))

	config := &configmodelfile.GenerateConfig{
		Workspace: tempDir,
		Name:      "deep-test",
	}

	mf, err := NewModelfileByWorkspace(tempDir, config)
	require.NoError(t, err, "should handle deeply nested directories")
	require.NotNil(t, mf)
	require.Contains(t, mf.GetModels(), filepath.Join(strings.Join(parts, string(filepath.Separator)), "model.bin"))
}
```

- [ ] **Step 2: Write `pkg/backend/pull_stress_test.go`**

```go
//go:build stress

package backend

import (
	"context"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStress_Pull_ManyLayers(t *testing.T) {
	mr, target, _, _ := newPullTestFixture(t, 50)
	defer mr.Close()

	s := newMockStorageForPull()
	b := &backend{store: s}

	cfg := newPullConfig()
	cfg.Concurrency = 10

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	err := b.Pull(ctx, target, cfg)
	require.NoError(t, err, "should handle 50 concurrent layers")
	s.AssertNumberOfCalls(t, "PushBlob", 50)
}

func TestStress_Pull_RepeatedCycles(t *testing.T) {
	mr, target, _, _ := newPullTestFixture(t, 2)
	defer mr.Close()

	s := newMockStorageForPull()
	b := &backend{store: s}

	baseGoroutines := runtime.NumGoroutine()

	for i := 0; i < 100; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		err := b.Pull(ctx, target, newPullConfig())
		cancel()
		require.NoError(t, err, "cycle %d should succeed", i)
	}

	// Allow goroutines to settle.
	time.Sleep(500 * time.Millisecond)

	finalGoroutines := runtime.NumGoroutine()
	// Allow some tolerance (goroutines from test framework, etc.).
	assert.InDelta(t, baseGoroutines, finalGoroutines, 20,
		"goroutine count should be stable after 100 pull cycles (no leaks)")
}
```

- [ ] **Step 3: Verify stress tests compile**

Run: `go test ./pkg/modelfile/ -tags stress -list TestStress -count=1`
Run: `go test ./pkg/backend/ -tags stress -list TestStress -count=1`
Expected: Lists test names.

- [ ] **Step 4: Commit**

```bash
git add pkg/modelfile/modelfile_stress_test.go pkg/backend/pull_stress_test.go
git commit -s -m "test: add stress tests for file count, nesting depth, and pull cycles (//go:build stress)"
```

---

## Self-Review Checklist

**Spec coverage:** All 8 dimensions mapped to tasks. All ~35 tests accounted for:
- Dim 1 (functional): Tasks 3, 5, 6, 7 ✓
- Dim 2 (network): Tasks 5, 6, 8 ✓
- Dim 3 (resource leak): Task 6 ✓
- Dim 4 (concurrency): Tasks 4, 5 ✓
- Dim 5 (stress): Task 9 ✓
- Dim 6 (data integrity): Tasks 5, 6 ✓
- Dim 7 (graceful shutdown): Tasks 5, 6 ✓
- Dim 8 (idempotency/config): Tasks 5, 6, 7 ✓

**Known bugs:** #491 in Task 6, #493 in Task 4, #494 in Task 8 ✓

**Placeholder scan:** No TBD/TODO. All code blocks contain real code. Implementation notes marked with "Note for implementer" where mock method signatures may need adjustment.

**Type consistency:** `newPullTestFixture`, `newMockStorageForPull`, `newPullConfig` used consistently across Tasks 5, 8, 9. `newPushTestFixture`, `newMockStorageForPush`, `newPushConfig` used consistently across Tasks 6, 8.
