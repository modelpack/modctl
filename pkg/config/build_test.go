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
	"testing"
)

func TestNewBuild(t *testing.T) {
	build := NewBuild()
	if build.Concurrency == 0 {
		t.Errorf("expected Concurrency to be greater than 0, got %d", build.Concurrency)
	}

	if build.Target != "" {
		t.Errorf("expected Target to be empty, got %s", build.Target)
	}

	if build.Modelfile != "Modelfile" {
		t.Errorf("expected Modelfile to be 'Modelfile', got %s", build.Modelfile)
	}
}

func TestBuild_Validate(t *testing.T) {
	tests := []struct {
		name      string
		build     *Build
		expectErr bool
	}{
		{
			name: "valid build",
			build: &Build{
				Concurrency: 1,
				Target:      "target",
				Modelfile:   "Modelfile",
			},
			expectErr: false,
		},
		{
			name: "missing concurrency",
			build: &Build{
				Concurrency: 0,
				Target:      "target",
				Modelfile:   "Modelfile",
			},
			expectErr: true,
		},
		{
			name: "missing target",
			build: &Build{
				Target:    "",
				Modelfile: "Modelfile",
			},
			expectErr: true,
		},
		{
			name: "missing modelfile",
			build: &Build{
				Target:    "target",
				Modelfile: "",
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.build.Validate()
			if (err != nil) != tt.expectErr {
				t.Errorf("expected error: %v, got: %v", tt.expectErr, err)
			}
		})
	}
}
