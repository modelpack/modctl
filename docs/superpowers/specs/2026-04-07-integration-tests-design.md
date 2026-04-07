# Integration Tests Design Spec

## Goal

Add comprehensive integration tests across modctl to cover functional correctness, stress scenarios, and error handling. Tests only — no functional code changes.

## Scope

8 test dimensions, ~41 test scenarios, covering `pkg/modelfile`, `pkg/backend`, `internal/pb`, and `cmd/` layers.

**Out of scope:** Security tests (path traversal, TLS, credential handling) — deferred to a future effort.

## Test Infrastructure

### Mock OCI Registry (`test/helpers/mockregistry.go`)

A reusable `httptest`-based mock OCI registry server with fault injection.

```go
type MockRegistry struct {
    server    *httptest.Server
    manifests map[string][]byte   // ref -> manifest bytes
    blobs     map[string][]byte   // digest -> blob bytes
    faults    *FaultConfig
}

type FaultConfig struct {
    LatencyPerRequest  time.Duration   // per-request delay
    FailAfterNBytes    int64           // disconnect after N bytes (partial response)
    DropConnectionRate float64         // random connection drop probability [0,1)
    StatusCodeOverride int             // force HTTP status (500, 401, 429)
    FailOnNthRequest   int             // fail on Nth request (test retry)
    PathFaults         map[string]*FaultConfig // per-path overrides
}
```

**Implemented OCI endpoints (minimal subset used by oras-go):**
- `GET /v2/` — ping
- `HEAD|GET /v2/<name>/manifests/<ref>` — manifest operations
- `PUT /v2/<name>/manifests/<ref>` — push manifest
- `HEAD|GET /v2/<name>/blobs/<digest>` — blob operations
- `POST /v2/<name>/blobs/uploads/` — start upload
- `PUT /v2/<name>/blobs/uploads/<id>` — complete upload

**API:**
```go
func NewMockRegistry() *MockRegistry
func (r *MockRegistry) WithFault(f *FaultConfig) *MockRegistry
func (r *MockRegistry) AddManifest(ref string, manifest []byte) *MockRegistry
func (r *MockRegistry) AddBlob(digest string, content []byte) *MockRegistry
func (r *MockRegistry) Host() string
func (r *MockRegistry) Close()
func (r *MockRegistry) RequestCount() int
func (r *MockRegistry) RequestCountByPath(path string) int
```

Self-tested in `test/helpers/mockregistry_test.go`.

### Resource Tracking Utilities (`test/helpers/tracking.go`)

```go
type TrackingReadCloser struct {
    io.ReadCloser
    closed atomic.Bool
}

func NewTrackingReadCloser(rc io.ReadCloser) *TrackingReadCloser
func (t *TrackingReadCloser) Close() error
func (t *TrackingReadCloser) WasClosed() bool
```

Used to assert resources are properly closed on all code paths.

## Test Dimensions & Scenarios

### Dimension 1: Functional Correctness (no build tag)

**File: `pkg/modelfile/modelfile_integration_test.go`**

| Test | Description |
|------|-------------|
| `TestIntegration_ExcludePatterns_SinglePattern` | `--exclude *.log` removes .log files from generated Modelfile |
| `TestIntegration_ExcludePatterns_MultiplePatterns` | `--exclude *.log --exclude checkpoints/*` removes both |
| `TestIntegration_ExcludePatterns_NestedDirExclude` | `--exclude data/` skips entire directory tree |
| `TestIntegration_ContentRoundTrip` | workspace -> NewModelfileByWorkspace -> Content() -> NewModelfile parse back, verify equivalence |
| `TestIntegration_ConfigJson_Malformed` | Malformed config.json doesn't crash, metadata fields empty |
| `TestIntegration_ConfigJson_PartialFields` | config.json with only `model_type`, other fields stay empty |
| `TestIntegration_ConfigJson_ConflictingValues` | config.json and generation_config.json have different `torch_dtype`, last-wins behavior |

**File: `pkg/backend/pull_integration_test.go`**

| Test | Description |
|------|-------------|
| `TestIntegration_Pull_HappyPath` | Pull manifest + blobs from mock registry, verify stored correctly |
| `TestIntegration_Pull_BlobAlreadyExists` | Pre-populate local blob, pull skips it |
| `TestIntegration_Pull_ConcurrentLayers` | 5 blobs in parallel, all stored correctly |

