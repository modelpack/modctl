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

package config

import (
	"fmt"
	"io"
	"os"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

const (
	// defaultPullConcurrency is the default number of concurrent pull operations.
	defaultPullConcurrency = 5
)

type Pull struct {
	Concurrency       int
	PlainHTTP         bool
	Proxy             string
	Insecure          bool
	ExtractDir        string
	ExtractFromRemote bool
	Hooks             PullHooks
	ProgressWriter    io.Writer
}

func NewPull() *Pull {
	return &Pull{
		Concurrency:       defaultPullConcurrency,
		PlainHTTP:         false,
		Proxy:             "",
		Insecure:          false,
		ExtractDir:        "",
		ExtractFromRemote: false,
		Hooks:             &emptyPullHook{},
		ProgressWriter:    os.Stdout,
	}
}

func (p *Pull) Validate() error {
	if p.Concurrency < 1 {
		return fmt.Errorf("invalid concurrency: %d", p.Concurrency)
	}

	// Validate the ExtractDir if user specify the ExtractFromRemote to true.
	if p.ExtractFromRemote {
		if p.ExtractDir == "" {
			return fmt.Errorf("the extract dir must be specified when enabled extract from remote")
		}
	}

	return nil
}

// PullHooks is the hook events during the pull operation.
type PullHooks interface {
	// BeforePullLayer will execute before pulling the layer described as desc, will carry the manifest as well.
	BeforePullLayer(desc ocispec.Descriptor, manifest ocispec.Manifest)

	// AfterPullLayer will execute after pulling the layer described as desc, the error will be nil if pulled successfully.
	AfterPullLayer(desc ocispec.Descriptor, err error)
}

// emptyPullHook is the empty pull hook implementation with do nothing.
type emptyPullHook struct{}

func (emptyPullHook) BeforePullLayer(desc ocispec.Descriptor, manifest ocispec.Manifest) {}
func (emptyPullHook) AfterPullLayer(desc ocispec.Descriptor, err error)                  {}
