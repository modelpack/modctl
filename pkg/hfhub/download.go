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
	HuggingFaceBaseURL = "https://huggingface.co"
)

// ParseModelURL parses a Hugging Face model URL and extracts the owner and repository name
func ParseModelURL(modelURL string) (owner, repo string, err error) {
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
			return "", "", fmt.Errorf("invalid Hugging Face URL format, expected https://huggingface.co/owner/repo")
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

// DownloadModel downloads a model from Hugging Face using the huggingface-cli
// It assumes the user is already logged in via `huggingface-cli login`
func DownloadModel(ctx context.Context, modelURL, destDir string) (string, error) {
	owner, repo, err := ParseModelURL(modelURL)
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

	fmt.Printf("Downloading model %s to %s...\n", repoID, downloadPath)

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to download model using huggingface-cli: %w", err)
	}

	fmt.Printf("Successfully downloaded model to %s\n", downloadPath)

	return downloadPath, nil
}

// CheckHuggingFaceAuth checks if the user is authenticated with Hugging Face
func CheckHuggingFaceAuth() error {
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
		if err := cmd.Run(); err == nil {
			return nil
		}
	}

	return fmt.Errorf("not authenticated with Hugging Face. Please run: huggingface-cli login")
}

// GetToken retrieves the Hugging Face token from environment or token file
func GetToken() (string, error) {
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

// DownloadFile downloads a single file from Hugging Face
func DownloadFile(ctx context.Context, owner, repo, filename, destPath string) error {
	token, err := GetToken()
	if err != nil {
		return fmt.Errorf("failed to get Hugging Face token: %w", err)
	}

	// Construct the download URL
	// Format: https://huggingface.co/{owner}/{repo}/resolve/main/{filename}
	fileURL := fmt.Sprintf("%s/%s/%s/resolve/main/%s", HuggingFaceBaseURL, owner, repo, filename)

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