**File: `pkg/backend/push_integration_test.go`**

| Test | Description |
|------|-------------|
| `TestIntegration_Push_HappyPath` | Push manifest + blobs to mock registry, verify received |
| `TestIntegration_Push_BlobAlreadyExists` | Mock registry reports blob exists, push skips re-upload and re-tags |

**File: `cmd/modelfile/generate_integration_test.go`**

| Test | Description |
|------|-------------|
| `TestIntegration_CLI_Generate_BasicFlags` | CLI with `--name`, `--arch`, `--family` produces correct Modelfile |
| `TestIntegration_CLI_Generate_OutputAndOverwrite` | `--output` writes to specified dir; without `--overwrite` fails if exists |
| `TestIntegration_CLI_Generate_MutualExclusion` | `--model-url` with path arg returns error |

### Dimension 2: Network Errors (no build tag)

**File: `pkg/backend/pull_integration_test.go`**

| Test | Description |
|------|-------------|
| `TestIntegration_Pull_RetryOnTransientError` | `FailOnNthRequest: 2`, third request succeeds; verify retry works |
| `TestIntegration_Pull_RetryExhausted` | `StatusCodeOverride: 500` on all requests; verify clean error after retries |
| `TestIntegration_Pull_AuthError_NoRetry` | `StatusCodeOverride: 401`; verify immediate failure, no retry |
| `TestIntegration_Pull_RateLimited` | `StatusCodeOverride: 429`; verify backoff retry behavior |
| `TestIntegration_Pull_ContextTimeout` | `LatencyPerRequest: 30s` + short context deadline; verify context cancelled error |
| `TestIntegration_Pull_PartialResponse` | `FailAfterNBytes: 1024`; verify error propagation, no corrupted data stored |
| `TestIntegration_Pull_ManifestOK_BlobFails` | `PathFaults` on specific blob digest; verify error mentions the failed digest |

**File: `pkg/backend/push_integration_test.go`**

| Test | Description |
|------|-------------|
| `TestIntegration_Push_RetryOnTransientError` | Transient 500 on blob push, verify retry succeeds |
| `TestIntegration_Push_ManifestPushFails` | Manifest endpoint returns 500; verify error after blob push succeeds |

### Dimension 3: Resource Leak Detection (no build tag)

**File: `pkg/backend/push_integration_test.go`**

| Test | Description |
|------|-------------|
| `TestKnownBug_Push_ReadCloserLeak` | Push with blob upload failure; TrackingReadCloser asserts Close() called. **Known bug: push.go:175 does not close ReadCloser on error path.** |

**File: `pkg/backend/pull_integration_test.go`**

| Test | Description |
|------|-------------|
| `TestIntegration_Pull_ReaderClosedOnError` | Mock registry returns error mid-blob; verify manifest and blob readers all closed |
| `TestIntegration_Pull_TempDirCleanup` | Pull with download to temp dir then error; verify temp dir removed |

### Dimension 4: Concurrency Safety (no build tag)

Run with `go test -race`.

**File: `internal/pb/pb_integration_test.go`**

| Test | Description |
|------|-------------|
| `TestKnownBug_DisableProgress_DataRace` | Concurrent `SetDisableProgress()` + `Add()` from multiple goroutines. **Known bug: pb.go global flag has no atomic protection.** |
| `TestIntegration_ProgressBar_ConcurrentUpdates` | Multiple goroutines calling `Add()`, `Abort()`, `Complete()` simultaneously; no panic |

**File: `pkg/backend/pull_integration_test.go`**

| Test | Description |
|------|-------------|
| `TestIntegration_Pull_ConcurrentPartialFailure` | 5 blobs, 2 fail; verify errgroup cancels remaining, no goroutine hang |
| `TestIntegration_Pull_ConcurrentContextCancel` | Cancel context during concurrent pull; verify all goroutines exit |

### Dimension 5: Stress Tests (`//go:build stress`)

**File: `pkg/modelfile/modelfile_stress_test.go`**

| Test | Description |
|------|-------------|
| `TestStress_NearMaxFileCount` | 2040 files (near MaxWorkspaceFileCount=2048); verify success |
| `TestStress_DeeplyNestedDirs` | 100-level nested directory; verify no stack overflow |

**File: `pkg/backend/pull_integration_test.go` (stress-tagged section)**

