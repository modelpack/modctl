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
	"fmt"
	"io"
	"os"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"
)

// Tracker aggregates IO throughput metrics from multiple concurrent
// goroutines. Create one per operation (push/pull/fetch) and pass it
// to each goroutine so they share the same atomic counters.
type Tracker struct {
	operation     string
	startTime     time.Time
	bytes         atomic.Int64 // total bytes read from source
	sourceNanos   atomic.Int64 // cumulative source Read() call durations
	transferNanos atomic.Int64 // cumulative per-goroutine wall clock
}

// NewTracker creates a new Tracker and records the start time.
// Call this before launching the errgroup.
func NewTracker(operation string) *Tracker {
	return &Tracker{
		operation: operation,
		startTime: time.Now(),
	}
}

// WrapReader wraps an io.Reader to count bytes and accumulate Read()
// call durations into the shared atomic counters.
func (t *Tracker) WrapReader(r io.Reader) io.Reader {
	return &countingReader{
		reader: r,
		bytes:  &t.bytes,
		nanos:  &t.sourceNanos,
	}
}

// TrackTransfer measures the wall-clock duration of a single transfer
// (one goroutine handling one blob) and accumulates it. Place this
// inside retry.Do so each retry attempt is measured independently.
func (t *Tracker) TrackTransfer(fn func() error) error {
	start := time.Now()
	err := fn()
	t.transferNanos.Add(int64(time.Since(start)))
	return err
}

// Summary outputs a throughput summary to both the log file (logrus)
// and the terminal (stderr). Call this after all goroutines have
// completed (after g.Wait()) — the happens-before from errgroup
// guarantees all atomic stores are visible.
func (t *Tracker) Summary() {
	wallClock := time.Since(t.startTime)
	totalBytes := t.bytes.Load()
	sourceNanos := t.sourceNanos.Load()
	transferNanos := t.transferNanos.Load()

	if totalBytes == 0 {
		return
	}

	sourceReadTime := time.Duration(sourceNanos)

	var readFraction float64
	if transferNanos > 0 {
		readFraction = float64(sourceNanos) / float64(transferNanos)
	}

	// Log structured fields to log file.
	logrus.WithFields(logrus.Fields{
		"operation":            t.operation,
		"totalBytes":           formatBytes(uint64(totalBytes)),
		"wallClock":            wallClock.Round(time.Millisecond).String(),
		"sourceReadTime":       sourceReadTime.Round(time.Millisecond).String(),
		"sourceReadThroughput": formatThroughput(totalBytes, sourceReadTime),
		"effectiveThroughput":  formatThroughput(totalBytes, wallClock),
		"readFraction":         fmt.Sprintf("%.2f", readFraction),
	}).Info("io throughput summary")

	// Print concise summary to terminal.
	fmt.Fprintf(os.Stderr, "IO summary: %s in %s, effective %s, source %s, read ratio %.2f\n",
		formatBytes(uint64(totalBytes)),
		wallClock.Round(time.Millisecond),
		formatThroughput(totalBytes, wallClock),
		formatThroughput(totalBytes, sourceReadTime),
		readFraction,
	)
}
