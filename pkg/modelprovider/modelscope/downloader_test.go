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

package modelscope

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
			name:      "full URL with models prefix",
			modelURL:  "https://modelscope.cn/models/qwen/Qwen-7B",
			wantOwner: "qwen",
			wantRepo:  "Qwen-7B",
			wantErr:   false,
		},
		{
			name:      "full URL without models prefix",
			modelURL:  "https://modelscope.cn/damo/nlp_structbert_backbone_base_std",
			wantOwner: "damo",
			wantRepo:  "nlp_structbert_backbone_base_std",
			wantErr:   false,
		},
		{
			name:      "full URL with trailing slash",
			modelURL:  "https://modelscope.cn/models/qwen/Qwen-7B/",
			wantOwner: "qwen",
			wantRepo:  "Qwen-7B",
			wantErr:   false,
		},
		{
			name:      "short form",
			modelURL:  "qwen/Qwen-7B",
			wantOwner: "qwen",
			wantRepo:  "Qwen-7B",
			wantErr:   false,
		},
		{
			name:      "http URL",
			modelURL:  "http://modelscope.cn/models/damo/nlp_structbert_backbone_base_std",
			wantOwner: "damo",
			wantRepo:  "nlp_structbert_backbone_base_std",
			wantErr:   false,
		},
		{
			name:        "invalid format - missing repo",
			modelURL:    "https://modelscope.cn/models/qwen",
			wantErr:     true,
			errContains: "invalid ModelScope URL format",
		},
		{
			name:        "invalid format - only owner",
			modelURL:    "qwen",
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
			modelURL:  "  qwen/Qwen-7B  ",
			wantOwner: "qwen",
			wantRepo:  "Qwen-7B",
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			owner, repo, err := parseModelURL(tt.modelURL)

			if tt.wantErr {
				if err == nil {
					t.Errorf("parseModelURL() expected error but got nil")
					return
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("parseModelURL() error = %v, want error containing %v", err, tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("parseModelURL() unexpected error = %v", err)
				return
			}

			if owner != tt.wantOwner {
				t.Errorf("parseModelURL() owner = %v, want %v", owner, tt.wantOwner)
			}

			if repo != tt.wantRepo {
				t.Errorf("parseModelURL() repo = %v, want %v", repo, tt.wantRepo)
			}
		})
	}
}

func TestProvider_SupportsURL(t *testing.T) {
	provider := New()

	tests := []struct {
		name string
		url  string
		want bool
	}{
		{
			name: "full ModelScope URL",
			url:  "https://modelscope.cn/models/qwen/Qwen-7B",
			want: true,
		},
		{
			name: "ModelScope URL without models prefix",
			url:  "https://modelscope.cn/damo/nlp_structbert_backbone_base_std",
			want: true,
		},
		{
			name: "HuggingFace URL",
			url:  "https://huggingface.co/meta-llama/Llama-2-7b-hf",
			want: false,
		},
		{
			name: "short form repo (ambiguous, returns false)",
			url:  "qwen/Qwen-7B",
			want: false,
		},
		{
			name: "invalid format",
			url:  "just-a-string",
			want: false,
		},
		{
			name: "HTTP URL",
			url:  "http://example.com/owner/repo",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := provider.SupportsURL(tt.url); got != tt.want {
				t.Errorf("Provider.SupportsURL() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProvider_Name(t *testing.T) {
	provider := New()
	if got := provider.Name(); got != "modelscope" {
		t.Errorf("Provider.Name() = %v, want %v", got, "modelscope")
	}
}
