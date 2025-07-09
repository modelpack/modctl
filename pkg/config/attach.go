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

package config

import "fmt"

type Attach struct {
	Source       string
	Target       string
	OutputRemote bool
	PlainHTTP    bool
	Insecure     bool
	Nydusify     bool
	Force        bool
	Raw          bool
	Config       bool
}

func NewAttach() *Attach {
	return &Attach{
		Source:       "",
		Target:       "",
		OutputRemote: false,
		PlainHTTP:    false,
		Insecure:     false,
		Nydusify:     false,
		Force:        false,
		Raw:          false,
		Config:       false,
	}
}

func (a *Attach) Validate() error {
	if a.Source == "" || a.Target == "" {
		return fmt.Errorf("source and target must be specified")
	}

	if a.Nydusify {
		if !a.OutputRemote {
			return fmt.Errorf("nydusify only works with output remote")
		}
	}

	return nil
}
