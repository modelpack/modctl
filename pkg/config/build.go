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
	// defaultBuildConcurrency is the default number of concurrent builds.
	defaultBuildConcurrency = 5
)

type Build struct {
	Concurrency    int
	Target         string
	Modelfile      string
	OutputRemote   bool
	PlainHTTP      bool
	Insecure       bool
	Nydusify       bool
	SourceURL      string
	SourceRevision string
	Raw            bool
	Reasoning      bool
	NoCreationTime bool
}

func NewBuild() *Build {
	return &Build{
		Concurrency:    defaultBuildConcurrency,
		Target:         "",
		Modelfile:      "Modelfile",
		OutputRemote:   false,
		PlainHTTP:      false,
		Insecure:       false,
		Nydusify:       false,
		SourceURL:      "",
		SourceRevision: "",
		Raw:            false,
		Reasoning:      false,
		NoCreationTime: false,
	}
}

func (b *Build) Validate() error {
	if b.Concurrency <= 0 {
		return fmt.Errorf("concurrency must be greater than 0")
	}

	if len(b.Target) == 0 {
		return fmt.Errorf("target model artifact name is required")
	}

	if len(b.Modelfile) == 0 {
		return fmt.Errorf("model file path is required")
	}

	if b.Nydusify {
		if !b.OutputRemote {
			return fmt.Errorf("nydusify only works with output remote")
		}
	}

	return nil
}
