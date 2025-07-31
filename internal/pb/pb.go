/*
 *     Copyright 2025 The CNAI Authors
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
	"time"

	humanize "github.com/dustin/go-humanize"
	"github.com/sirupsen/logrus"
	mpbv8 "github.com/vbauerster/mpb/v8"
	"github.com/vbauerster/mpb/v8/decor"
)

var (
	// disableProgress is the flag to disable progress bar.
	disableProgress bool
)

// SetDisableProgress disables the progress bar.
func SetDisableProgress(disable bool) {
	disableProgress = disable
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
	msg       string
	startTime time.Time
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
	if disableProgress {
		return reader
	}

	p.mu.RLock()
	oldBar := p.bars[name]
	p.mu.RUnlock()

	// If the bar exists, drop and remove it.
	if oldBar != nil {
		oldBar.Abort(true)
	}

	newBar := &progressBar{
		size:      size,
		msg:       fmt.Sprintf("%s %s", prompt, name),
		startTime: time.Now(),
	}
	// Create a new bar if it does not exist.
	newBar.Bar = p.mpb.New(size,
		mpbv8.BarStyle(),
		mpbv8.BarFillerOnComplete("|"),
		mpbv8.PrependDecorators(
			decor.Any(func(s decor.Statistics) string {
				return newBar.msg
			}, decor.WCSyncSpaceR),
		),
		mpbv8.AppendDecorators(
			decor.OnComplete(decor.Counters(decor.SizeB1024(0), "% .2f / % .2f"), humanize.Bytes(uint64(size))),
			decor.OnComplete(decor.Name(" | ", decor.WCSyncWidthR), " | "),
			decor.OnCompleteMeta(
				decor.AverageSpeed(decor.SizeB1024(0), "% .2f", decor.WCSyncWidthR),
				func(_ string) string {
					duration := time.Since(newBar.startTime).Seconds()
					return fmt.Sprintf("done(%.1fs)", duration)
				},
			),
		),
	)

	p.mu.Lock()
	p.bars[name] = newBar
	p.mu.Unlock()

	if reader != nil {
		return newBar.ProxyReader(reader)
	}

	return reader
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
		bar.msg = msg
		bar.Bar.SetCurrent(bar.size)
	}
}

// Abort aborts the progress bar.
func (p *ProgressBar) Abort(name string, err error) {
	p.mu.RLock()
	bar, ok := p.bars[name]
	p.mu.RUnlock()

	if ok {
		logrus.Errorf("abort the progress bar[%s] as error occurred: %v", name, err)
		bar.Abort(true)
	}
}

// Start starts the progress bar.
func (p *ProgressBar) Start() {}

// Stop waits for the progress bar to finish.
func (p *ProgressBar) Stop() {
	p.mpb.Shutdown()
}
