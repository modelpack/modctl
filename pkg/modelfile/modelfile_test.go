/*
 *     Copyright 2024 The CNAI Authors
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

package modelfile

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewModelfile(t *testing.T) {
	testCases := []struct {
		input        string
		expectErr    error
		configs      []string
		models       []string
		codes        []string
		datasets     []string
		name         string
		arch         string
		family       string
		format       string
		paramsize    string
		precision    string
		quantization string
	}{
		{
			input: `
# This is a comment
config config1
model model1
code code1
dataset dataset1
name name1
arch arch1
family family1
format format1
paramsize paramsize1
precision precision1
quantization quantization1
`,
			expectErr:    nil,
			configs:      []string{"config1"},
			models:       []string{"model1"},
			codes:        []string{"code1"},
			datasets:     []string{"dataset1"},
			name:         "name1",
			arch:         "arch1",
			family:       "family1",
			format:       "format1",
			paramsize:    "paramsize1",
			precision:    "precision1",
			quantization: "quantization1",
		},
		{
			input: `
# This is a comment
config        config1
model         model1
code          code1
dataset       dataset1
name          name1
arch          arch1
family        family1
format        format1
paramsize     paramsize1
precision     precision1
quantization  quantization1
`,
			expectErr:    nil,
			configs:      []string{"config1"},
			models:       []string{"model1"},
			codes:        []string{"code1"},
			datasets:     []string{"dataset1"},
			name:         "name1",
			arch:         "arch1",
			family:       "family1",
			format:       "format1",
			paramsize:    "paramsize1",
			precision:    "precision1",
			quantization: "quantization1",
		},
		{
			input: `
config config1
config config2
model model1
model model2
code code1
code code2
dataset dataset1
dataset dataset2
name name1
arch arch1
family family1
format format1
paramsize paramsize1
precision precision1
quantization quantization1
		`,
			expectErr:    nil,
			configs:      []string{"config1", "config2"},
			models:       []string{"model1", "model2"},
			codes:        []string{"code1", "code2"},
			datasets:     []string{"dataset1", "dataset2"},
			name:         "name1",
			arch:         "arch1",
			family:       "family1",
			format:       "format1",
			paramsize:    "paramsize1",
			precision:    "precision1",
			quantization: "quantization1",
		},
		{
			input: `
config config1
config config1
config config2
model model1
model model1
model model2
code code1
code code1
code code2
dataset dataset1
dataset dataset1
dataset dataset2
name name1
arch arch1
family family1
format format1
paramsize paramsize1
precision precision1
quantization quantization1
		`,
			expectErr:    nil,
			configs:      []string{"config1", "config2"},
			models:       []string{"model1", "model2"},
			codes:        []string{"code1", "code2"},
			datasets:     []string{"dataset1", "dataset2"},
			name:         "name1",
			arch:         "arch1",
			family:       "family1",
			format:       "format1",
			paramsize:    "paramsize1",
			precision:    "precision1",
			quantization: "quantization1",
		},
		{
			input: `
invalid command
		`,
			expectErr: errors.New("parse error on line 1: invalid command"),
		},
		{
			input: `
# This is a comment
invalid command
		`,
			expectErr: errors.New("parse error on line 2: invalid command"),
		},
		{
			input: `


# This is a comment
invalid command
		`,
			expectErr: errors.New("parse error on line 4: invalid command"),
		},
		{
			input: `
# This is a comment

invalid command
		`,
			expectErr: errors.New("parse error on line 3: invalid command"),
		},
		{
			input: `
# This is a comment
model adapter1
name foo
name bar
		`,
			expectErr: errors.New("duplicate name command on line 4"),
		},
	}

	assert := assert.New(t)
	for _, tc := range testCases {
		tmpfile, err := os.CreateTemp("", "Modelfile")
		assert.NoError(err)

		_, err = tmpfile.WriteString(tc.input)
		assert.NoError(err)

		err = tmpfile.Close()
		assert.NoError(err)

		mf, err := NewModelfile(tmpfile.Name())
		if tc.expectErr != nil {
			assert.Equal(err, tc.expectErr)
			assert.Nil(mf)
			continue
		}

		assert.NoError(err)
		assert.NotNil(mf)
		configs := mf.GetConfigs()
		models := mf.GetModels()
		codes := mf.GetCodes()
		datasets := mf.GetDatasets()
		sort.Strings(configs)
		sort.Strings(models)
		sort.Strings(codes)
		sort.Strings(datasets)
		assert.Equal(tc.configs, configs)
		assert.Equal(tc.models, models)
		assert.Equal(tc.codes, codes)
		assert.Equal(tc.datasets, datasets)
		assert.Equal(tc.name, mf.GetName())
		assert.Equal(tc.arch, mf.GetArch())
		assert.Equal(tc.family, mf.GetFamily())
		assert.Equal(tc.format, mf.GetFormat())
		assert.Equal(tc.paramsize, mf.GetParamsize())
		assert.Equal(tc.precision, mf.GetPrecision())
		assert.Equal(tc.quantization, mf.GetQuantization())

		os.Remove(tmpfile.Name())
	}
}

func TestAutoModelfile(t *testing.T) {
	testCases := []struct {
		name      string
		files     map[string]string
		config    *ModelfileGenConfig
		expectErr error
		validate  func(*testing.T, Modelfile)
	}{
		{
			name: "basic model directory",
			files: map[string]string{
				"config.json":            `{"model_type": "llama", "transformers_version": "1.0", "torch_dtype": "float16"}`,
				"generation_config.json": `{}`,
				"tokenizer.model":        "dummy content",
				"pytorch_model.bin":      "dummy content",
				"model.safetensors":      "dummy content",
				"train.py":               "print('hello')",
				"README.md":              "# Model Documentation",
				".git/config":            "should be ignored",
				"__pycache__/cache.pyc":  "should be ignored",
			},
			config: &ModelfileGenConfig{
				Name:               "llama2-7b",
				Format:             "safetensors",
				Paramsize:          7000000000,
				Quantization:       "q4_k_m",
				IgnoreUnrecognized: true,
			},
			expectErr: nil,
			validate: func(t *testing.T, mf Modelfile) {
				assert := assert.New(t)

				// Check configs (sorted)
				expectedConfigs := []string{
					"README.md",
					"config.json",
					"generation_config.json",
					"tokenizer.model",
				}
				configs := mf.GetConfigs()
				sort.Strings(configs)
				assert.Equal(expectedConfigs, configs)

				// Check models (sorted)
				expectedModels := []string{
					"model.safetensors",
					"pytorch_model.bin",
				}
				models := mf.GetModels()
				sort.Strings(models)
				assert.Equal(expectedModels, models)

				// Check codes (sorted)
				expectedCodes := []string{
					"train.py",
				}
				codes := mf.GetCodes()
				sort.Strings(codes)
				assert.Equal(expectedCodes, codes)

				// Check other fields
				assert.Equal("llama2-7b", mf.GetName())
				assert.Equal("transformer", mf.GetArch()) // from config.json
				assert.Equal("llama", mf.GetFamily())     // from config.json
				assert.Equal("safetensors", mf.GetFormat())
				assert.Equal("7000000000", mf.GetParamsize())
				assert.Equal("float16", mf.GetPrecision()) // from config.json
				assert.Equal("q4_k_m", mf.GetQuantization())
			},
		},
		{
			name: "unrecognized files without ignore flag",
			files: map[string]string{
				"unknown.xyz": "some content",
			},
			config: &ModelfileGenConfig{
				Name:               "test-model",
				IgnoreUnrecognized: false,
			},
			expectErr: errors.New("unknown file type: unknown.xyz - use --ignore-unrecognized to ignore, and edit the Modelfile manually"),
		},
		{
			name:      "empty directory",
			files:     map[string]string{},
			config:    &ModelfileGenConfig{Name: "empty-model"},
			expectErr: errors.New("no recognized model files found in directory - you may need to edit the Modelfile manually"),
		},
		{
			name:      "invalid config json",
			files:     map[string]string{"config.json": `{"model_type": "llama", invalid json`},
			config:    &ModelfileGenConfig{Name: "invalid-config"},
			expectErr: errors.New("no recognized model files found in directory - you may need to edit the Modelfile manually"),
		},
		{
			name:      "nested directories",
			files:     map[string]string{"config.json": `{"model_type": "llama"}`, "models/shard1.safetensors": "dummy content", "models/shard2.safetensors": "dummy content", "configs/main.json": "dummy content", "src/train.py": "print('hello')"},
			config:    &ModelfileGenConfig{Name: "nested-model", IgnoreUnrecognized: true},
			expectErr: nil,
			validate: func(t *testing.T, mf Modelfile) {
				assert := assert.New(t)
				models := mf.GetModels()
				sort.Strings(models)
				assert.Equal([]string{"models/shard1.safetensors", "models/shard2.safetensors"}, models)

				codes := mf.GetCodes()
				sort.Strings(codes)
				assert.Equal([]string{"src/train.py"}, codes)
			},
		},
		{
			name: "special characters in paths",
			files: map[string]string{
				"config.json":                   `{"model_type": "llama"}`,
				"model with spaces.safetensors": "dummy content",
				"特殊字符.bin":                      "dummy content",
				"src/test-file.py":              "print('hello')",
			},
			config: &ModelfileGenConfig{
				Name:               "special-chars",
				IgnoreUnrecognized: true,
			},
			validate: func(t *testing.T, mf Modelfile) {
				assert := assert.New(t)
				models := mf.GetModels()
				sort.Strings(models)
				assert.Equal([]string{"model with spaces.safetensors", "特殊字符.bin"}, models)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create temporary directory
			tmpDir, err := os.MkdirTemp("", "modelfile_test_*")
			assert.NoError(t, err)
			defer os.RemoveAll(tmpDir)

			// Create test files
			for path, content := range tc.files {
				fullPath := filepath.Join(tmpDir, path)

				// Create parent directories if needed
				err := os.MkdirAll(filepath.Dir(fullPath), 0755)
				assert.NoError(t, err)

				err = os.WriteFile(fullPath, []byte(content), 0644)
				assert.NoError(t, err)
			}

			// Run AutoModelfile
			mf, err := AutoModelfile(tmpDir, tc.config)

			if tc.expectErr != nil {
				assert.Error(t, err)
				assert.Equal(t, tc.expectErr.Error(), err.Error())
				assert.Nil(t, mf)
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, mf)

			// Run validation
			tc.validate(t, mf)
		})
	}
}