| Test | Description |
|------|-------------|
| `TestStress_Pull_ManyLayers` | 50 concurrent blobs; verify errgroup scheduling, no deadlock |
| `TestStress_Pull_RepeatedCycles` | 100x pull loop; verify goroutine count stays stable via `runtime.NumGoroutine()` |

### Dimension 6: Data Integrity (no build tag)

**File: `pkg/backend/pull_integration_test.go`**

| Test | Description |
|------|-------------|
| `TestIntegration_Pull_TruncatedBlob` | Mock returns fewer bytes than Content-Length; verify digest validation catches it |
| `TestIntegration_Pull_CorruptedBlob` | Mock returns wrong bytes; verify digest mismatch error |

**File: `pkg/backend/push_integration_test.go`**

| Test | Description |
|------|-------------|
| `TestIntegration_Push_VerifyBlobIntegrity` | After push, verify mock registry received exact bytes matching source digest |

### Dimension 7: Graceful Shutdown (no build tag)

**File: `pkg/backend/pull_integration_test.go`**

| Test | Description |
|------|-------------|
| `TestIntegration_Pull_ContextCancelMidDownload` | Cancel context while blob download in progress; verify goroutines exit, temp files cleaned |

**File: `pkg/backend/push_integration_test.go`**

| Test | Description |
|------|-------------|
| `TestIntegration_Push_ContextCancelMidUpload` | Cancel context during blob upload; verify clean exit |

### Dimension 8: Idempotency & Config Boundary (no build tag)

**File: `pkg/backend/pull_integration_test.go`**

| Test | Description |
|------|-------------|
| `TestIntegration_Pull_Idempotent` | Pull twice; second pull makes zero blob fetches |

**File: `pkg/backend/push_integration_test.go`**

| Test | Description |
|------|-------------|
| `TestIntegration_Push_Idempotent` | Push twice; second push skips all blobs |

**File: `cmd/cli_integration_test.go`**

| Test | Description |
|------|-------------|
| `TestIntegration_CLI_ConcurrencyZero` | `--concurrency=0` returns validation error |
| `TestIntegration_CLI_ConcurrencyNegative` | `--concurrency=-1` returns validation error |
| `TestIntegration_CLI_ExtractFromRemoteNoDir` | `--extract-from-remote` without `--extract-dir` returns error |

## Known Bug Tests

Tests prefixed with `TestKnownBug_` are expected to fail. They document real bugs discovered during analysis:

| Test | Bug Location | Description |
|------|-------------|-------------|
| `TestKnownBug_Push_ReadCloserLeak` | `push.go:175-189` | PullBlob ReadCloser not closed on error path |
| `TestKnownBug_DisableProgress_DataRace` | `pb.go:33-40` | Global disableProgress flag has no atomic protection |

These tests serve as regression documentation. Each should reference a tracked GitHub issue in its failure message.

## File Structure

```
test/helpers/
  mockregistry.go
  mockregistry_test.go
  tracking.go

pkg/modelfile/
  modelfile_integration_test.go        # Dimensions 1
  modelfile_stress_test.go             # Dimension 5 (//go:build stress)

pkg/backend/
  pull_integration_test.go             # Dimensions 1,2,3,4,5,6,7,8
  push_integration_test.go             # Dimensions 1,2,3,6,7,8
  fetch_integration_test.go            # Dimensions 1,2

internal/pb/
  pb_integration_test.go               # Dimension 4

cmd/modelfile/
  generate_integration_test.go         # Dimension 1

cmd/
  cli_integration_test.go              # Dimension 8
```

## Conventions

- **Naming:** `TestIntegration_*` for normal tests, `TestKnownBug_*` for documented bugs, `TestStress_*` for stress tests
- **Isolation:** Each test creates its own temp dir / mock server; no shared mutable state
- **Build tags:** Fast tests have no tag (`go test ./...`); stress tests require `//go:build stress`
- **Race detection:** All concurrency tests must pass `go test -race`
- **Known bugs:** Let tests fail with descriptive message + issue link; do not use `t.Skip`

## Dependencies

No new external dependencies. Uses:
- `net/http/httptest` (stdlib)
- `github.com/stretchr/testify` (already in go.mod)
- `runtime` (stdlib, for goroutine counting in stress tests)
- `sync/atomic` (stdlib, for TrackingReadCloser)
