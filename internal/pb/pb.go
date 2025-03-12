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
	"sync"

	humanize "github.com/dustin/go-humanize"
	mpbv8 "github.com/vbauerster/mpb/v8"
	"github.com/vbauerster/mpb/v8/decor"
)

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
	size int64
	msg  string
}

// NewProgressBar creates a new progress bar.
func NewProgressBar() *ProgressBar {
	return &ProgressBar{
		mpb:  mpbv8.New(mpbv8.WithWidth(60)),
		bars: make(map[string]*progressBar),
	}
}

// Add adds a new progress bar.
func (p *ProgressBar) Add(prompt, name string, size int64, reader io.Reader) io.Reader {
	p.mu.RLock()
	oldBar := p.bars[name]
	p.mu.RUnlock()

	if oldBar != nil {
		return reader
	}

	// Create a new bar if it does not exist.
	bar := p.mpb.New(size,
		mpbv8.BarStyle(),
		mpbv8.BarFillerOnComplete("|"),
		mpbv8.PrependDecorators(
			decor.Any(func(s decor.Statistics) string {
				p.mu.RLock()
				defer p.mu.RUnlock()

				bar, ok := p.bars[name]
				if ok && bar.msg != "" {
					return bar.msg
				}

				return fmt.Sprintf("%s %s", prompt, name)
			}, decor.WCSyncSpaceR),
		),
		mpbv8.AppendDecorators(
			decor.OnComplete(decor.Counters(decor.SizeB1024(0), "% .2f / % .2f"), humanize.Bytes(uint64(size))),
			decor.OnComplete(decor.Name(" | ", decor.WCSyncWidthR), " | "),
			decor.OnComplete(
				decor.AverageSpeed(decor.SizeB1024(0), "% .2f", decor.WCSyncWidthR), "done",
			),
		),
	)

	p.mu.Lock()
	p.bars[name] = &progressBar{Bar: bar, size: size}
	p.mu.Unlock()

	return bar.ProxyReader(reader)
}

// Complete completes the progress bar.
func (p *ProgressBar) Complete(name string, msg string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	bar, ok := p.bars[name]
	if ok {
		bar.msg = msg
		bar.Bar.SetCurrent(bar.size)
	}
}

// Start starts the progress bar.
func (p *ProgressBar) Start() {}

// Stop waits for the progress bar to finish.
func (p *ProgressBar) Stop() {
	p.mpb.Shutdown()
}
