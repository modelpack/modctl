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
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	huggingFaceBaseURL = "https://huggingface.co"
)

// parseModelURL parses a HuggingFace model URL and extracts the owner and repository name
func parseModelURL(modelURL string) (owner, repo string, err error) {
	// Handle both full URLs and short-form repo names
	modelURL = strings.TrimSpace(modelURL)

	// Remove trailing slashes
	modelURL = strings.TrimSuffix(modelURL, "/")

	// If it's a full URL, parse it
	if strings.HasPrefix(modelURL, "http://") || strings.HasPrefix(modelURL, "https://") {
		u, err := url.Parse(modelURL)
		if err != nil {
			return "", "", fmt.Errorf("invalid URL: %w", err)
		}

		// Expected format: https://huggingface.co/owner/repo
		parts := strings.Split(strings.Trim(u.Path, "/"), "/")
		if len(parts) < 2 {
			return "", "", fmt.Errorf("invalid HuggingFace URL format, expected https://huggingface.co/owner/repo")
		}

		owner = parts[0]
		repo = parts[1]
	} else {
		// Handle short-form like "owner/repo"
		parts := strings.Split(modelURL, "/")
		if len(parts) != 2 {
			return "", "", fmt.Errorf("invalid model identifier, expected format: owner/repo")
		}

		owner = parts[0]
		repo = parts[1]
	}

	if owner == "" || repo == "" {
		return "", "", fmt.Errorf("owner and repository name cannot be empty")
	}

	return owner, repo, nil
}

// tokenFilePaths returns the list of candidate token file paths to check,
// in priority order. It respects the HF_HOME environment variable and
// supports both legacy (~/.huggingface/token) and modern
// (~/.cache/huggingface/token) locations.
func tokenFilePaths() []string {
	var paths []string

	// If HF_HOME is set, check there first
	if hfHome := os.Getenv("HF_HOME"); hfHome != "" {
		paths = append(paths, filepath.Join(hfHome, "token"))
	}

	homeDir, err := os.UserHomeDir()
	if err == nil {
		// Modern location used by huggingface_hub >= 0.14 and the `hf` CLI
		paths = append(paths, filepath.Join(homeDir, ".cache", "huggingface", "token"))
		// Legacy location
		paths = append(paths, filepath.Join(homeDir, ".huggingface", "token"))
	}

	return paths
}

// checkHuggingFaceAuth checks if the user is authenticated with HuggingFace
func checkHuggingFaceAuth() error {
	// Try to find the HF token via environment variable
	token := os.Getenv("HF_TOKEN")
	if token != "" {
		return nil
	}

	// Check token file paths (modern and legacy locations)
	for _, tokenPath := range tokenFilePaths() {
		if _, err := os.Stat(tokenPath); err == nil {
			return nil
		}
	}

	// Try using whoami command with available CLI tool
	for _, cli := range []string{"hf", "huggingface-cli"} {
		if path, err := exec.LookPath(cli); err == nil {
			cmd := exec.Command(path, "whoami")
			cmd.Stdout = io.Discard
			cmd.Stderr = io.Discard
			if err := cmd.Run(); err == nil {
				return nil
			}
		}
	}

	return fmt.Errorf("not authenticated with HuggingFace. Please run: hf auth login")
}

// getToken retrieves the HuggingFace token from environment or token file
func getToken() (string, error) {
	// First check environment variable
	token := os.Getenv("HF_TOKEN")
	if token != "" {
		return token, nil
	}

	// Check token file paths (modern and legacy locations)
	for _, tokenPath := range tokenFilePaths() {
		data, err := os.ReadFile(tokenPath)
		if err == nil {
			return strings.TrimSpace(string(data)), nil
		}
	}

	return "", fmt.Errorf("HuggingFace token not found. Please run: hf auth login")
}
