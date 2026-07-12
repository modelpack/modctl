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
	DisableProgress   bool
	DragonflyEndpoint string
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
		DisableProgress:   false,
		DragonflyEndpoint: "",
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

	// DragonflyEndpoint only can work with ExtractFromRemote scenario.
	if p.DragonflyEndpoint != "" && !p.ExtractFromRemote {
		return fmt.Errorf("dragonfly endpoint only can work with extract from remote scenario")
	}

	return nil
}

// PullHooks is the hook events during the pull operation.
//
// Note: every retry attempt re-invokes BeforePullLayer / AfterPullLayer.
type PullHooks interface {
	// BeforePullLayer will execute before pulling the layer described as desc,
	// will carry the manifest as well.
	//
	// If the hook returns skip=true, the backend will treat this layer as
	// already satisfied and will NOT actually pull/extract it. The caller is
	// responsible for ensuring the corresponding content already exists and
	// matches the descriptor's digest. AfterPullLayer will still be invoked
	// with skipped=true and a nil error.
	BeforePullLayer(desc ocispec.Descriptor, manifest ocispec.Manifest) (skip bool)

	// AfterPullLayer will execute after pulling the layer described as desc.
	// skipped indicates whether the layer was skipped by BeforePullLayer's
	// decision. err will be nil if pulled (or skipped) successfully.
	AfterPullLayer(desc ocispec.Descriptor, skipped bool, err error)
}

// emptyPullHook is the empty pull hook implementation with do nothing.
type emptyPullHook struct{}

func (emptyPullHook) BeforePullLayer(desc ocispec.Descriptor, manifest ocispec.Manifest) bool {
	return false
}
func (emptyPullHook) AfterPullLayer(desc ocispec.Descriptor, skipped bool, err error) {}
