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

// Package retrypolicy provides retry behavior for blob transfer operations.
//
// The package decouples two timing concerns that are commonly conflated:
//
//   - Per-attempt timeout: how long a single transfer attempt may take.
//     This scales with file size, since larger files need longer to transfer.
//
//   - Retry policy: how many attempts to make and how long to wait between
//     them. This is bounded by attempt count and per-sleep cap; it does not
//     scale with file size, because transient-failure recovery time is
//     independent of payload size.
//
// Earlier designs used a single wall-clock budget covering both transfer
// time and retry waits, which made retries scarce precisely when networks
// were slow. The current design fixes that by giving each concern its own
// knob.
package retrypolicy

import (
	"context"
	"errors"
	"math"
	"regexp"
	"strings"
	"time"

	retry "github.com/avast/retry-go/v4"
	humanize "github.com/dustin/go-humanize"
	log "github.com/sirupsen/logrus"
)

const (
	// DefaultMaxAttempts is the total number of attempts (initial + retries).
	DefaultMaxAttempts = 6

	// DefaultInitialDelay is the first sleep between attempts.
	DefaultInitialDelay = 5 * time.Second

	// DefaultMaxBackoff caps a single sleep between attempts. It does not
	// scale with file size: transient outages have a payload-independent
	// duration distribution.
	DefaultMaxBackoff = 2 * time.Minute

	// DefaultMaxJitter is the upper bound on randomized jitter added to each
	// sleep.
	DefaultMaxJitter = 5 * time.Second

	// minThroughput is the assumed worst-case usable throughput when
	// computing per-attempt timeouts. Networks slower than this are out of
	// scope; a programmatic embedder can raise the ceiling by setting
	// Config.PerAttemptTimeout explicitly (modctl does not expose this as a
	// CLI flag today).
	minThroughput = 10 * (1 << 20) // 10 MiB/s

	// safetyFactor multiplies the ideal transfer time when sizing
	// per-attempt timeouts, leaving headroom for protocol overhead and
	// short-lived speed dips.
	safetyFactor = 2

	// minPerAttemptTimeout is the floor for derived per-attempt timeouts —
	// small blobs (manifests, configs) still need enough time for TLS,
	// auth, and slow-start.
	minPerAttemptTimeout = 5 * time.Minute

	// maxPerAttemptTimeout is the ceiling. Very large files (e.g. 100GB+
	// LLM shards) will still hit this cap, after which the user should
	// override via Config.PerAttemptTimeout.
	maxPerAttemptTimeout = 8 * time.Hour
)

// Config holds retry parameters for programmatic embedders. modctl does not
// currently wire these to CLI flags, so end users get the defaults; the struct
// exists so callers embedding this package (and tests) can override behavior.
//
// The zero value is valid and yields production defaults: 6 attempts, with
// per-attempt timeout derived from file size and exponential backoff up to
// DefaultMaxBackoff.
type Config struct {
	// MaxAttempts is the total number of attempts (initial + retries).
	// 0 means "use DefaultMaxAttempts". Set to 1 to disable retries
	// (single attempt, no retry on failure).
	MaxAttempts int

	// PerAttemptTimeout is the maximum duration for a single attempt.
	//   0  → derive from file size (see ComputePerAttemptTimeout).
	//   <0 → no per-attempt timeout (caller fully controls deadlines).
	//   >0 → use this value verbatim.
	PerAttemptTimeout time.Duration

	// InitialDelay overrides the first inter-attempt sleep. 0 = default.
	// Primarily for tests.
	InitialDelay time.Duration

	// MaxBackoff overrides the per-sleep cap. 0 = default.
	MaxBackoff time.Duration

	// MaxJitter: -1 = no jitter, 0 = default, >0 = override. For tests.
	MaxJitter time.Duration
}

// DoOpts configures a single Do call.
type DoOpts struct {
	// FileSize sizes the per-attempt timeout when Config.PerAttemptTimeout
	// is unset. May be 0 for non-blob operations (manifest, config); the
	// timeout will then clamp to minPerAttemptTimeout.
	FileSize int64

	// FileName is logged on each retry.
	FileName string

	// Config is the user-supplied policy. nil means defaults.
	Config *Config

	// OnRetry is invoked before each sleep (after a failed attempt) so
	// callers can update progress UI. attempt is 1-based; reason is a
	// short label from ShortReason. backoff is the deterministic base delay;
	// the actual sleep may add up to Config.MaxJitter.
	OnRetry func(attempt uint, reason string, backoff time.Duration)
}

