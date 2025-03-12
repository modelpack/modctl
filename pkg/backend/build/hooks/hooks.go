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

package hooks

import (
	"io"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

// OnStartFunc defines the signature for the OnStart hook function.
type OnStartFunc func(name string, size int64, reader io.Reader) io.Reader

// OnErrorFunc defines the signature for the OnError hook function.
type OnErrorFunc func(name string, err error)

// OnCompleteFunc defines the signature for the OnComplete hook function.
type OnCompleteFunc func(name string, desc ocispec.Descriptor)

// Hooks is a struct that contains hook functions.
type Hooks struct {
	// OnStart is called when the build process starts.
	OnStart OnStartFunc

	// OnError is called when the build process encounters an error.
	OnError OnErrorFunc

	// OnComplete is called when the build process completes successfully.
	OnComplete OnCompleteFunc
}

// NewHooks creates a new Hooks instance with optional function parameters.
func NewHooks(opts ...Option) Hooks {
	h := Hooks{
		OnStart: func(name string, size int64, reader io.Reader) io.Reader {
			return reader
		},
		OnError:    func(name string, err error) {},
		OnComplete: func(name string, desc ocispec.Descriptor) {},
	}

	for _, opt := range opts {
		opt(&h)
	}

	return h
}

// Option is a function type that can be used to customize a Hooks instance.
type Option func(*Hooks)

// WithOnStart returns an Option that sets the OnStart hook.
func WithOnStart(f OnStartFunc) Option {
	return func(h *Hooks) {
		if f != nil {
			h.OnStart = f
		}
	}
}

// WithOnError returns an Option that sets the OnError hook.
func WithOnError(f OnErrorFunc) Option {
	return func(h *Hooks) {
		if f != nil {
			h.OnError = f
		}
	}
}

// WithOnComplete returns an Option that sets the OnComplete hook.
func WithOnComplete(f OnCompleteFunc) Option {
	return func(h *Hooks) {
		if f != nil {
			h.OnComplete = f
		}
	}
}
