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

type Option func(*Options)

type Options struct {
	concurrency int
	plainHTTP   bool
	proxy       string
	insecure    bool
	output      string
}

// WithConcurrency sets the concurrency option.
func WithConcurrency(concurrency int) Option {
	return func(opts *Options) {
		opts.concurrency = concurrency
	}
}

// WithPlainHTTP sets the plain HTTP option.
func WithPlainHTTP(plainHTTP bool) Option {
	return func(opts *Options) {
		opts.plainHTTP = plainHTTP
	}
}

// WithProxy sets the proxy option.
func WithProxy(proxy string) Option {
	return func(opts *Options) {
		opts.proxy = proxy
	}
}

// WithInsecure sets the insecure option.
func WithInsecure(insecure bool) Option {
	return func(opts *Options) {
		opts.insecure = insecure
	}
}

// WithOutput sets the output option.
func WithOutput(output string) Option {
	return func(opts *Options) {
		opts.output = output
	}
}
