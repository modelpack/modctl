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
	"io"
	"net/http"
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

// checkHuggingFaceAuth checks if the user is authenticated with HuggingFace
func checkHuggingFaceAuth() error {
	// Try to find the HF token
	token := os.Getenv("HF_TOKEN")
	if token != "" {
		return nil
	}

	// Check if the token file exists
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get user home directory: %w", err)
	}

	tokenPath := filepath.Join(homeDir, ".huggingface", "token")
	if _, err := os.Stat(tokenPath); err == nil {
		return nil
	}

	// Try using whoami command
	if _, err := exec.LookPath("huggingface-cli"); err == nil {
		cmd := exec.Command("huggingface-cli", "whoami")
		cmd.Stdout = io.Discard
		cmd.Stderr = io.Discard
		if err := cmd.Run(); err == nil {
			return nil
		}
	}

	return fmt.Errorf("not authenticated with HuggingFace. Please run: huggingface-cli login")
}

// getToken retrieves the HuggingFace token from environment or token file
func getToken() (string, error) {
	// First check environment variable
	token := os.Getenv("HF_TOKEN")
	if token != "" {
		return token, nil
	}

	// Then check the token file
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}

	tokenPath := filepath.Join(homeDir, ".huggingface", "token")
	data, err := os.ReadFile(tokenPath)
	if err != nil {
		return "", fmt.Errorf("failed to read token file: %w", err)
	}

	return strings.TrimSpace(string(data)), nil
}

// downloadFile downloads a single file from HuggingFace
func downloadFile(ctx context.Context, owner, repo, filename, destPath string) error {
	token, err := getToken()
	if err != nil {
		return fmt.Errorf("failed to get HuggingFace token: %w", err)
	}

	// Construct the download URL
	// Format: https://huggingface.co/{owner}/{repo}/resolve/main/{filename}
	fileURL := fmt.Sprintf("%s/%s/%s/resolve/main/%s", huggingFaceBaseURL, owner, repo, filename)

	req, err := http.NewRequestWithContext(ctx, "GET", fileURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Add authorization header
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download file, status code: %d", resp.StatusCode)
	}

	// Create destination directory
	destDir := filepath.Dir(destPath)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Create the destination file
	outFile, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer outFile.Close()

	// Copy the content
	_, err = io.Copy(outFile, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}
