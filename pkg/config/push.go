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
	// defaultPushConcurrency is the default number of concurrent push operations.
	defaultPushConcurrency = 5
)

type Push struct {
	Concurrency int
	PlainHTTP   bool
}

func NewPush() *Pull {
	return &Pull{
		Concurrency: defaultPushConcurrency,
		PlainHTTP:   false,
	}
}

func (p *Push) Validate() error {
	if p.Concurrency < 1 {
		return fmt.Errorf("invalid concurrency: %d", p.Concurrency)
	}

	return nil
}
