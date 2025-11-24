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
	"testing"
)

func TestRegistry_GetProvider(t *testing.T) {
	registry := NewRegistry()

	tests := []struct {
		name         string
		modelURL     string
		wantProvider string
		wantErr      bool
	}{
		{
			name:         "HuggingFace full URL",
			modelURL:     "https://huggingface.co/meta-llama/Llama-2-7b-hf",
			wantProvider: "huggingface",
			wantErr:      false,
		},
		{
			name:         "HuggingFace short form",
			modelURL:     "meta-llama/Llama-2-7b-hf",
			wantProvider: "huggingface",
			wantErr:      false,
		},
		{
			name:         "ModelScope full URL",
			modelURL:     "https://modelscope.cn/models/qwen/Qwen-7B",
			wantProvider: "modelscope",
			wantErr:      false,
		},
		{
			name:         "ModelScope URL without models prefix",
			modelURL:     "https://modelscope.cn/damo/nlp_structbert_backbone_base_std",
			wantProvider: "modelscope",
			wantErr:      false,
		},
		{
			name:     "Unsupported URL",
			modelURL: "https://example.com/model/repo",
			wantErr:  true,
		},
		{
			name:     "Invalid format",
			modelURL: "just-a-string",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := registry.GetProvider(tt.modelURL)

			if tt.wantErr {
				if err == nil {
					t.Errorf("GetProvider() expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("GetProvider() unexpected error = %v", err)
				return
			}

			if provider.Name() != tt.wantProvider {
				t.Errorf("GetProvider() provider name = %v, want %v", provider.Name(), tt.wantProvider)
			}
		})
	}
}

func TestRegistry_GetProviderByName(t *testing.T) {
	registry := NewRegistry()

	tests := []struct {
		name         string
		providerName string
		wantErr      bool
	}{
		{
			name:         "Get HuggingFace provider",
			providerName: "huggingface",
			wantErr:      false,
		},
		{
			name:         "Get ModelScope provider",
			providerName: "modelscope",
			wantErr:      false,
		},
		{
			name:         "Get non-existent provider",
			providerName: "civitai",
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := registry.GetProviderByName(tt.providerName)

			if tt.wantErr {
				if err == nil {
					t.Errorf("GetProviderByName() expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("GetProviderByName() unexpected error = %v", err)
				return
			}

			if provider.Name() != tt.providerName {
				t.Errorf("GetProviderByName() provider name = %v, want %v", provider.Name(), tt.providerName)
			}
		})
	}
}

func TestRegistry_ListProviders(t *testing.T) {
	registry := NewRegistry()
	providers := registry.ListProviders()

	if len(providers) != 2 {
		t.Errorf("ListProviders() returned %d providers, want 2", len(providers))
	}

	expectedProviders := map[string]bool{
		"huggingface": false,
		"modelscope":  false,
	}

	for _, name := range providers {
		if _, ok := expectedProviders[name]; !ok {
			t.Errorf("ListProviders() returned unexpected provider: %s", name)
		}
		expectedProviders[name] = true
	}

	for name, found := range expectedProviders {
		if !found {
			t.Errorf("ListProviders() missing expected provider: %s", name)
		}
	}
}
