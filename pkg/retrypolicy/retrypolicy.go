/*
 *     Copyright 2024 The CNAI Authors
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

package retrypolicy

import (
	"context"
	"errors"
	"fmt"
	"math"
	"regexp"
	"strings"
	"time"

	retry "github.com/avast/retry-go/v4"
	log "github.com/sirupsen/logrus"
)

const (
	oneGB  = 1 << 30 // 1 GiB in bytes
	tenGB  = 10 << 30
	nineGB = tenGB - oneGB

	minMaxRetryTime = 10 * time.Minute
	maxMaxRetryTime = 60 * time.Minute

	minMaxBackoff = 1 * time.Minute
	maxMaxBackoff = 10 * time.Minute

	initialDelay = 5 * time.Second
	maxJitter    = 5 * time.Second

	// maxBackoff cap when derived from user-specified MaxRetryTime
	absoluteMaxBackoff = 10 * time.Minute
)

// Config holds user-configurable retry parameters from CLI flags.
type Config struct {
	MaxRetryTime time.Duration // 0 = dynamic based on file size
	NoRetry      bool          // disable retry entirely
}

// DoOpts configures a single Do call.
type DoOpts struct {
	FileSize int64  // for dynamic parameter calculation
	FileName string // for logging
	Config   *Config
	OnRetry  func(attempt uint, reason string, backoff time.Duration)
}

// Do executes fn with retry. It computes dynamic retry parameters from fileSize,
// creates an internal deadline context (and defers its cancel to prevent leak),
// sets up retry logging, and calls retry.Do.
// The parent ctx is only used for user-initiated cancellation.
func Do(ctx context.Context, fn func(ctx context.Context) error, opts DoOpts) error {
	cfg := opts.Config
	if cfg == nil {
		cfg = &Config{}
	}

	// NoRetry: call fn once with the parent context, return the result.
	if cfg.NoRetry {
		return fn(ctx)
	}

	maxRetryTime, maxBackoff := computeDynamicParams(opts.FileSize)

	// Override with user-specified MaxRetryTime if set.
	if cfg.MaxRetryTime > 0 {
		maxRetryTime = cfg.MaxRetryTime
		maxBackoff = cfg.MaxRetryTime / 6
		if maxBackoff > absoluteMaxBackoff {
			maxBackoff = absoluteMaxBackoff
		}
	}

	startTime := time.Now()
	deadlineCtx, deadlineCancel := context.WithDeadline(ctx, startTime.Add(maxRetryTime))
	defer deadlineCancel()

	sizeStr := humanizeBytes(opts.FileSize)

	return retry.Do(
		func() error {
			return fn(deadlineCtx)
		},
		retry.Attempts(0),
		retry.Context(deadlineCtx),
		retry.DelayType(retry.BackOffDelay),
		retry.Delay(initialDelay),
		retry.MaxDelay(maxBackoff),
		retry.MaxJitter(maxJitter),
		retry.LastErrorOnly(true),
		retry.WrapContextErrorWithLastError(true),
		retry.RetryIf(func(err error) bool {
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
			backoff := computeBackoff(n+1, initialDelay, maxBackoff)
			elapsed := time.Since(startTime)

			log.WithFields(log.Fields{
				"file":           opts.FileName,
				"size":           sizeStr,
				"error":          err.Error(),
				"max_retry_time": maxRetryTime.String(),
				"max_backoff":    maxBackoff.String(),
				"next_retry_in":  backoff.Truncate(time.Second).String(),
				"elapsed":        fmt.Sprintf("%s / %s", elapsed.Truncate(time.Second), maxRetryTime),
			}).Warnf("[RETRY] attempt %d for %q (%s)", n+1, opts.FileName, sizeStr)

			if opts.OnRetry != nil {
				reason := ShortReason(err)
				opts.OnRetry(n+1, reason, backoff)
			}
		}),
	)
}

// computeDynamicParams calculates maxRetryTime and maxBackoff based on file size.
//
// For files <= 1 GB: maxRetryTime=10min, maxBackoff=1min
// For files >= 10 GB: maxRetryTime=60min, maxBackoff=10min
// Linear interpolation between.
func computeDynamicParams(fileSize int64) (time.Duration, time.Duration) {
	ratio := float64(fileSize-oneGB) / float64(nineGB)
	if ratio < 0 {
		ratio = 0
	}
	if ratio > 1 {
		ratio = 1
	}

	maxRetryTime := minMaxRetryTime + time.Duration(ratio*float64(maxMaxRetryTime-minMaxRetryTime))
	maxBackoff := minMaxBackoff + time.Duration(ratio*float64(maxMaxBackoff-minMaxBackoff))

	return maxRetryTime, maxBackoff
}

// computeBackoff estimates the backoff duration for display purposes.
// It mirrors the exponential backoff calculation without jitter.
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

// IsRetryable returns true for transient errors that warrant a retry.
func IsRetryable(err error) bool {
	if err == nil {
		return false
	}

	// context.Canceled is never retryable — it means user/system cancellation.
	if errors.Is(err, context.Canceled) {
		return false
	}

	errMsg := err.Error()

	// Check for HTTP status codes embedded in error messages (ORAS style).
	if matches := httpStatusPattern.FindStringSubmatch(errMsg); len(matches) == 2 {
		code := matches[1]
		// 5xx server errors are retryable.
		if code[0] == '5' {
			return true
		}
		// 408 (Request Timeout) and 429 (Too Many Requests) are retryable.
		if code == "408" || code == "429" {
			return true
		}
		// Other 4xx are not retryable (401, 403, 404, etc.)
		return false
	}

	// Network-level transient errors.
	if strings.Contains(errMsg, "i/o timeout") {
		return true
	}
	if strings.Contains(errMsg, "connection reset by peer") {
		return true
	}
	if strings.Contains(errMsg, "connection refused") {
		return true
	}
	if strings.Contains(errMsg, "broken pipe") {
		return true
	}
	if strings.Contains(errMsg, "EOF") {
		return true
	}

	// Unknown errors default to retryable.
	return true
}

// ShortReason extracts a brief human-readable label from an error for progress bar display.
func ShortReason(err error) string {
	if err == nil {
		return ""
	}

	errMsg := err.Error()

	// Check for HTTP status codes.
	if matches := httpStatusPattern.FindStringSubmatch(errMsg); len(matches) == 2 {
		return "HTTP " + matches[1]
	}

	if strings.Contains(errMsg, "i/o timeout") {
		return "i/o timeout"
	}
	if strings.Contains(errMsg, "connection reset by peer") {
		return "conn reset"
	}
	if strings.Contains(errMsg, "connection refused") {
		return "conn refused"
	}
	if strings.Contains(errMsg, "broken pipe") {
		return "broken pipe"
	}
	if strings.Contains(errMsg, "EOF") {
		return "EOF"
	}

	return "unknown error"
}

// humanizeBytes converts a byte count to a human-readable string.
func humanizeBytes(b int64) string {
	const (
		kb = 1024
		mb = 1024 * kb
		gb = 1024 * mb
	)

	switch {
	case b >= gb:
		return fmt.Sprintf("%.1f GB", float64(b)/float64(gb))
	case b >= mb:
		return fmt.Sprintf("%.1f MB", float64(b)/float64(mb))
	case b >= kb:
		return fmt.Sprintf("%.1f KB", float64(b)/float64(kb))
	default:
		return fmt.Sprintf("%d B", b)
	}
}
