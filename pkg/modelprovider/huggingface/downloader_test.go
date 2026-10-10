/*
 *     Copyright 2025 The ModelPack Authors
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

package huggingface

import (
	"os"
	"path/filepath"
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
			errContains: "invalid HuggingFace URL format",
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
			name: "full HuggingFace URL",
			url:  "https://huggingface.co/meta-llama/Llama-2-7b-hf",
			want: true,
		},
		{
			name: "short form repo (requires explicit --provider)",
			url:  "meta-llama/Llama-2-7b-hf",
			want: false,
		},
		{
			name: "ModelScope URL",
			url:  "https://modelscope.cn/models/owner/repo",
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
	if got := provider.Name(); got != "huggingface" {
		t.Errorf("Provider.Name() = %v, want %v", got, "huggingface")
	}
}

func TestTokenFilePaths(t *testing.T) {
	t.Run("without HF_HOME", func(t *testing.T) {
		// Ensure HF_HOME is unset
		t.Setenv("HF_HOME", "")

		paths := tokenFilePaths()

		// Should have exactly 2 paths: modern and legacy (no HF_HOME path)
		if len(paths) != 2 {
			t.Fatalf("tokenFilePaths() returned %d paths, want 2", len(paths))
		}

		// First should be the modern cache path
		if !strings.Contains(paths[0], filepath.Join(".cache", "huggingface", "token")) {
			t.Errorf("tokenFilePaths()[0] = %q, want path containing .cache/huggingface/token", paths[0])
		}

		// Second should be the legacy path
		if !strings.Contains(paths[1], filepath.Join(".huggingface", "token")) {
			t.Errorf("tokenFilePaths()[1] = %q, want path containing .huggingface/token", paths[1])
		}
	})

	t.Run("with HF_HOME", func(t *testing.T) {
		customDir := t.TempDir()
		t.Setenv("HF_HOME", customDir)

		paths := tokenFilePaths()

		// Should have 3 paths: HF_HOME, modern, legacy
		if len(paths) != 3 {
			t.Fatalf("tokenFilePaths() returned %d paths, want 3", len(paths))
		}

		// First should be the HF_HOME path
		expected := filepath.Join(customDir, "token")
		if paths[0] != expected {
			t.Errorf("tokenFilePaths()[0] = %q, want %q", paths[0], expected)
		}
	})
}

func TestCheckHuggingFaceAuth(t *testing.T) {
	t.Run("authenticated via HF_TOKEN env var", func(t *testing.T) {
		t.Setenv("HF_TOKEN", "hf_test_token_123")

		err := checkHuggingFaceAuth()
		if err != nil {
			t.Errorf("checkHuggingFaceAuth() returned error %v, want nil", err)
		}
	})

	t.Run("authenticated via token file at HF_HOME", func(t *testing.T) {
		t.Setenv("HF_TOKEN", "")

		tmpDir := t.TempDir()
		t.Setenv("HF_HOME", tmpDir)
		// Override HOME so the modern/legacy paths don't accidentally find a real token
		t.Setenv("HOME", t.TempDir())

		tokenPath := filepath.Join(tmpDir, "token")
		if err := os.WriteFile(tokenPath, []byte("hf_test_token"), 0644); err != nil {
			t.Fatal(err)
		}

		err := checkHuggingFaceAuth()
		if err != nil {
			t.Errorf("checkHuggingFaceAuth() returned error %v, want nil", err)
		}
	})

	t.Run("authenticated via modern token path", func(t *testing.T) {
		t.Setenv("HF_TOKEN", "")
		t.Setenv("HF_HOME", "")

		fakeHome := t.TempDir()
		t.Setenv("HOME", fakeHome)

		tokenPath := filepath.Join(fakeHome, ".cache", "huggingface", "token")
		if err := os.MkdirAll(filepath.Dir(tokenPath), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(tokenPath, []byte("hf_test_token"), 0644); err != nil {
			t.Fatal(err)
		}

		err := checkHuggingFaceAuth()
		if err != nil {
			t.Errorf("checkHuggingFaceAuth() returned error %v, want nil", err)
		}
	})

	t.Run("authenticated via legacy token path", func(t *testing.T) {
		t.Setenv("HF_TOKEN", "")
		t.Setenv("HF_HOME", "")

		fakeHome := t.TempDir()
		t.Setenv("HOME", fakeHome)

		tokenPath := filepath.Join(fakeHome, ".huggingface", "token")
		if err := os.MkdirAll(filepath.Dir(tokenPath), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(tokenPath, []byte("hf_test_token"), 0644); err != nil {
			t.Fatal(err)
		}

		err := checkHuggingFaceAuth()
		if err != nil {
			t.Errorf("checkHuggingFaceAuth() returned error %v, want nil", err)
		}
	})

	t.Run("not authenticated", func(t *testing.T) {
		t.Setenv("HF_TOKEN", "")
		t.Setenv("HF_HOME", "")
		t.Setenv("HOME", t.TempDir())
		// Ensure no CLI tools are found by overriding PATH
		t.Setenv("PATH", t.TempDir())

		err := checkHuggingFaceAuth()
		if err == nil {
			t.Error("checkHuggingFaceAuth() returned nil, want error")
		}
		if err != nil && !strings.Contains(err.Error(), "not authenticated") {
			t.Errorf("checkHuggingFaceAuth() error = %q, want error containing 'not authenticated'", err.Error())
		}
	})
}

func TestGetToken(t *testing.T) {
	t.Run("token from HF_TOKEN env var", func(t *testing.T) {
		t.Setenv("HF_TOKEN", "hf_env_token_abc")

		token, err := getToken()
		if err != nil {
			t.Fatalf("getToken() returned error: %v", err)
		}
		if token != "hf_env_token_abc" {
			t.Errorf("getToken() = %q, want %q", token, "hf_env_token_abc")
		}
	})

	t.Run("token from HF_HOME file", func(t *testing.T) {
		t.Setenv("HF_TOKEN", "")

		tmpDir := t.TempDir()
		t.Setenv("HF_HOME", tmpDir)
		t.Setenv("HOME", t.TempDir())

		tokenPath := filepath.Join(tmpDir, "token")
		if err := os.WriteFile(tokenPath, []byte("  hf_file_token_xyz  \n"), 0644); err != nil {
			t.Fatal(err)
		}

		token, err := getToken()
		if err != nil {
			t.Fatalf("getToken() returned error: %v", err)
		}
		if token != "hf_file_token_xyz" {
			t.Errorf("getToken() = %q, want %q (should be trimmed)", token, "hf_file_token_xyz")
		}
	})

	t.Run("token from modern cache path", func(t *testing.T) {
		t.Setenv("HF_TOKEN", "")
		t.Setenv("HF_HOME", "")

		fakeHome := t.TempDir()
		t.Setenv("HOME", fakeHome)

		tokenPath := filepath.Join(fakeHome, ".cache", "huggingface", "token")
		if err := os.MkdirAll(filepath.Dir(tokenPath), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(tokenPath, []byte("hf_modern_token"), 0644); err != nil {
			t.Fatal(err)
		}

		token, err := getToken()
		if err != nil {
			t.Fatalf("getToken() returned error: %v", err)
		}
		if token != "hf_modern_token" {
			t.Errorf("getToken() = %q, want %q", token, "hf_modern_token")
		}
	})

	t.Run("token from legacy path", func(t *testing.T) {
		t.Setenv("HF_TOKEN", "")
		t.Setenv("HF_HOME", "")

		fakeHome := t.TempDir()
		t.Setenv("HOME", fakeHome)

		tokenPath := filepath.Join(fakeHome, ".huggingface", "token")
		if err := os.MkdirAll(filepath.Dir(tokenPath), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(tokenPath, []byte("hf_legacy_token"), 0644); err != nil {
			t.Fatal(err)
		}

		token, err := getToken()
		if err != nil {
			t.Fatalf("getToken() returned error: %v", err)
		}
		if token != "hf_legacy_token" {
			t.Errorf("getToken() = %q, want %q", token, "hf_legacy_token")
		}
	})

	t.Run("modern path takes priority over legacy", func(t *testing.T) {
		t.Setenv("HF_TOKEN", "")
		t.Setenv("HF_HOME", "")

		fakeHome := t.TempDir()
		t.Setenv("HOME", fakeHome)

		// Create both modern and legacy token files
		modernPath := filepath.Join(fakeHome, ".cache", "huggingface", "token")
		if err := os.MkdirAll(filepath.Dir(modernPath), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(modernPath, []byte("modern_token"), 0644); err != nil {
			t.Fatal(err)
		}

		legacyPath := filepath.Join(fakeHome, ".huggingface", "token")
		if err := os.MkdirAll(filepath.Dir(legacyPath), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(legacyPath, []byte("legacy_token"), 0644); err != nil {
			t.Fatal(err)
		}

		token, err := getToken()
		if err != nil {
			t.Fatalf("getToken() returned error: %v", err)
		}
		if token != "modern_token" {
			t.Errorf("getToken() = %q, want %q (modern should take priority)", token, "modern_token")
		}
	})

	t.Run("no token found", func(t *testing.T) {
		t.Setenv("HF_TOKEN", "")
		t.Setenv("HF_HOME", "")
		t.Setenv("HOME", t.TempDir())

		_, err := getToken()
		if err == nil {
			t.Error("getToken() returned nil error, want error")
		}
		if err != nil && !strings.Contains(err.Error(), "token not found") {
			t.Errorf("getToken() error = %q, want error containing 'token not found'", err.Error())
		}
	})
}
