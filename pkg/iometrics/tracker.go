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

	"github.com/dustin/go-humanize"
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

// sourceLabel and sinkLabel return human-readable labels for the source
// and sink sides based on the operation type.
func (t *Tracker) sourceLabel() string {
	if t.operation == "push" {
		return "disk read"
	}
	return "network read"
}

func (t *Tracker) sinkLabel() string {
	if t.operation == "push" {
		return "network write"
	}
	return "disk write"
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

	sourceDuration := time.Duration(sourceNanos)
	sinkNanos := transferNanos - sourceNanos
	sinkDuration := time.Duration(max(sinkNanos, 0))

	sourceThroughput := formatThroughput(totalBytes, sourceDuration)
	sinkThroughput := formatThroughput(totalBytes, sinkDuration)

	// Identify the bottleneck by comparing cumulative durations.
	bottleneck := t.sinkLabel()
	if sourceNanos > sinkNanos {
		bottleneck = t.sourceLabel()
	}

	// Log structured fields to log file.
	logrus.WithFields(logrus.Fields{
		"operation":            t.operation,
		"totalBytes":           humanize.IBytes(uint64(totalBytes)),
		"wallClock":            wallClock.Round(time.Millisecond).String(),
		"effectiveThroughput":  formatThroughput(totalBytes, wallClock),
		t.sourceLabel():        sourceThroughput,
		t.sinkLabel():          sinkThroughput,
		"bottleneck":           bottleneck,
	}).Info("io throughput summary")

	// Print concise summary to terminal.
	srcArrow := ""
	snkArrow := ""
	if bottleneck == t.sourceLabel() {
		srcArrow = " ← bottleneck"
	} else {
		snkArrow = " ← bottleneck"
	}
	fmt.Fprintf(os.Stderr, "IO summary: %s in %s, %s | %s: %s%s | %s: %s%s\n",
		humanize.IBytes(uint64(totalBytes)),
		wallClock.Round(time.Millisecond),
		formatThroughput(totalBytes, wallClock),
		t.sourceLabel(), sourceThroughput, srcArrow,
		t.sinkLabel(), sinkThroughput, snkArrow,
	)
}
