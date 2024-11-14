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

package backend

import (
	"fmt"
	"io"
	"sync"

	humanize "github.com/dustin/go-humanize"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	mpbv8 "github.com/vbauerster/mpb/v8"
	"github.com/vbauerster/mpb/v8/decor"
)

const (
	promptCopyingBlob     = "Copying blob     "
	promptCopyingConfig   = "Copying config   "
	promptCopyingManifest = "Copying manifest "
)

// ProgressBar is a progress bar.
type ProgressBar struct {
	mu   sync.Mutex
	mpb  *mpbv8.Progress
	bars map[string]*mpbv8.Bar
}

// NewProgressBar creates a new progress bar.
func NewProgressBar() *ProgressBar {
	return &ProgressBar{
		mpb:  mpbv8.New(mpbv8.WithWidth(60)),
		bars: make(map[string]*mpbv8.Bar),
	}
}

// Add adds a new progress bar.
func (p *ProgressBar) Add(prompt string, desc ocispec.Descriptor, reader io.Reader) io.Reader {
	p.mu.Lock()
	defer p.mu.Unlock()
	// if the bar already exists, return the reader directly.
	if _, ok := p.bars[desc.Digest.String()]; ok {
		return reader
	}

	// create a new bar if it does not exist.
	bar := p.mpb.New(desc.Size,
		mpbv8.BarStyle().Rbound("|"),
		mpbv8.BarFillerOnComplete("|"),
		mpbv8.PrependDecorators(
			decor.Name(fmt.Sprintf("%s%s", prompt, desc.Digest.String())),
		),
		mpbv8.AppendDecorators(
			decor.OnComplete(decor.Counters(decor.SizeB1024(0), "% .2f / % .2f"), humanize.Bytes(uint64(desc.Size))),
			decor.OnComplete(decor.Name(" | "), " | "),
			decor.OnComplete(
				decor.AverageSpeed(decor.SizeB1024(0), "% .2f"), "done",
			),
		),
	)

	p.bars[desc.Digest.String()] = bar
	return bar.ProxyReader(reader)
}

// PrintMessage adds a new progress bar for printing the message.
// background: before the all progress bar finished, any stdout print such as fmt.Println
// will effect the rendering of progress bar, so here create a bar to print message.
// issue: https://github.com/vbauerster/mpb/issues/118
func (p *ProgressBar) PrintMessage(prompt string, desc ocispec.Descriptor, message string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	// if the bar already exists, return directly.
	if _, ok := p.bars[desc.Digest.String()]; ok {
		return
	}

	// create a new bar if it does not exist.
	bar := p.mpb.New(0,
		mpbv8.BarStyle(),
		mpbv8.BarFillerClearOnComplete(),
		mpbv8.PrependDecorators(
			decor.Name(fmt.Sprintf("%s%s ", prompt, desc.Digest.String())),
			decor.OnComplete(
				decor.Name(""), message,
			),
		),
	)

	bar.EnableTriggerComplete()
	p.bars[desc.Digest.String()] = bar
}

// Wait waits for the progress bar to finish.
func (p *ProgressBar) Wait() {
	p.mpb.Wait()
}