// Do executes fn with retry. Each attempt runs under its own deadline
// derived from PerAttemptTimeout (or file size); retries are bounded by
// MaxAttempts and exponential backoff capped at MaxBackoff. The parent ctx
// is honored for user-initiated cancellation only — its expiry is not
// coupled to per-attempt transfer time.
func Do(ctx context.Context, fn func(ctx context.Context) error, opts DoOpts) error {
	cfg := opts.Config
	if cfg == nil {
		cfg = &Config{}
	}

	perAttemptTimeout := cfg.PerAttemptTimeout
	switch {
	case perAttemptTimeout == 0:
		perAttemptTimeout = ComputePerAttemptTimeout(opts.FileSize)
	case perAttemptTimeout < 0:
		perAttemptTimeout = 0 // disabled
	}

	// runAttempt applies the per-attempt deadline. retry-go calls this
	// for each attempt; if MaxAttempts == 1 the loop exits after one
	// invocation (equivalent to "no retry").
	runAttempt := func() error {
		attemptCtx := ctx
		if perAttemptTimeout > 0 {
			var cancel context.CancelFunc
			attemptCtx, cancel = context.WithTimeout(ctx, perAttemptTimeout)
			defer cancel()
		}
		return fn(attemptCtx)
	}

	maxAttempts := cfg.MaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = DefaultMaxAttempts
	}

	initialDelay := cfg.InitialDelay
	if initialDelay <= 0 {
		initialDelay = DefaultInitialDelay
	}

	maxBackoff := cfg.MaxBackoff
	if maxBackoff <= 0 {
		maxBackoff = DefaultMaxBackoff
	}

	jitter := DefaultMaxJitter
	if cfg.MaxJitter < 0 {
		jitter = 0
	} else if cfg.MaxJitter > 0 {
		jitter = cfg.MaxJitter
	}

	sizeStr := humanizeBytes(opts.FileSize)
	startTime := time.Now()

	// retry-go's zero-config default is CombineDelay(BackOffDelay, RandomDelay);
	// passing BackOffDelay alone silently drops the jitter term, so MaxJitter
	// would be dead and concurrent layers failing together would retry in
	// lockstep against a rate-limited registry. Re-add RandomDelay only when
	// jitter > 0 — RandomDelay calls rand.Int63n(maxJitter), which panics when
	// maxJitter == 0.
	delayType := retry.BackOffDelay
	if jitter > 0 {
		delayType = retry.CombineDelay(retry.BackOffDelay, retry.RandomDelay)
	}

	return retry.Do(
		runAttempt,
		retry.Attempts(uint(maxAttempts)),
		retry.Context(ctx),
		retry.DelayType(delayType),
		retry.Delay(initialDelay),
		retry.MaxDelay(maxBackoff),
		retry.MaxJitter(jitter),
		retry.LastErrorOnly(true),
		retry.RetryIf(func(err error) bool {
			// Per-attempt timeout fired but parent ctx is alive: this is a
			// transient transfer timeout, not a user cancellation. Retry.
			if errors.Is(err, context.DeadlineExceeded) && ctx.Err() == nil {
				return true
			}
			retryable := IsRetryable(err)
			if !retryable {
				log.WithFields(log.Fields{
					"file":  opts.FileName,
					"size":  sizeStr,
					"error": err.Error(),
				}).Error("[RETRY] non-retryable error, not retrying")
			}
			return retryable
		}),
		retry.OnRetry(func(n uint, err error) {
			// retry-go calls OnRetry with n = 0-based retry index. Convert
			// to 1-based for both the log and the user callback.
			attempt := n + 1
			elapsed := time.Since(startTime)

			// retry-go invokes OnRetry after every failed attempt, including
			// the final one where no further retry follows. Don't advertise a
			// "next retry" (backoff wait or user callback) that will never
			// happen; log a terminal give-up line instead.
			if attempt >= uint(maxAttempts) {
				log.WithFields(log.Fields{
					"file":           opts.FileName,
					"size":           sizeStr,
					"error":          err.Error(),
					"max_attempts":   maxAttempts,
					"per_attempt_to": perAttemptTimeout.String(),
					"elapsed":        elapsed.Truncate(time.Second).String(),
				}).Warnf("[RETRY] attempt %d/%d for %q (%s) failed; giving up", attempt, maxAttempts, opts.FileName, sizeStr)
				return
			}

			backoff := computeBackoff(attempt, initialDelay, maxBackoff)
			log.WithFields(log.Fields{
				"file":           opts.FileName,
				"size":           sizeStr,
				"error":          err.Error(),
				"max_attempts":   maxAttempts,
				"max_backoff":    maxBackoff.String(),
				"per_attempt_to": perAttemptTimeout.String(),
				"base_backoff":   backoff.Truncate(time.Second).String(),
				"max_jitter":     jitter.String(),
				"elapsed":        elapsed.Truncate(time.Second).String(),
			}).Warnf("[RETRY] attempt %d/%d for %q (%s)", attempt, maxAttempts, opts.FileName, sizeStr)

			if opts.OnRetry != nil {
				reason := ShortReason(err)
				opts.OnRetry(attempt, reason, backoff)
			}
		}),
	)
}

