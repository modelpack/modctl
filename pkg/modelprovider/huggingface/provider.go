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

package huggingface

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Provider implements the modelprovider.Provider interface for HuggingFace
type Provider struct{}

// New creates a new HuggingFace provider instance
func New() *Provider {
	return &Provider{}
}

// Name returns the name of this provider
func (p *Provider) Name() string {
	return "huggingface"
}

// SupportsURL checks if this provider can handle the given URL
// It only supports full HuggingFace URLs with the huggingface.co domain
// For short-form repo identifiers (owner/repo), users must explicitly specify --provider huggingface
func (p *Provider) SupportsURL(url string) bool {
	url = strings.TrimSpace(url)

	// Only support full HuggingFace URLs
	return strings.Contains(url, "huggingface.co")
}

// DownloadModel downloads a model from HuggingFace using the huggingface-cli
func (p *Provider) DownloadModel(ctx context.Context, modelURL, destDir string) (string, error) {
	owner, repo, err := parseModelURL(modelURL)
	if err != nil {
		return "", err
	}

	repoID := fmt.Sprintf("%s/%s", owner, repo)

	// Check if huggingface-cli is available
	if _, err := exec.LookPath("huggingface-cli"); err != nil {
		return "", fmt.Errorf("huggingface-cli not found in PATH. Please install it using: pip install huggingface_hub[cli]")
	}

	// Create destination directory if it doesn't exist
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Construct the download path
	downloadPath := filepath.Join(destDir, repo)

	// Use huggingface-cli to download the model
	// The --local-dir-use-symlinks=False flag ensures files are copied, not symlinked
	cmd := exec.CommandContext(ctx, "huggingface-cli", "download", repoID, "--local-dir", downloadPath, "--local-dir-use-symlinks", "False")

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to download model using huggingface-cli: %w", err)
	}

	return downloadPath, nil
}

// CheckAuth verifies that the user is authenticated with HuggingFace
func (p *Provider) CheckAuth() error {
	return checkHuggingFaceAuth()
}
