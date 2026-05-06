/*
 *     Copyright 2024 The ModelPack Authors
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
	"sync/atomic"
	"testing"
	"time"
)

// --- ComputePerAttemptTimeout ---

func TestComputePerAttemptTimeout(t *testing.T) {
	const (
		oneMB  = int64(1) << 20
		oneGB  = int64(1) << 30
		tenGB  = int64(10) << 30
		hundGB = int64(100) << 30
	)
	tests := []struct {
		name string
		size int64
		want time.Duration
	}{
		{"zero size clamps to floor", 0, minPerAttemptTimeout},
		{"100 MB clamps to floor", 100 * oneMB, minPerAttemptTimeout},
		{"1 GB clamps to floor", oneGB, minPerAttemptTimeout},
		// 5 GB / 10 MB/s * 2 = 1024s ≈ 17 min, above the 5min floor
		{
			"5 GB scales above floor",
			5 * oneGB,
			time.Duration(5*oneGB/minThroughput*safetyFactor) * time.Second,
		},
		// 10 GB / 10 MB/s * 2 = 2048s ≈ 34 min
		{
			"10 GB scales linearly",
			tenGB,
			time.Duration(tenGB/minThroughput*safetyFactor) * time.Second,
		},
		// 100 GB / 10 MB/s * 2 = 20480s ≈ 5.7h, still under the 8h ceiling
		{
			"100 GB still under ceiling",
			hundGB,
			time.Duration(hundGB/minThroughput*safetyFactor) * time.Second,
		},
		// 200 GB hits ceiling
		{"200 GB clamps to ceiling", 2 * hundGB, maxPerAttemptTimeout},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ComputePerAttemptTimeout(tt.size)
			if got != tt.want {
				t.Errorf("ComputePerAttemptTimeout(%d) = %v, want %v", tt.size, got, tt.want)
			}
		})
	}
}

// --- IsRetryable ---

func TestIsRetryable(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"context.Canceled", context.Canceled, false},
		{"context.DeadlineExceeded", context.DeadlineExceeded, false},
		{"5xx server error", errors.New("response status code 500"), true},
		{
			"503 service unavailable",
			errors.New("response status code 503: Service Unavailable"),
			true,
		},
		{"408 request timeout", errors.New("response status code 408"), true},
		{"429 too many requests", errors.New("response status code 429"), true},
		{"401 unauthorized", errors.New("response status code 401"), false},
		{"403 forbidden", errors.New("response status code 403"), false},
		{"404 not found", errors.New("response status code 404"), false},
		{"i/o timeout", errors.New("dial tcp: i/o timeout"), true},
		{"connection reset", errors.New("read tcp: connection reset by peer"), true},
		{"connection refused", errors.New("dial tcp: connection refused"), true},
		{"broken pipe", errors.New("write tcp: broken pipe"), true},
		{"EOF", errors.New("unexpected EOF"), true},
		{"permission denied", errors.New("open /etc/foo: permission denied"), false},
		{"no space left", errors.New("write /tmp/x: no space left on device"), false},
		{"file exists", errors.New("link /a /b: file exists"), false},
		{"no such file", errors.New("open /no/such: no such file or directory"), false},
		{"unknown defaults retryable", errors.New("some weird error"), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsRetryable(tt.err); got != tt.want {
				t.Errorf("IsRetryable(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

// --- ShortReason ---

func TestShortReason(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{"nil", nil, ""},
		{"5xx", errors.New("response status code 503"), "HTTP 503"},
		{"i/o timeout", errors.New("dial tcp: i/o timeout"), "i/o timeout"},
		{"conn reset", errors.New("read tcp: connection reset by peer"), "conn reset"},
		{"conn refused", errors.New("dial: connection refused"), "conn refused"},
		{"broken pipe", errors.New("write: broken pipe"), "broken pipe"},
		{"EOF", errors.New("unexpected EOF"), "EOF"},
		{"DeadlineExceeded", context.DeadlineExceeded, "attempt timeout"},
		{"unknown", errors.New("totally unrelated"), "unknown error"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ShortReason(tt.err); got != tt.want {
				t.Errorf("ShortReason(%v) = %q, want %q", tt.err, got, tt.want)
			}
		})
	}
}

// --- computeBackoff ---

func TestComputeBackoff(t *testing.T) {
	const initial = 1 * time.Second
	const cap_ = 10 * time.Second
	tests := []struct {
		attempt uint
		want    time.Duration
	}{
		{1, 1 * time.Second}, // first sleep
		{2, 2 * time.Second}, // doubled
		{3, 4 * time.Second},
		{4, 8 * time.Second},
		{5, 10 * time.Second}, // capped
		{20, 10 * time.Second},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("attempt=%d", tt.attempt), func(t *testing.T) {
			got := computeBackoff(tt.attempt, initial, cap_)
			if got != tt.want {
				t.Errorf("computeBackoff(%d) = %v, want %v", tt.attempt, got, tt.want)
			}
		})
	}
}

// --- Do: defaults & success path ---

func TestDo_SuccessFirstAttempt(t *testing.T) {
	calls := 0
	err := Do(context.Background(), func(ctx context.Context) error {
		calls++
		return nil
	}, DoOpts{FileName: "ok"})
	if err != nil {
		t.Fatalf("Do returned %v", err)
	}
	if calls != 1 {
		t.Errorf("calls = %d, want 1", calls)
	}
}

// --- Do: NoRetry ---

func TestDo_NoRetry(t *testing.T) {
	calls := 0
	transient := errors.New("response status code 503")
	err := Do(context.Background(), func(ctx context.Context) error {
		calls++
		return transient
	}, DoOpts{
		FileName: "noretry",
		Config:   &Config{NoRetry: true},
	})
	if !errors.Is(err, transient) {
		t.Errorf("err = %v, want %v", err, transient)
	}
	if calls != 1 {
		t.Errorf("calls = %d, want 1 (NoRetry)", calls)
	}
}

// --- Do: MaxAttempts caps the number of tries ---

func TestDo_MaxAttempts(t *testing.T) {
	var calls int32
	err := Do(context.Background(), func(ctx context.Context) error {
		atomic.AddInt32(&calls, 1)
		return errors.New("response status code 503")
	}, DoOpts{
		FileName: "always-fails",
		Config: &Config{
			MaxAttempts:  3,
			InitialDelay: 1 * time.Millisecond,
			MaxBackoff:   1 * time.Millisecond,
			MaxJitter:    -1,
		},
	})
	if err == nil {
		t.Fatal("Do returned nil, want error after attempts exhausted")
	}
	if got := atomic.LoadInt32(&calls); got != 3 {
		t.Errorf("calls = %d, want 3", got)
	}
}

// --- Do: non-retryable error stops immediately ---

func TestDo_NonRetryableStopsImmediately(t *testing.T) {
	var calls int32
	permErr := errors.New("permission denied")
	err := Do(context.Background(), func(ctx context.Context) error {
		atomic.AddInt32(&calls, 1)
		return permErr
	}, DoOpts{
		FileName: "perm",
		Config: &Config{
			MaxAttempts:  5,
			InitialDelay: 1 * time.Millisecond,
			MaxJitter:    -1,
		},
	})
	if err == nil {
		t.Fatal("Do returned nil, want non-retryable error")
	}
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Errorf("calls = %d, want 1 (non-retryable)", got)
	}
}

// --- Do: per-attempt timeout fires & retries continue ---

func TestDo_PerAttemptTimeoutTriggersRetry(t *testing.T) {
	var calls int32
	const succeededOn int32 = 3
	err := Do(context.Background(), func(ctx context.Context) error {
		n := atomic.AddInt32(&calls, 1)
		if n < succeededOn {
			<-ctx.Done()
			return ctx.Err()
		}
		return nil
	}, DoOpts{
		FileName: "slow",
		Config: &Config{
			MaxAttempts:       5,
			PerAttemptTimeout: 30 * time.Millisecond,
			InitialDelay:      1 * time.Millisecond,
			MaxBackoff:        1 * time.Millisecond,
			MaxJitter:         -1,
		},
	})
	if err != nil {
		t.Fatalf("Do returned %v, want success after retries", err)
	}
	if got := atomic.LoadInt32(&calls); got != succeededOn {
		t.Errorf("calls = %d, want %d", got, succeededOn)
	}
}

// --- Do: per-attempt timeout exhausts after MaxAttempts ---

func TestDo_PerAttemptTimeoutExhausts(t *testing.T) {
	var calls int32
	err := Do(context.Background(), func(ctx context.Context) error {
		atomic.AddInt32(&calls, 1)
		<-ctx.Done()
		return ctx.Err()
	}, DoOpts{
		FileName: "always-times-out",
		Config: &Config{
			MaxAttempts:       3,
			PerAttemptTimeout: 20 * time.Millisecond,
			InitialDelay:      1 * time.Millisecond,
			MaxBackoff:        1 * time.Millisecond,
			MaxJitter:         -1,
		},
	})
	if err == nil {
		t.Fatal("Do returned nil, want error after exhausting attempts")
	}
	if got := atomic.LoadInt32(&calls); got != 3 {
		t.Errorf("calls = %d, want 3", got)
	}
}

// --- Do: parent context cancellation aborts retries ---

func TestDo_ParentContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var calls int32
	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	err := Do(ctx, func(ctx context.Context) error {
		atomic.AddInt32(&calls, 1)
		return errors.New("response status code 503")
	}, DoOpts{
		FileName: "user-cancels",
		Config: &Config{
			MaxAttempts:  100,
			InitialDelay: 5 * time.Millisecond,
			MaxBackoff:   5 * time.Millisecond,
			MaxJitter:    -1,
		},
	})
	if err == nil {
		t.Fatal("Do returned nil, want context cancellation error")
	}
	if got := atomic.LoadInt32(&calls); got > 50 {
		t.Errorf(
			"calls = %d, want significantly fewer than MaxAttempts (parent ctx cancelled)",
			got,
		)
	}
}

// --- Do: OnRetry callback invoked with 1-based attempt ---

func TestDo_OnRetryCallback(t *testing.T) {
	var attempts []uint
	var reasons []string
	var calls int32
	err := Do(context.Background(), func(ctx context.Context) error {
		n := atomic.AddInt32(&calls, 1)
		if n < 3 {
			return errors.New("response status code 500")
		}
		return nil
	}, DoOpts{
		FileName: "cb",
		Config: &Config{
			MaxAttempts:  5,
			InitialDelay: 1 * time.Millisecond,
			MaxBackoff:   1 * time.Millisecond,
			MaxJitter:    -1,
		},
		OnRetry: func(attempt uint, reason string, backoff time.Duration) {
			attempts = append(attempts, attempt)
			reasons = append(reasons, reason)
		},
	})
	if err != nil {
		t.Fatalf("Do returned %v", err)
	}
	want := []uint{1, 2}
	if fmt.Sprintf("%v", attempts) != fmt.Sprintf("%v", want) {
		t.Errorf("attempts = %v, want %v", attempts, want)
	}
	for _, r := range reasons {
		if r != "HTTP 500" {
			t.Errorf("reason = %q, want \"HTTP 500\"", r)
		}
	}
}

// --- Do: retry budget invariance — same total time regardless of file size ---
//
// The whole point of the redesign: with PerAttemptTimeout disabled and a
// fixed backoff schedule, total wall-clock for retries does not depend on
// FileSize. (The old API's MaxRetryTime scaled with size and changed this.)
func TestDo_RetryBudgetIndependentOfFileSize(t *testing.T) {
	measure := func(fileSize int64) time.Duration {
		var calls int32
		start := time.Now()
		_ = Do(context.Background(), func(ctx context.Context) error {
			atomic.AddInt32(&calls, 1)
			return errors.New("response status code 503")
		}, DoOpts{
			FileName: "size-invariance",
			FileSize: fileSize,
			Config: &Config{
				MaxAttempts:       4,
				PerAttemptTimeout: -1,
				InitialDelay:      10 * time.Millisecond,
				MaxBackoff:        10 * time.Millisecond,
				MaxJitter:         -1,
			},
		})
		return time.Since(start)
	}

	small := measure(1 << 20)       // 1 MB
	huge := measure(int64(1) << 40) // 1 TB
	diff := small - huge
	if diff < 0 {
		diff = -diff
	}
	if diff > 100*time.Millisecond {
		t.Errorf(
			"retry budget varied with file size: small=%v huge=%v (diff=%v)",
			small,
			huge,
			diff,
		)
	}
}

// --- Do: PerAttemptTimeout < 0 disables the deadline ---

func TestDo_PerAttemptTimeoutDisabled(t *testing.T) {
	var seenDeadline bool
	err := Do(context.Background(), func(ctx context.Context) error {
		_, ok := ctx.Deadline()
		seenDeadline = ok
		return nil
	}, DoOpts{
		FileName: "no-deadline",
		FileSize: 1 << 30,
		Config: &Config{
			PerAttemptTimeout: -1,
		},
	})
	if err != nil {
		t.Fatalf("Do returned %v", err)
	}
	if seenDeadline {
		t.Error("attempt context had a deadline; expected none when PerAttemptTimeout < 0")
	}
}

// --- Do: PerAttemptTimeout = 0 derives from file size ---

func TestDo_PerAttemptTimeoutDerivedFromSize(t *testing.T) {
	var seenTimeout time.Duration
	const size = int64(50) << 30 // 50 GB → > floor, < ceiling
	err := Do(context.Background(), func(ctx context.Context) error {
		dl, ok := ctx.Deadline()
		if !ok {
			t.Fatal("expected deadline")
		}
		seenTimeout = time.Until(dl)
		return nil
	}, DoOpts{
		FileName: "derived",
		FileSize: size,
	})
	if err != nil {
		t.Fatalf("Do returned %v", err)
	}
	want := ComputePerAttemptTimeout(size)
	if seenTimeout > want || seenTimeout < want-time.Second {
		t.Errorf("derived timeout = %v, want ~%v", seenTimeout, want)
	}
}

// --- humanizeBytes ---

func TestHumanizeBytes(t *testing.T) {
	tests := []struct {
		size int64
		want string
	}{
		{0, "0 B"},
		{500, "500 B"},
		{2048, "2.0 KB"},
		{int64(5) << 20, "5.0 MB"},
		{int64(7) << 30, "7.0 GB"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := humanizeBytes(tt.size); got != tt.want {
				t.Errorf("humanizeBytes(%d) = %q, want %q", tt.size, got, tt.want)
			}
		})
	}
}
