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

package iometrics

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

func TestCountingReaderDataIntegrity(t *testing.T) {
	data := []byte("hello world, this is a test of the counting reader")
	tracker := NewTracker("test")
	wrapped := tracker.WrapReader(bytes.NewReader(data))

	got, err := io.ReadAll(wrapped)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !bytes.Equal(got, data) {
		t.Errorf("data mismatch: got %q, want %q", got, data)
	}

	if tracker.bytes.Load() != int64(len(data)) {
		t.Errorf("bytes = %d, want %d", tracker.bytes.Load(), len(data))
	}

	if tracker.sourceNanos.Load() <= 0 {
		t.Error("sourceNanos should be > 0 after reads")
	}
}

// slowReader simulates a slow source by sleeping on each Read call.
type slowReader struct {
	data  []byte
	pos   int
	delay time.Duration
}

func (r *slowReader) Read(p []byte) (int, error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	time.Sleep(r.delay)
	n := copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}

func TestCountingReaderTimingAccumulation(t *testing.T) {
	tracker := NewTracker("test")
	sr := &slowReader{data: make([]byte, 128), delay: 10 * time.Millisecond}
	wrapped := tracker.WrapReader(sr)

	_, err := io.ReadAll(wrapped)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sourceTime := time.Duration(tracker.sourceNanos.Load())
	// At least one Read call with 10ms delay.
	if sourceTime < 10*time.Millisecond {
		t.Errorf("sourceNanos = %v, expected >= 10ms", sourceTime)
	}
}

func TestConcurrentAggregation(t *testing.T) {
	tracker := NewTracker("test")
	numGoroutines := 10
	dataPerGoroutine := 1024

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for range numGoroutines {
		go func() {
			defer wg.Done()
			data := make([]byte, dataPerGoroutine)
			wrapped := tracker.WrapReader(bytes.NewReader(data))
			io.ReadAll(wrapped)
		}()
	}

	wg.Wait()

	expectedBytes := int64(numGoroutines * dataPerGoroutine)
	if tracker.bytes.Load() != expectedBytes {
		t.Errorf("bytes = %d, want %d", tracker.bytes.Load(), expectedBytes)
	}
}

func TestTrackTransfer(t *testing.T) {
	tracker := NewTracker("test")

	err := tracker.TrackTransfer(func() error {
		time.Sleep(20 * time.Millisecond)
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	transferTime := time.Duration(tracker.transferNanos.Load())
	if transferTime < 20*time.Millisecond {
		t.Errorf("transferNanos = %v, expected >= 20ms", transferTime)
	}
}

func TestTrackTransferPropagatesError(t *testing.T) {
	tracker := NewTracker("test")
	expected := errors.New("transfer failed")

	err := tracker.TrackTransfer(func() error {
		return expected
	})

	if !errors.Is(err, expected) {
		t.Errorf("err = %v, want %v", err, expected)
	}

	// Duration should still be recorded even on error.
	if tracker.transferNanos.Load() <= 0 {
		t.Error("transferNanos should be > 0 even on error")
	}
}

func TestTrackTransferConcurrent(t *testing.T) {
	tracker := NewTracker("test")
	numGoroutines := 10

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for range numGoroutines {
		go func() {
			defer wg.Done()
			tracker.TrackTransfer(func() error {
				time.Sleep(10 * time.Millisecond)
				return nil
			})
		}()
	}

	wg.Wait()

	// Each goroutine sleeps 10ms. Total cumulative should be >= 10ms * numGoroutines.
	transferTime := time.Duration(tracker.transferNanos.Load())
	minExpected := time.Duration(numGoroutines) * 10 * time.Millisecond
	if transferTime < minExpected {
		t.Errorf("transferNanos = %v, expected >= %v", transferTime, minExpected)
	}
}

func TestSummaryZeroBytes(t *testing.T) {
	var buf bytes.Buffer
	logrus.SetOutput(&buf)
	defer logrus.SetOutput(nil)

	tracker := NewTracker("test")
	// Should not panic or produce output when no bytes were transferred.
	tracker.Summary()

	if buf.Len() > 0 {
		t.Errorf("expected no log output for zero bytes, got: %s", buf.String())
	}
}

func TestSummaryOutput(t *testing.T) {
	var buf bytes.Buffer
	logrus.SetOutput(&buf)
	logrus.SetLevel(logrus.InfoLevel)
	logrus.SetFormatter(&logrus.TextFormatter{DisableTimestamp: true})
	defer logrus.SetOutput(nil)

	tracker := NewTracker("push")

	// Simulate a transfer: read some bytes with source tracking.
	data := make([]byte, 1024*1024) // 1MB
	wrapped := tracker.WrapReader(bytes.NewReader(data))
	tracker.TrackTransfer(func() error {
		_, err := io.ReadAll(wrapped)
		return err
	})

	tracker.Summary()

	output := buf.String()
	for _, expected := range []string{
		"io throughput summary",
		"operation=push",
		"totalBytes=",
		"readFraction=",
	} {
		if !strings.Contains(output, expected) {
			t.Errorf("log output missing %q, got:\n%s", expected, output)
		}
	}
}

func TestReadFractionAccuracy(t *testing.T) {
	tracker := NewTracker("test")

	// Simulate: source read takes 80ms, total transfer takes 100ms.
	// readFraction should be ~0.8.
	sr := &slowReader{data: make([]byte, 64), delay: 80 * time.Millisecond}
	wrapped := tracker.WrapReader(sr)
	tracker.TrackTransfer(func() error {
		_, err := io.ReadAll(wrapped)
		if err != nil {
			return err
		}
		// Simulate 20ms of sink write time.
		time.Sleep(20 * time.Millisecond)
		return nil
	})

	sourceNanos := tracker.sourceNanos.Load()
	transferNanos := tracker.transferNanos.Load()

	if transferNanos == 0 {
		t.Fatal("transferNanos should be > 0")
	}

	readFraction := float64(sourceNanos) / float64(transferNanos)
	// Expect readFraction to be roughly 0.8 (with tolerance for scheduling jitter).
	if readFraction < 0.5 || readFraction > 0.95 {
		t.Errorf("readFraction = %.2f, expected ~0.8 (sourceNanos=%d, transferNanos=%d)",
			readFraction, sourceNanos, transferNanos)
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		input    uint64
		expected string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1024, "1.00 KB"},
		{1048576, "1.00 MB"},
		{1073741824, "1.00 GB"},
		{1099511627776, "1.00 TB"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := formatBytes(tt.input); got != tt.expected {
				t.Errorf("formatBytes(%d) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestFormatThroughput(t *testing.T) {
	// 1MB in 1 second = 1.00 MB/s
	result := formatThroughput(1024*1024, time.Second)
	if result != "1.00 MB/s" {
		t.Errorf("formatThroughput(1MB, 1s) = %q, want %q", result, "1.00 MB/s")
	}

	// Zero duration returns N/A.
	if got := formatThroughput(1024, 0); got != "N/A" {
		t.Errorf("formatThroughput(1024, 0) = %q, want %q", got, "N/A")
	}

	// Zero bytes returns N/A.
	if got := formatThroughput(0, time.Second); got != "N/A" {
		t.Errorf("formatThroughput(0, 1s) = %q, want %q", got, "N/A")
	}
}
