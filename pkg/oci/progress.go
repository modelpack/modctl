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

package oci

import (
	"fmt"
	"io"
	"sync"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/vbauerster/mpb/v8"
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
	mpb  *mpb.Progress
	bars map[string]*mpb.Bar
}

// NewProgressBar creates a new progress bar.
func NewProgressBar() *ProgressBar {
	return &ProgressBar{
		mpb:  mpb.New(mpb.WithWidth(60)),
		bars: make(map[string]*mpb.Bar),
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
		mpb.BarStyle().Rbound("|"),
		mpb.PrependDecorators(
			decor.Name(fmt.Sprintf("%s%s", prompt, desc.Digest.String())),
		),
		mpb.AppendDecorators(
			decor.Counters(decor.SizeB1024(0), "% .2f / % .2f"),
			decor.Name(" ] "),
			decor.OnComplete(
				decor.EwmaSpeed(decor.SizeB1024(0), "% .2f", 30), "done",
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
		mpb.BarStyle(),
		mpb.BarFillerClearOnComplete(),
		mpb.PrependDecorators(
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
