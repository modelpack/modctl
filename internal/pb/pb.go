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

package pb

import (
	"fmt"
	"io"
	"os"
	"sync"
	"sync/atomic"
	"time"

	humanize "github.com/dustin/go-humanize"
	"github.com/sirupsen/logrus"
	mpbv8 "github.com/vbauerster/mpb/v8"
	"github.com/vbauerster/mpb/v8/decor"
)

var (
	// disableProgress is the flag to disable progress bar. Read from the
	// progress-bar hot path by concurrent pull/push workers while a single
	// goroutine may flip it via SetDisableProgress, so the access is guarded
	// by sync/atomic to stay race-free under `go test -race`.
	disableProgress atomic.Bool
)

// SetDisableProgress disables the progress bar.
func SetDisableProgress(disable bool) {
	disableProgress.Store(disable)
}

// NormalizePrompt normalizes the prompt string.
func NormalizePrompt(prompt string) string {
	return fmt.Sprintf("%s =>", prompt)
}

// ProgressBar is a progress bar.
type ProgressBar struct {
	mu   sync.RWMutex
	mpb  *mpbv8.Progress
	bars map[string]*progressBar
}

type progressBar struct {
	*mpbv8.Bar
	size      int64
	msg       atomic.Value // stores string; accessed by mpb render goroutine
	startTime time.Time
	// indeterminate marks a spinner bar created by Spinner. It has no byte
	// counter and no completion trigger until Complete, Reset or Abort.
	indeterminate bool
}

// NewProgressBar creates a new progress bar.
func NewProgressBar(writers ...io.Writer) *ProgressBar {
	opts := []mpbv8.ContainerOption{
		mpbv8.PopCompletedMode(),
		mpbv8.WithAutoRefresh(),
		mpbv8.WithWidth(60),
		mpbv8.WithRefreshRate(300 * time.Millisecond),
	}

	// If no writer specified, use stdout.
	if len(writers) == 0 {
		opts = append(opts, mpbv8.WithOutput(os.Stdout))
	} else if len(writers) == 1 {
		opts = append(opts, mpbv8.WithOutput(writers[0]))
	} else {
		opts = append(opts, mpbv8.WithOutput(io.MultiWriter(writers...)))
	}

	return &ProgressBar{
		mpb:  mpbv8.New(opts...),
		bars: make(map[string]*progressBar),
	}
}

// Add adds a new progress bar.
func (p *ProgressBar) Add(prompt, name string, size int64, reader io.Reader) io.Reader {
	// Return the reader directly if progress is disabled.
	if disableProgress.Load() {
		return reader
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	// If a bar with the same name exists, abort and drop it before creating
	// the replacement. Note: mpb ignores Abort on already-completed bars, but
	// with PopCompletedMode those have been popped out of the rendering area,
	// so the replacement still displays correctly and the map entry is
	// updated either way.
	if oldBar := p.bars[name]; oldBar != nil {
		oldBar.Abort(true)
	}

	newBar := &progressBar{
		size:      size,
		startTime: time.Now(),
	}
	newBar.msg.Store(fmt.Sprintf("%s %s", prompt, name))

	newBar.Bar = p.mpb.New(size,
		mpbv8.BarStyle(),
		mpbv8.BarFillerOnComplete("|"),
		mpbv8.PrependDecorators(
			decor.Any(func(s decor.Statistics) string {
				return newBar.msg.Load().(string)
			}, decor.WCSyncSpaceR),
		),
		mpbv8.AppendDecorators(
			decor.OnComplete(decor.Counters(decor.SizeB1000(0), "% .2f / % .2f"), humanize.Bytes(uint64(size))),
			decor.OnComplete(decor.Name(" | ", decor.WCSyncWidthR), " | "),
			decor.OnCompleteMeta(
				decor.AverageSpeed(decor.SizeB1000(0), "% .2f", decor.WCSyncWidthR),
				func(_ string) string {
					duration := time.Since(newBar.startTime).Seconds()
					return fmt.Sprintf("done(%.1fs)", duration)
				},
			),
		),
	)

	p.bars[name] = newBar

	if reader != nil {
		return newBar.ProxyReader(reader)
	}

	return reader
}

// Spinner adds an indeterminate bar for a phase that transfers no bytes, such
// as a remote existence check. It renders a spinner, the prompt, the total
// size, and the elapsed time. It has no byte counter and no transfer rate, so
// a slow phase does not show a misleading "0 B / N B" and "0 B/s". Call Reset
// (or Add) with the same name to replace it with a transfer bar once bytes
// start flowing, or Complete / Abort to finish it.
func (p *ProgressBar) Spinner(prompt, name string, size int64) {
	if disableProgress.Load() {
		return
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	// Same replacement semantics as Add.
	if oldBar := p.bars[name]; oldBar != nil {
		oldBar.Abort(true)
	}

	newBar := &progressBar{
		size:          size,
		startTime:     time.Now(),
		indeterminate: true,
	}
	newBar.msg.Store(fmt.Sprintf("%s %s", prompt, name))

	// A zero total disables mpb's completion trigger, so the bar keeps
	// spinning until it is replaced, completed, or aborted.
	newBar.Bar = p.mpb.New(0,
		mpbv8.SpinnerStyle(),
		mpbv8.PrependDecorators(
			decor.Any(func(_ decor.Statistics) string {
				return newBar.msg.Load().(string)
			}, decor.WCSyncSpaceR),
		),
		mpbv8.AppendDecorators(
			decor.Name(humanize.Bytes(uint64(size)), decor.WCSyncWidthR),
			decor.Name(" | ", decor.WCSyncWidthR),
			decor.Elapsed(decor.ET_STYLE_GO, decor.WCSyncWidthR),
		),
	)

	p.bars[name] = newBar
}

// Get returns the progress bar.
func (p *ProgressBar) Get(name string) *progressBar {
	p.mu.RLock()
	bar := p.bars[name]
	p.mu.RUnlock()

	return bar
}

// Complete completes the progress bar.
func (p *ProgressBar) Complete(name string, msg string) {
	p.mu.RLock()
	bar, ok := p.bars[name]
	p.mu.RUnlock()

	if ok {
		bar.msg.Store(msg)
		if bar.indeterminate {
			// A spinner has no total; give it one so mpb marks it complete.
			bar.SetTotal(bar.size, true)
			return
		}
		bar.Bar.SetCurrent(bar.size)
	}
}

// Abort aborts the progress bar.
func (p *ProgressBar) Abort(name string, err error) {
	p.mu.RLock()
	bar, ok := p.bars[name]
	p.mu.RUnlock()

	if ok {
		logrus.Errorf("progress: aborting bar %s: %v", name, err)
		bar.Abort(true)
	}
}

// Reset resets an existing progress bar for a new phase.
// Aborts the old bar (if any, see Add for completed-bar semantics) and
// creates a new one with updated prompt, reset progress, and fresh speed
// counter. Creates the bar if it does not exist yet. Parameter order
// matches Add.
func (p *ProgressBar) Reset(prompt, name string, size int64, reader io.Reader) io.Reader {
	return p.Add(prompt, name, size, reader)
}

// Start starts the progress bar.
func (p *ProgressBar) Start() {}

// Stop waits for the progress bar to finish.
func (p *ProgressBar) Stop() {
	p.mpb.Shutdown()
}
