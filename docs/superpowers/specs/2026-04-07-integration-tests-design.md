# Integration Tests Design Spec

## Goal

Add comprehensive integration tests across modctl to cover functional correctness, stress scenarios, and error handling. Tests only — no functional code changes.

## Scope

8 test dimensions, ~35 test scenarios, covering `pkg/modelfile`, `pkg/backend`, `internal/pb`, and `cmd/` layers.

**Out of scope:** Security tests (path traversal, TLS, credential handling) — deferred to a future effort.

## Known Bugs Discovered

The following bugs were found during analysis. Each must be filed as a GitHub issue before implementation begins, and referenced by issue number in the corresponding `TestKnownBug_*` test.

| Bug | Location | Description | Severity | Issue |
|-----|----------|-------------|----------|-------|
| Push ReadCloser leak | `push.go:175-189` | `src.PullBlob()` returns `io.ReadCloser` that is never explicitly closed — on both success and error paths. The reader is wrapped with `io.NopCloser()` before passing to `dst.Blobs().Push()`, so even when push closes the wrapper, the original content is not closed. | Critical | [#491](https://github.com/modelpack/modctl/issues/491) |
| disableProgress data race | `internal/pb/pb.go:33-40` | Global `disableProgress` bool is written by `SetDisableProgress()` and read by `Add()` without any synchronization (no mutex, no atomic). Triggers data race under concurrent pull/push. | Critical | [#493](https://github.com/modelpack/modctl/issues/493) |
| splitReader goroutine leak | `pkg/backend/build/builder.go:413-430` | `splitReader()` spawns a goroutine that copies to a `MultiWriter` over two `PipeWriter`s. If either pipe reader is abandoned (e.g., interceptor fails), the goroutine blocks indefinitely on write. No context-based cancellation. | Major | [#492](https://github.com/modelpack/modctl/issues/492) |
| Auth errors retried unconditionally | `pull.go:116-129`, `retry.go:25-29` | `retry.Do()` retries all errors including 401/403. Auth errors should fail immediately. | Minor | [#494](https://github.com/modelpack/modctl/issues/494) |

## Test Infrastructure

### Mock OCI Registry (`test/helpers/mockregistry.go`)

A reusable `httptest`-based mock OCI registry server with fault injection. Centralizes the pattern already used in `fetch_test.go` into a shared helper.

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

Used in push leak tests where mock Storage's `PullBlob()` can return a tracked reader.

## Test Dimensions & Scenarios

### Dimension 1: Functional Correctness (no build tag)

Existing coverage considered:
- `fetch_test.go` already has 6 httptest scenarios for fetch — no new fetch happy path tests needed.
- `pkg/modelfile/modelfile_test.go` covers config.json parsing, content generation, workspace walking — only `ExcludePatterns` integration and CLI-level tests are new.

**File: `pkg/modelfile/modelfile_integration_test.go`**

| Test | Description |
|------|-------------|
| `TestIntegration_ExcludePatterns_SinglePattern` | `ExcludePatterns: ["*.log"]` through `NewModelfileByWorkspace` removes .log files |
| `TestIntegration_ExcludePatterns_MultiplePatterns` | `ExcludePatterns: ["*.log", "checkpoints/*"]` removes both |

**File: `pkg/backend/pull_integration_test.go`**

| Test | Description |
|------|-------------|
| `TestIntegration_Pull_HappyPath` | Pull manifest + blobs from mock registry via httptest, verify stored correctly |
| `TestIntegration_Pull_BlobAlreadyExists` | Pre-populate local blob, pull skips it |
| `TestIntegration_Pull_ConcurrentLayers` | 5 blobs in parallel, all stored correctly |

**File: `pkg/backend/push_integration_test.go`**

| Test | Description |
|------|-------------|
| `TestIntegration_Push_HappyPath` | Push manifest + blobs to mock registry via httptest, verify received |
| `TestIntegration_Push_BlobAlreadyExists` | Mock registry reports blob exists, push skips re-upload and re-tags |

**File: `cmd/modelfile/generate_integration_test.go`**

| Test | Description |
|------|-------------|
| `TestIntegration_CLI_Generate_BasicFlags` | CLI with `--name`, `--arch`, `--family` produces correct Modelfile |
| `TestIntegration_CLI_Generate_OutputAndOverwrite` | `--output` writes to specified dir; without `--overwrite` fails if exists |
| `TestIntegration_CLI_Generate_MutualExclusion` | `--model-url` with path arg returns error |

### Dimension 2: Network Errors

Existing coverage considered:
- `retry_test.go` tests the retry-go library in isolation (zero-delay, pure function). New tests cover retry behavior integrated with real HTTP through pull/push functions.
- `fetch_test.go` already covers error references and non-matching patterns. No new fetch error tests needed — pull/push cover the same retry.Do() codepath.

**Split:** Fast tests (context-based, no retry wait) run by default. Slow tests (real retry backoff) require `//go:build slowtest`.

**Fast tests (no build tag) — File: `pkg/backend/pull_integration_test.go`**

| Test | Description |
|------|-------------|
| `TestIntegration_Pull_ContextTimeout` | `LatencyPerRequest: 30s` + short context deadline (100ms); verify context cancelled, no retry wait |
| `TestIntegration_Pull_PartialResponse` | `FailAfterNBytes: 1024`; verify error propagation, no corrupted data stored |
| `TestIntegration_Pull_ManifestOK_BlobFails` | `PathFaults` on specific blob digest; verify error mentions the failed digest |

**Fast tests (no build tag) — File: `pkg/backend/push_integration_test.go`**

| Test | Description |
|------|-------------|
| `TestIntegration_Push_ManifestPushFails` | Manifest endpoint returns 500; verify error surfaces |

**Slow tests (`//go:build slowtest`) — File: `pkg/backend/pull_slowtest_test.go`**

Note: Current implementation applies `retry.Do(...)` unconditionally (`pull.go:116-129`) with 6 attempts, 5s initial delay, 60s max (`retry.go:25-29`). Auth errors (401) ARE retried — this is a design issue documented as a known-bug test.

| Test | Description |
|------|-------------|
| `TestSlow_Pull_RetryOnTransientError` | `FailOnNthRequest: 2`, third succeeds; verify retry works |
| `TestSlow_Pull_RetryExhausted` | `StatusCodeOverride: 500` all requests; verify clean error after 6 retries |
| `TestSlow_Pull_RateLimited` | `StatusCodeOverride: 429`; verify backoff retry |
| `TestKnownBug_Pull_AuthErrorStillRetries` | `StatusCodeOverride: 401`; assert request count == 6 (documents: auth errors should not retry). Reverse assertion — passes now, fails when fixed. |

**Slow tests (`//go:build slowtest`) — File: `pkg/backend/push_slowtest_test.go`**

| Test | Description |
|------|-------------|
| `TestSlow_Push_RetryOnTransientError` | Transient 500 on blob push, verify retry succeeds |

### Dimension 3: Resource Leak Detection (no build tag)

Existing coverage: None.

Scope adjustment: Pull reader/tempdir leak tests removed — Pull() does not create temp dirs (`pull.go:61-170`), and blob readers are created inside oras-go internals where TrackingReadCloser cannot be injected in black-box tests. Push leak tests ARE feasible because `src.PullBlob()` goes through mock Storage, allowing TrackingReadCloser injection.

**File: `pkg/backend/push_integration_test.go`**

| Test | Description |
|------|-------------|
| `TestKnownBug_Push_ReadCloserNotClosed_SuccessPath` | Push succeeds; TrackingReadCloser from mock PullBlob asserts Close() was NOT called (reverse assertion — documents the bug on success path). |
| `TestKnownBug_Push_ReadCloserNotClosed_ErrorPath` | Push fails (blob upload error); same assertion. Documents leak on error path. |

Both tests use reverse assertions: they pass today (asserting the leak exists). When the bug is fixed, they fail, prompting the developer to flip the assertion and remove the `KnownBug` prefix.

### Dimension 4: Concurrency Safety (no build tag)

Existing coverage: Zero concurrency tests anywhere in the project.

Run with `go test -race`.

**File: `internal/pb/pb_integration_test.go`**

| Test | Description |
|------|-------------|
| `TestKnownBug_DisableProgress_DataRace` | Concurrent `SetDisableProgress()` + `Add()` from multiple goroutines. Reverse assertion: run without `-race` and assert the code executes without panic (passes). The real detection is via `go test -race` which will report the data race. |
| `TestIntegration_ProgressBar_ConcurrentUpdates` | Multiple goroutines calling `Add()`, `Abort()`, `Complete()` simultaneously; no panic, no deadlock |

**File: `pkg/backend/pull_integration_test.go`**

| Test | Description |
|------|-------------|
| `TestIntegration_Pull_ConcurrentPartialFailure` | 5 blobs, 2 fail; verify errgroup cancels remaining, all goroutines exit, no hang |

### Dimension 5: Stress Tests (`//go:build stress`)

Existing coverage: `TestWorkspaceLimits` tests exceeding MaxWorkspaceFileCount (error case). New tests cover near-limit success and other stress vectors.

**File: `pkg/modelfile/modelfile_stress_test.go`**

| Test | Description |
|------|-------------|
| `TestStress_NearMaxFileCount` | 2040 files (near MaxWorkspaceFileCount=2048); verify success |
| `TestStress_DeeplyNestedDirs` | 100-level nested directory; verify no stack overflow |

**File: `pkg/backend/pull_stress_test.go`**

| Test | Description |
|------|-------------|
| `TestStress_Pull_ManyLayers` | 50 concurrent blobs; verify errgroup scheduling, no deadlock |
| `TestStress_Pull_RepeatedCycles` | 100x pull loop; verify goroutine count stable via `runtime.NumGoroutine()` |

### Dimension 6: Data Integrity (no build tag)

Existing coverage: None for pull/push data integrity.

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

Existing coverage: `retry_test.go` tests context cancel on retry library only.

**File: `pkg/backend/pull_integration_test.go`**

| Test | Description |
|------|-------------|
| `TestIntegration_Pull_ContextCancelMidDownload` | Cancel context while concurrent blob download in progress; verify all goroutines exit (covers both graceful shutdown and concurrent cancel) |

**File: `pkg/backend/push_integration_test.go`**

| Test | Description |
|------|-------------|
| `TestIntegration_Push_ContextCancelMidUpload` | Cancel context during blob upload; verify clean exit |

### Dimension 8: Idempotency & Config Boundary (no build tag)

Existing coverage: None.

**File: `pkg/backend/pull_integration_test.go`**

| Test | Description |
|------|-------------|
| `TestIntegration_Pull_Idempotent` | Pull twice; second pull makes zero blob fetches (verified via mock registry request count) |

**File: `pkg/backend/push_integration_test.go`**

| Test | Description |
|------|-------------|
| `TestIntegration_Push_Idempotent` | Push twice; second push skips all blob uploads |

**File: `cmd/cli_integration_test.go`**

| Test | Description |
|------|-------------|
| `TestIntegration_CLI_ConcurrencyZero` | `--concurrency=0` returns validation error |
| `TestIntegration_CLI_ConcurrencyNegative` | `--concurrency=-1` returns validation error |
| `TestIntegration_CLI_ExtractFromRemoteNoDir` | `--extract-from-remote` without `--extract-dir` returns error |

## Known Bug Test Strategy

Tests prefixed with `TestKnownBug_` use **reverse assertions**: they assert the buggy behavior EXISTS, so they pass in CI today. When the bug is fixed, the test fails, prompting the developer to:
1. Flip the assertion (e.g., `assert.False` → `assert.True`)
2. Remove the `KnownBug_` prefix
3. Close the linked GitHub issue

Example pattern:
```go
func TestKnownBug_Push_ReadCloserNotClosed_SuccessPath(t *testing.T) {
    // ... trigger push via mock storage + mock registry ...

    // BUG: content ReadCloser is never explicitly closed.
    // See: https://github.com/modelpack/modctl/issues/491
    assert.False(t, tracker.WasClosed(),
        "If this fails, the ReadCloser leak has been fixed! "+
        "Flip to assert.True and remove KnownBug prefix.")
}
```

## File Structure

```
test/helpers/
  mockregistry.go              # shared mock OCI registry with fault injection
  mockregistry_test.go         # self-tests for mock registry
  tracking.go                  # TrackingReadCloser and other resource trackers

pkg/modelfile/
  modelfile_integration_test.go   # Dim 1: ExcludePatterns integration (2 tests)
  modelfile_stress_test.go        # Dim 5: //go:build stress (2 tests)

pkg/backend/
  pull_integration_test.go        # Dim 1,2,4,6,7,8 (10 tests)
  push_integration_test.go        # Dim 1,2,3,6,7,8 (9 tests)
  pull_slowtest_test.go           # Dim 2: //go:build slowtest (4 tests)
  push_slowtest_test.go           # Dim 2: //go:build slowtest (1 test)
  pull_stress_test.go             # Dim 5: //go:build stress (2 tests)

internal/pb/
  pb_integration_test.go          # Dim 4: concurrency safety (2 tests)

cmd/modelfile/
  generate_integration_test.go    # Dim 1: CLI layer (3 tests)

cmd/
  cli_integration_test.go         # Dim 8: config boundary (3 tests)
```

## Conventions

- **Naming:** `TestIntegration_*` for normal, `TestKnownBug_*` for documented bugs (reverse assertion), `TestStress_*` for stress, `TestSlow_*` for retry-dependent
- **Isolation:** Each test creates its own temp dir / mock server; no shared mutable state
- **Build tags:** Default (`go test ./...`) for fast tests; `//go:build slowtest` for retry tests; `//go:build stress` for stress tests
- **Race detection:** All concurrency tests must pass `go test -race`
- **Known bugs:** Reverse assertion pattern; each test references a GitHub issue number in its failure message
- **Issue tracking:** All known bugs filed as GitHub issues before implementation begins

## Dependencies

No new external dependencies. Uses:
- `net/http/httptest` (stdlib)
- `github.com/stretchr/testify` (already in go.mod)
- `runtime` (stdlib, for goroutine counting in stress tests)
- `sync/atomic` (stdlib, for TrackingReadCloser)
