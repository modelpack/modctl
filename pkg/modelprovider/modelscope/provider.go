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
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Provider implements the modelprovider.Provider interface for ModelScope
type Provider struct{}

// New creates a new ModelScope provider instance
func New() *Provider {
	return &Provider{}
}

// Name returns the name of this provider
func (p *Provider) Name() string {
	return "modelscope"
}

// SupportsURL checks if this provider can handle the given URL
// It only supports full ModelScope URLs with the modelscope.cn domain
// For short-form repo identifiers (owner/repo), users must explicitly specify --provider modelscope
func (p *Provider) SupportsURL(url string) bool {
	url = strings.TrimSpace(url)

	// Only support full ModelScope URLs
	return strings.Contains(url, "modelscope.cn")
}

// DownloadModel downloads a model from ModelScope using the modelscope CLI
func (p *Provider) DownloadModel(ctx context.Context, modelURL, destDir string) (string, error) {
	owner, repo, err := parseModelURL(modelURL)
	if err != nil {
		return "", err
	}

	repoID := fmt.Sprintf("%s/%s", owner, repo)

	// Check if modelscope CLI is available
	if _, err := exec.LookPath("modelscope"); err != nil {
		return "", fmt.Errorf("modelscope CLI not found in PATH. Please install it using: pip install modelscope")
	}

	// Create destination directory if it doesn't exist
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Construct the download path
	downloadPath := filepath.Join(destDir, repo)

	// Use modelscope download command
	// The modelscope CLI uses: modelscope download --model <model_id> --local_dir <path>
	cmd := exec.CommandContext(ctx, "modelscope", "download", "--model", repoID, "--local_dir", downloadPath)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to download model using modelscope CLI: %w", err)
	}

	return downloadPath, nil
}

// CheckAuth verifies that the user is authenticated with ModelScope
func (p *Provider) CheckAuth() error {
	return checkModelScopeAuth()
}
