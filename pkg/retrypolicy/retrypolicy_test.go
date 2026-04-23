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
	"net"
	"sync/atomic"
	"testing"
	"time"
)

// --- helpers for tests ---

// timeoutError implements net.Error with Timeout() returning true.
type timeoutError struct {
	msg string
}

func (e *timeoutError) Error() string   { return e.msg }
func (e *timeoutError) Timeout() bool   { return true }
func (e *timeoutError) Temporary() bool { return true }

// --- computeDynamicParams tests ---

func TestComputeDynamicParams(t *testing.T) {
	tests := []struct {
		name            string
		fileSize        int64
		wantRetryTime   time.Duration
		wantMaxBackoff  time.Duration
	}{
		{
			name:           "zero bytes - clamped to minimum",
			fileSize:       0,
			wantRetryTime:  10 * time.Minute,
			wantMaxBackoff: 1 * time.Minute,
		},
		{
			name:           "500 MB - below 1 GB, clamped to minimum",
			fileSize:       500 * 1024 * 1024,
			wantRetryTime:  10 * time.Minute,
			wantMaxBackoff: 1 * time.Minute,
		},
		{
			name:           "1 GB - boundary, ratio=0, minimum values",
			fileSize:       1 << 30,
			wantRetryTime:  10 * time.Minute,
			wantMaxBackoff: 1 * time.Minute,
		},
		{
			name:           "5.5 GB - midpoint, interpolated values",
			fileSize:       int64(5.5 * float64(1<<30)),
			wantRetryTime:  35 * time.Minute,
			wantMaxBackoff: 5*time.Minute + 30*time.Second,
		},
		{
			name:           "10 GB - boundary, ratio=1, maximum values",
			fileSize:       10 << 30,
			wantRetryTime:  60 * time.Minute,
			wantMaxBackoff: 10 * time.Minute,
		},
		{
			name:           "20 GB - above 10 GB, clamped to maximum",
			fileSize:       20 << 30,
			wantRetryTime:  60 * time.Minute,
			wantMaxBackoff: 10 * time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotRetryTime, gotMaxBackoff := computeDynamicParams(tt.fileSize)

			// Allow 1 second tolerance for floating point interpolation.
			retryTimeDiff := absDuration(gotRetryTime - tt.wantRetryTime)
			if retryTimeDiff > time.Second {
				t.Errorf("maxRetryTime = %v, want %v (diff %v)", gotRetryTime, tt.wantRetryTime, retryTimeDiff)
			}

			backoffDiff := absDuration(gotMaxBackoff - tt.wantMaxBackoff)
			if backoffDiff > time.Second {
				t.Errorf("maxBackoff = %v, want %v (diff %v)", gotMaxBackoff, tt.wantMaxBackoff, backoffDiff)
			}
		})
	}
}

func absDuration(d time.Duration) time.Duration {
	if d < 0 {
		return -d
	}
	return d
}

// --- IsRetryable tests ---

