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

package modelprovider

import "context"

// Provider defines the interface that all model providers must implement.
// A provider is responsible for downloading models from a specific source
// (e.g., HuggingFace, ModelScope, Civitai, etc.)
type Provider interface {
	// Name returns the human-readable name of the provider
	// Example: "huggingface", "modelscope", "civitai"
	Name() string

	// SupportsURL checks if this provider can handle the given model URL
	// This enables automatic provider detection based on URL patterns
	SupportsURL(url string) bool

	// DownloadModel downloads a model from the provider and returns the local path
	// Parameters:
	//   - ctx: context for cancellation and timeout
	//   - modelURL: the URL or identifier of the model to download
	//   - destDir: the destination directory where the model should be downloaded
	// Returns:
	//   - string: the local path where the model was downloaded
	//   - error: any error that occurred during download
	DownloadModel(ctx context.Context, modelURL, destDir string) (string, error)

	// CheckAuth verifies that the user is authenticated with the provider
	// Returns an error if authentication is missing or invalid
	CheckAuth() error
}
