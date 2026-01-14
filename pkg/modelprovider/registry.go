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

import (
	"fmt"
	"sync"

	"github.com/modelpack/modctl/pkg/modelprovider/huggingface"
	"github.com/modelpack/modctl/pkg/modelprovider/mlflow"
	"github.com/modelpack/modctl/pkg/modelprovider/modelscope"
)

// Registry manages all available model providers and provides
// functionality to select the appropriate provider for a given URL
type Registry struct {
	providers []Provider
}

var (
	instance *Registry
	once     sync.Once
)

// GetRegistry returns the singleton instance of the registry
// This is thread-safe and will only create the instance once
func GetRegistry() *Registry {
	once.Do(func() {
		instance = &Registry{
			providers: []Provider{
				huggingface.New(),
				modelscope.New(),
				mlflow.New(),
				// Future providers can be added here:
				// civitai.New(),
			},
		}
	})
	return instance
}

// ResetRegistry resets the singleton instance
// This should only be used in tests to ensure isolation between test cases
func ResetRegistry() {
	once = sync.Once{}
	instance = nil
}

// GetProvider returns the appropriate provider for the given model URL
// It iterates through all registered providers and returns the first one
// that supports the URL. This only works for full URLs with domain names.
// For short-form URLs (owner/repo), use GetProviderByName with an explicit provider
func (r *Registry) GetProvider(modelURL string) (Provider, error) {
	for _, p := range r.providers {
		if p.SupportsURL(modelURL) {
			return p, nil
		}
	}
	return nil, fmt.Errorf("no provider found for URL: %s. For short-form URLs (owner/repo), use --provider flag to specify the provider explicitly", modelURL)
}

// SelectProvider returns the appropriate provider based on the URL and explicit provider name
// If providerName is specified, it uses GetProviderByName for short-form URLs
// Otherwise, it uses GetProvider for auto-detection with full URLs
func (r *Registry) SelectProvider(modelURL, providerName string) (Provider, error) {
	if providerName != "" {
		// Explicit provider specified, use it
		return r.GetProviderByName(providerName)
	}
	// No explicit provider, try auto-detection
	return r.GetProvider(modelURL)
}

// GetProviderByName returns a specific provider by its name
// This is useful when you want to explicitly select a provider
func (r *Registry) GetProviderByName(name string) (Provider, error) {
	for _, p := range r.providers {
		if p.Name() == name {
			return p, nil
		}
	}
	return nil, fmt.Errorf("provider not found: %s", name)
}

// ListProviders returns the names of all registered providers
func (r *Registry) ListProviders() []string {
	names := make([]string, len(r.providers))
	for i, p := range r.providers {
		names[i] = p.Name()
	}
	return names
}
