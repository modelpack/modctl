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

import "fmt"

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
}

func NewPull() *Pull {
	return &Pull{
		Concurrency:       defaultPullConcurrency,
		PlainHTTP:         false,
		Proxy:             "",
		Insecure:          false,
		ExtractDir:        "",
		ExtractFromRemote: false,
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