func TestIsRetryable(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
		{
			name: "context.Canceled",
			err:  context.Canceled,
			want: false,
		},
		{
			name: "wrapped context.Canceled",
			err:  fmt.Errorf("operation failed: %w", context.Canceled),
			want: false,
		},
		{
			name: "context.DeadlineExceeded",
			err:  context.DeadlineExceeded,
			want: false,
		},
		{
			name: "wrapped context.DeadlineExceeded",
			err:  fmt.Errorf("operation timed out: %w", context.DeadlineExceeded),
			want: false,
		},
		{
			name: "HTTP 500 server error",
			err:  fmt.Errorf("PUT /blobs/uploads: response status code 500: internal server error"),
			want: true,
		},
		{
			name: "HTTP 502 bad gateway",
			err:  fmt.Errorf("response status code 502: bad gateway"),
			want: true,
		},
		{
			name: "HTTP 503 service unavailable",
			err:  fmt.Errorf("response status code 503: service unavailable"),
			want: true,
		},
		{
			name: "HTTP 408 request timeout",
			err:  fmt.Errorf("response status code 408: request timeout"),
			want: true,
		},
		{
			name: "HTTP 429 too many requests",
			err:  fmt.Errorf("response status code 429: too many requests"),
			want: true,
		},
		{
			name: "HTTP 401 unauthorized - not retryable",
			err:  fmt.Errorf("response status code 401: unauthorized"),
			want: false,
		},
		{
			name: "HTTP 403 forbidden - not retryable",
			err:  fmt.Errorf("response status code 403: access denied"),
			want: false,
		},
		{
			name: "HTTP 404 not found - not retryable",
			err:  fmt.Errorf("response status code 404: not found"),
			want: false,
		},
		{
			name: "i/o timeout",
			err: &net.OpError{
				Op:  "read",
				Net: "tcp",
				Err: &timeoutError{msg: "i/o timeout"},
			},
			want: true,
		},
		{
			name: "i/o timeout in wrapped error message",
			err:  fmt.Errorf("read tcp 10.0.0.1:1234->10.0.0.2:443: i/o timeout"),
			want: true,
		},
		{
			name: "connection reset by peer",
			err:  fmt.Errorf("read tcp: connection reset by peer"),
			want: true,
		},
		{
			name: "connection refused",
			err:  fmt.Errorf("dial tcp 10.0.0.1:443: connection refused"),
			want: true,
		},
		{
			name: "broken pipe",
			err:  fmt.Errorf("write tcp: broken pipe"),
			want: true,
		},
		{
			name: "EOF",
			err:  fmt.Errorf("unexpected EOF"),
			want: true,
		},
		{
			name: "permission denied - not retryable",
			err:  fmt.Errorf("open /data/model.bin: permission denied"),
			want: false,
		},
		{
			name: "no space left on device - not retryable",
			err:  fmt.Errorf("write /data/model.bin: no space left on device"),
			want: false,
		},
		{
			name: "no such file or directory - not retryable",
			err:  fmt.Errorf("open /data/model.bin: no such file or directory"),
			want: false,
		},
		{
			name: "unknown error - defaults to retryable",
			err:  errors.New("something totally unexpected happened"),
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsRetryable(tt.err)
			if got != tt.want {
				t.Errorf("IsRetryable(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

// --- ShortReason tests ---

func TestShortReason(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{
			name: "nil error",
			err:  nil,
			want: "",
		},
		{
			name: "HTTP 500",
			err:  fmt.Errorf("response status code 500: internal server error"),
			want: "HTTP 500",
		},
		{
			name: "HTTP 502",
			err:  fmt.Errorf("response status code 502: bad gateway"),
			want: "HTTP 502",
		},
		{
			name: "HTTP 429",
			err:  fmt.Errorf("response status code 429: too many requests"),
			want: "HTTP 429",
		},
		{
			name: "HTTP 408",
			err:  fmt.Errorf("response status code 408: request timeout"),
			want: "HTTP 408",
		},
		{
			name: "i/o timeout",
			err:  fmt.Errorf("read tcp: i/o timeout"),
			want: "i/o timeout",
		},
		{
			name: "connection reset",
			err:  fmt.Errorf("read tcp: connection reset by peer"),
			want: "conn reset",
		},
		{
			name: "connection refused",
			err:  fmt.Errorf("dial tcp: connection refused"),
			want: "conn refused",
		},
		{
			name: "broken pipe",
			err:  fmt.Errorf("write tcp: broken pipe"),
			want: "broken pipe",
		},
		{
			name: "EOF",
			err:  fmt.Errorf("unexpected EOF"),
			want: "EOF",
		},
		{
			name: "unknown error",
			err:  errors.New("some weird error"),
			want: "unknown error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ShortReason(tt.err)
			if got != tt.want {
				t.Errorf("ShortReason(%v) = %q, want %q", tt.err, got, tt.want)
			}
		})
	}
}

// --- Do tests ---

func TestDo_SuccessFirstAttempt(t *testing.T) {
	callCount := int32(0)
	err := Do(context.Background(), func(ctx context.Context) error {
		atomic.AddInt32(&callCount, 1)
		return nil
	}, DoOpts{
		FileSize: 100,
		FileName: "test.bin",
	})

	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if atomic.LoadInt32(&callCount) != 1 {
		t.Fatalf("expected fn to be called once, got %d", callCount)
	}
}

func TestDo_RetryOnTransientError(t *testing.T) {
	callCount := int32(0)
	err := Do(context.Background(), func(ctx context.Context) error {
		n := atomic.AddInt32(&callCount, 1)
		if n < 3 {
			return fmt.Errorf("response status code 500: internal server error")
		}
		return nil
	}, DoOpts{
		FileSize: 100,
		FileName: "test.bin",
		Config:   &Config{InitialDelay: 10 * time.Millisecond, MaxJitter: -1},
	})

	if err != nil {
		t.Fatalf("expected nil error after retries, got %v", err)
	}
	if atomic.LoadInt32(&callCount) != 3 {
		t.Fatalf("expected fn to be called 3 times, got %d", callCount)
	}
}

func TestDo_NoRetryOnPermanentError(t *testing.T) {
	callCount := int32(0)
	err := Do(context.Background(), func(ctx context.Context) error {
		atomic.AddInt32(&callCount, 1)
		return fmt.Errorf("response status code 404: not found")
	}, DoOpts{
		FileSize: 100,
		FileName: "test.bin",
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if atomic.LoadInt32(&callCount) != 1 {
		t.Fatalf("expected fn to be called once for non-retryable error, got %d", callCount)
	}
}

func TestDo_NoRetryConfig(t *testing.T) {
	callCount := int32(0)
	err := Do(context.Background(), func(ctx context.Context) error {
		atomic.AddInt32(&callCount, 1)
		return fmt.Errorf("response status code 500: internal server error")
	}, DoOpts{
		FileSize: 100,
		FileName: "test.bin",
		Config:   &Config{NoRetry: true},
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if atomic.LoadInt32(&callCount) != 1 {
		t.Fatalf("expected fn to be called once with NoRetry, got %d", callCount)
	}
}

func TestDo_ParentContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	callCount := int32(0)
	err := Do(ctx, func(ctx context.Context) error {
		n := atomic.AddInt32(&callCount, 1)
		if n == 1 {
			// Cancel the parent context after the first attempt.
			cancel()
			return fmt.Errorf("response status code 500: internal server error")
		}
		return nil
	}, DoOpts{
		FileSize: 100,
		FileName: "test.bin",
		Config:   &Config{InitialDelay: 10 * time.Millisecond, MaxJitter: -1},
	})

	if err == nil {
		t.Fatal("expected error from cancelled context, got nil")
	}
}

func TestDo_OnRetryCallback(t *testing.T) {
	var retryAttempts []uint
	var retryReasons []string

	callCount := int32(0)
	err := Do(context.Background(), func(ctx context.Context) error {
		n := atomic.AddInt32(&callCount, 1)
		if n < 3 {
			return fmt.Errorf("response status code 500: internal server error")
		}
		return nil
	}, DoOpts{
		FileSize: 100,
		FileName: "test.bin",
		Config:   &Config{InitialDelay: 10 * time.Millisecond, MaxJitter: -1},
		OnRetry: func(attempt uint, reason string, backoff time.Duration) {
			retryAttempts = append(retryAttempts, attempt)
			retryReasons = append(retryReasons, reason)
		},
	})

	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if len(retryAttempts) != 2 {
		t.Fatalf("expected 2 OnRetry calls, got %d", len(retryAttempts))
	}
	if retryAttempts[0] != 1 || retryAttempts[1] != 2 {
		t.Errorf("expected attempts [1, 2], got %v", retryAttempts)
	}
	if retryReasons[0] != "HTTP 500" || retryReasons[1] != "HTTP 500" {
		t.Errorf("expected reasons [HTTP 500, HTTP 500], got %v", retryReasons)
	}
}

func TestDo_ConfigMaxRetryTimeOverride(t *testing.T) {
	// Use a very short MaxRetryTime to ensure the retry loop terminates quickly.
	callCount := int32(0)
	start := time.Now()
	err := Do(context.Background(), func(ctx context.Context) error {
		atomic.AddInt32(&callCount, 1)
		return fmt.Errorf("response status code 500: internal server error")
	}, DoOpts{
		FileSize: 100,
		FileName: "test.bin",
		Config: &Config{
			MaxRetryTime: 1 * time.Second,
			InitialDelay: 50 * time.Millisecond,
			MaxJitter:    -1,
		},
	})

	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected error after retry timeout, got nil")
	}
	// Should have run for approximately MaxRetryTime (1s), not the dynamic default (10min).
	if elapsed > 5*time.Second {
		t.Errorf("expected retry to terminate within ~1s, but elapsed %v", elapsed)
	}
	if atomic.LoadInt32(&callCount) < 2 {
		t.Errorf("expected at least 2 attempts, got %d", callCount)
	}
}

// --- humanizeBytes tests ---

func TestHumanizeBytes(t *testing.T) {
	tests := []struct {
		input int64
		want  string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{1073741824, "1.0 GB"},
		{int64(5.5 * float64(1<<30)), "5.5 GB"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := humanizeBytes(tt.input)
			if got != tt.want {
				t.Errorf("humanizeBytes(%d) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// --- computeBackoff tests ---

func TestComputeBackoff(t *testing.T) {
	initial := 5 * time.Second
	maxDelay := 1 * time.Minute

	// retry-go's OnRetry always supplies a 1-based attempt number, so 0 is
	// not a value the function is ever called with in production. Start from 1.

	// attempt 1 => 5s * 2^0 = 5s
	if got := computeBackoff(1, initial, maxDelay); got != 5*time.Second {
		t.Errorf("attempt 1: got %v, want %v", got, 5*time.Second)
	}

	// attempt 2 => 5s * 2^1 = 10s
	if got := computeBackoff(2, initial, maxDelay); got != 10*time.Second {
		t.Errorf("attempt 2: got %v, want %v", got, 10*time.Second)
	}

	// attempt 4 => 5s * 2^3 = 40s
	if got := computeBackoff(4, initial, maxDelay); got != 40*time.Second {
		t.Errorf("attempt 4: got %v, want %v", got, 40*time.Second)
	}

	// attempt 5 => 5s * 2^4 = 80s, but capped at 60s
	if got := computeBackoff(5, initial, maxDelay); got != maxDelay {
		t.Errorf("attempt 5: got %v, want %v (capped)", got, maxDelay)
	}
}
