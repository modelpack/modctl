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

package hfhub

import (
	"strings"
	"testing"
)

func TestParseModelURL(t *testing.T) {
	tests := []struct {
		name        string
		modelURL    string
		wantOwner   string
		wantRepo    string
		wantErr     bool
		errContains string
	}{
		{
			name:      "full URL",
			modelURL:  "https://huggingface.co/meta-llama/Llama-2-7b-hf",
			wantOwner: "meta-llama",
			wantRepo:  "Llama-2-7b-hf",
			wantErr:   false,
		},
		{
			name:      "full URL with trailing slash",
			modelURL:  "https://huggingface.co/meta-llama/Llama-2-7b-hf/",
			wantOwner: "meta-llama",
			wantRepo:  "Llama-2-7b-hf",
			wantErr:   false,
		},
		{
			name:      "short form",
			modelURL:  "meta-llama/Llama-2-7b-hf",
			wantOwner: "meta-llama",
			wantRepo:  "Llama-2-7b-hf",
			wantErr:   false,
		},
		{
			name:      "http URL",
			modelURL:  "http://huggingface.co/openai/gpt-2",
			wantOwner: "openai",
			wantRepo:  "gpt-2",
			wantErr:   false,
		},
		{
			name:        "invalid format - missing repo",
			modelURL:    "https://huggingface.co/meta-llama",
			wantErr:     true,
			errContains: "invalid Hugging Face URL format",
		},
		{
			name:        "invalid format - only owner",
			modelURL:    "meta-llama",
			wantErr:     true,
			errContains: "invalid model identifier",
		},
		{
			name:        "empty URL",
			modelURL:    "",
			wantErr:     true,
			errContains: "invalid model identifier",
		},
		{
			name:      "URL with spaces (trimmed)",
			modelURL:  "  meta-llama/Llama-2-7b-hf  ",
			wantOwner: "meta-llama",
			wantRepo:  "Llama-2-7b-hf",
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			owner, repo, err := ParseModelURL(tt.modelURL)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseModelURL() expected error but got nil")
					return
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("ParseModelURL() error = %v, want error containing %v", err, tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("ParseModelURL() unexpected error = %v", err)
				return
			}

			if owner != tt.wantOwner {
				t.Errorf("ParseModelURL() owner = %v, want %v", owner, tt.wantOwner)
			}

			if repo != tt.wantRepo {
				t.Errorf("ParseModelURL() repo = %v, want %v", repo, tt.wantRepo)
			}
		})
	}
}