// ComputePerAttemptTimeout estimates a single-attempt transfer deadline from
// file size, assuming minThroughput as a worst-case-but-usable rate and
// applying safetyFactor for headroom. The result is clamped to
// [minPerAttemptTimeout, maxPerAttemptTimeout].
//
// Examples (rounded):
//
//	1 GB   →  5 min   (floor)
//	10 GB  → 34 min
//	70 GB  →  ~4 h
//	140 GB →  ~8 h    (ceiling)
//
// fileSize <= 0 returns minPerAttemptTimeout.
func ComputePerAttemptTimeout(fileSize int64) time.Duration {
	if fileSize <= 0 {
		return minPerAttemptTimeout
	}
	secs := float64(fileSize) / float64(minThroughput) * safetyFactor
	t := time.Duration(secs * float64(time.Second))
	if t < minPerAttemptTimeout {
		return minPerAttemptTimeout
	}
	if t > maxPerAttemptTimeout {
		return maxPerAttemptTimeout
	}
	return t
}

// computeBackoff estimates the backoff duration for display purposes.
// It mirrors retry-go's exponential schedule (without jitter).
// attempt is 1-based: the first sleep (after attempt 1 fails) is
// initialDelay, the second is 2*initialDelay, capped at maxDelay.
func computeBackoff(attempt uint, initial, maxDelay time.Duration) time.Duration {
	if attempt == 0 {
		return initial
	}
	backoff := time.Duration(float64(initial) * math.Pow(2, float64(attempt-1)))
	if backoff > maxDelay {
		backoff = maxDelay
	}
	return backoff
}

// httpStatusPattern matches ORAS-style error messages that embed HTTP status codes.
var httpStatusPattern = regexp.MustCompile(`response status code (\d{3})`)

// transientErrSubstrings are fragments that mark a transient transport/network
// failure worth retrying. The set is intentionally conservative; anything not
// matched here (and not a retryable HTTP status) is treated as permanent. It is
// the single place to tune the transient classification.
var transientErrSubstrings = []string{
	"i/o timeout",
	"connection timed out",
	"TLS handshake timeout",
	"connection reset by peer",
	"connection refused",
	"broken pipe",
	"no route to host",
	"network is unreachable",
	"use of closed network connection",
	"server misbehaving", // transient DNS resolution failure
	"temporary failure",  // e.g. "temporary failure in name resolution"
	"EOF",                // stream truncated/dropped mid-transfer
}

// IsRetryable reports whether err is a transient failure worth retrying.
//
// It uses an allowlist: only recognized-transient errors (5xx/408/429 HTTP
// responses, or a known network/transport failure) are retried. Everything
// else — including unclassified errors — is treated as permanent. Retrying a
// deterministic failure (bad digest, invalid manifest, local I/O error) would
// otherwise burn the full per-attempt timeout on every attempt for no benefit.
//
// context.Canceled and bare context.DeadlineExceeded are not retryable here:
// the Do loop independently re-classifies a per-attempt timeout while the
// parent context is still alive as retryable, so this function only sees
// genuine cancellation.
func IsRetryable(err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	errMsg := err.Error()

	// HTTP status codes embedded in ORAS-style messages: only 5xx and the
	// retry-after-style 408/429 are transient; other 4xx are deterministic.
	if matches := httpStatusPattern.FindStringSubmatch(errMsg); len(matches) == 2 {
		code := matches[1]
		return code[0] == '5' || code == "408" || code == "429"
	}

	for _, s := range transientErrSubstrings {
		if strings.Contains(errMsg, s) {
			return true
		}
	}

	// Unclassified errors are treated as permanent: a single retry that cannot
	// succeed still costs a full per-attempt timeout.
	log.WithField("error", errMsg).Debug("[RETRY] error not classified as transient; not retrying")
	return false
}

// ShortReason extracts a brief human-readable label from an error for
// progress bar display.
func ShortReason(err error) string {
	if err == nil {
		return ""
	}

	errMsg := err.Error()

	if matches := httpStatusPattern.FindStringSubmatch(errMsg); len(matches) == 2 {
		return "HTTP " + matches[1]
	}

	switch {
	case strings.Contains(errMsg, "i/o timeout"):
		return "i/o timeout"
	case strings.Contains(errMsg, "connection reset by peer"):
		return "conn reset"
	case strings.Contains(errMsg, "connection refused"):
		return "conn refused"
	case strings.Contains(errMsg, "broken pipe"):
		return "broken pipe"
	case strings.Contains(errMsg, "EOF"):
		return "EOF"
	case errors.Is(err, context.DeadlineExceeded):
		return "attempt timeout"
	}

	return "unknown error"
}

// humanizeBytes converts a byte count to a human-readable IEC string
// (KiB/MiB/GiB, base 1024). It delegates to go-humanize — already a direct
// dependency (used by internal/pb) — so the unit labels match the 1024 divisor
// instead of reimplementing byte formatting.
func humanizeBytes(b int64) string {
	if b < 0 {
		b = 0
	}
	return humanize.IBytes(uint64(b))
}
