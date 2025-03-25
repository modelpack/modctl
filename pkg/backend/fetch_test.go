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

package backend

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	modelspec "github.com/CloudNativeAI/model-spec/specs-go/v1"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CloudNativeAI/modctl/pkg/config"
)

func TestFetch(t *testing.T) {
	// Setup temporary directory for output
	tempDir, err := os.MkdirTemp("", "fetch-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Setup mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v2/":
			// Registry API check
			w.WriteHeader(http.StatusOK)
		case "/v2/test/model/manifests/latest":
			// Return a manifest
			manifest := ocispec.Manifest{
				Layers: []ocispec.Descriptor{
					{
						MediaType: "application/octet-stream.raw",
						Digest:    "sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
						Size:      0,
						Annotations: map[string]string{
							modelspec.AnnotationFilepath: "file1.txt",
						},
					},
					{
						MediaType: "application/octet-stream.raw",
						Digest:    "sha256:a3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
						Size:      0,
						Annotations: map[string]string{
							modelspec.AnnotationFilepath: "file2.txt",
						},
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			require.NoError(t, json.NewEncoder(w).Encode(manifest))
		case "/v2/test/model/blobs/sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
			"/v2/test/model/blobs/sha256:a3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855":
			// Return empty content for blobs
			w.WriteHeader(http.StatusOK)
		default:
			t.Logf("Unexpected request to %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	// Create backend instance
	b := &backend{}

	url := strings.TrimPrefix(server.URL, "http://")

	// Setup test cases
	tests := []struct {
		name        string
		target      string
		cfg         *config.Fetch
		expectError bool
	}{
		{
			name:   "fetch with pattern matching file1",
			target: url + "/test/model:latest",
			cfg: &config.Fetch{
				Output:      tempDir,
				Patterns:    []string{"file1.txt"},
				PlainHTTP:   true,
				Concurrency: 2,
			},
			expectError: false,
		},
		{
			name:   "fetch with pattern matching both files",
			target: url + "/test/model:latest",
			cfg: &config.Fetch{
				Output:      tempDir,
				Patterns:    []string{"file*.txt"},
				PlainHTTP:   true,
				Concurrency: 2,
			},
			expectError: false,
		},
		{
			name:   "fetch with non-matching pattern",
			target: url + "/test/model:latest",
			cfg: &config.Fetch{
				Output:      tempDir,
				Patterns:    []string{"nonexistent.txt"},
				PlainHTTP:   true,
				Concurrency: 2,
			},
			expectError: true,
		},
		{
			name:   "fetch with invalid reference",
			target: "invalid-reference",
			cfg: &config.Fetch{
				Output:      tempDir,
				Patterns:    []string{"file1.txt"},
				PlainHTTP:   true,
				Concurrency: 2,
			},
			expectError: true,
		},
	}

	// Run tests
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := b.Fetch(context.Background(), tt.target, tt.cfg)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
