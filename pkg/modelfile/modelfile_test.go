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
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	configmodelfile "github.com/CloudNativeAI/modctl/pkg/config/modelfile"
	"github.com/emirpasic/gods/sets/hashset"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewModelfile(t *testing.T) {
	testCases := []struct {
		input        string
		expectErr    error
		configs      []string
		models       []string
		codes        []string
		datasets     []string
		docs         []string
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
CONFIG config1
MODEL model1
CODE code1
DATASET dataset1
DOC doc1
NAME name1
ARCH arch1
FAMILY family1
FORMAT format1
PARAMSIZE paramsize1
PRECISION precision1
QUANTIZATION quantization1
`,
			expectErr:    nil,
			configs:      []string{"config1"},
			models:       []string{"model1"},
			codes:        []string{"code1"},
			datasets:     []string{"dataset1"},
			docs:         []string{"doc1"},
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
CONFIG        config1
MODEL         model1
CODE          code1
DATASET       dataset1
DOC           doc1
NAME          name1
ARCH          arch1
FAMILY        family1
FORMAT        format1
PARAMSIZE     paramsize1
PRECISION     precision1
QUANTIZATION  quantization1
`,
			expectErr:    nil,
			configs:      []string{"config1"},
			models:       []string{"model1"},
			codes:        []string{"code1"},
			datasets:     []string{"dataset1"},
			docs:         []string{"doc1"},
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
CONFIG config1
CONFIG config2
MODEL model1
MODEL model2
CODE code1
CODE code2
DATASET dataset1
DATASET dataset2
DOC doc1
DOC doc2
NAME name1
ARCH arch1
FAMILY family1
FORMAT format1
PARAMSIZE paramsize1
PRECISION precision1
QUANTIZATION quantization1
		`,
			expectErr:    nil,
			configs:      []string{"config1", "config2"},
			models:       []string{"model1", "model2"},
			codes:        []string{"code1", "code2"},
			datasets:     []string{"dataset1", "dataset2"},
			docs:         []string{"doc1", "doc2"},
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
CONFIG config1
CONFIG config1
CONFIG config2
MODEL model1
MODEL model1
MODEL model2
CODE code1
CODE code1
CODE code2
DATASET dataset1
DATASET dataset1
DATASET dataset2
DOC doc1
DOC doc1
DOC doc2
NAME name1
ARCH arch1
FAMILY family1
FORMAT format1
PARAMSIZE paramsize1
PRECISION precision1
QUANTIZATION quantization1
		`,
			expectErr:    nil,
			configs:      []string{"config1", "config2"},
			models:       []string{"model1", "model2"},
			codes:        []string{"code1", "code2"},
			datasets:     []string{"dataset1", "dataset2"},
			docs:         []string{"doc1", "doc2"},
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
INVALID command
		`,
			expectErr: errors.New("parse error on line 1: INVALID command"),
		},
		{
			input: `
# This is a comment
INVALID command
		`,
			expectErr: errors.New("parse error on line 2: INVALID command"),
		},
		{
			input: `


# This is a comment
INVALID command
		`,
			expectErr: errors.New("parse error on line 4: INVALID command"),
		},
		{
			input: `
# This is a comment

INVALID command
		`,
			expectErr: errors.New("parse error on line 3: INVALID command"),
		},
		{
			input: `
# This is a comment
MODEL adapter1
NAME foo
NAME bar
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
		docs := mf.GetDocs()
		sort.Strings(configs)
		sort.Strings(models)
		sort.Strings(codes)
		sort.Strings(datasets)
		sort.Strings(docs)
		assert.Equal(tc.configs, configs)
		assert.Equal(tc.models, models)
		assert.Equal(tc.codes, codes)
		assert.Equal(tc.datasets, datasets)
		assert.Equal(tc.docs, docs)
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

func TestNewModelfileByWorkspace(t *testing.T) {
	testcases := []struct {
		name               string
		setupFiles         map[string]string
		setupDirs          []string
		configJson         map[string]interface{}
		genConfigJson      map[string]interface{}
		config             *configmodelfile.GenerateConfig
		expectError        bool
		expectConfigs      []string
		expectModels       []string
		expectCodes        []string
		expectDocs         []string
		expectName         string
		expectArch         string
		expectFamily       string
		expectFormat       string
		expectParamsize    string
		expectPrecision    string
		expectQuantization string
	}{
		{
			name: "basic case",
			setupFiles: map[string]string{
				"config.json":  "",
				"model.bin":    "",
				"model.py":     "",
				"tokenizer.py": "",
				"README.md":    "",
				"LICENSE":      "",
			},
			config: &configmodelfile.GenerateConfig{
				Name: "test-model",
			},
			expectError:   false,
			expectConfigs: []string{"config.json"},
			expectModels:  []string{"model.bin"},
			expectCodes:   []string{"model.py", "tokenizer.py"},
			expectDocs:    []string{"README.md", "LICENSE"},
			expectName:    "test-model",
		},
		{
			name:       "empty workspace",
			setupFiles: map[string]string{},
			config: &configmodelfile.GenerateConfig{
				Name: "empty-model",
			},
			expectError:   true,
			expectConfigs: []string{},
			expectModels:  []string{},
			expectCodes:   []string{},
			expectName:    "empty-model",
		},
		{
			name: "with config.json values",
			setupFiles: map[string]string{
				"model.bin":   "",
				"config.json": "",
			},
			configJson: map[string]interface{}{
				"model_type":           "llama",
				"torch_dtype":          "float16",
				"transformers_version": "4.28.0",
			},
			config: &configmodelfile.GenerateConfig{
				Name: "config-model",
			},
			expectError:     false,
			expectConfigs:   []string{"config.json"},
			expectModels:    []string{"model.bin"},
			expectCodes:     []string{},
			expectName:      "config-model",
			expectArch:      "transformer",
			expectFamily:    "llama",
			expectPrecision: "float16",
		},
		{
			name: "nested directory structure",
			setupFiles: map[string]string{
				"config.json":                 "",
				"weights/model.bin":           "",
				"weights/model.safetensors":   "",
				"src/utils.py":                "",
				"src/models/model.py":         "",
				"assets/README.md":            "",
				"assets/images/preview.jpg":   "",
				"docs/config/parameters.yaml": "",
			},
			setupDirs: []string{
				"weights",
				"src",
				"src/models",
				"assets",
				"assets/images",
				"docs/config",
			},
			config: &configmodelfile.GenerateConfig{
				Name: "nested-model",
			},
			expectError: false,
			expectConfigs: []string{
				"config.json",
				"docs/config/parameters.yaml",
			},
			expectModels: []string{
				"weights/model.bin",
				"weights/model.safetensors",
			},
			expectCodes: []string{
				"src/utils.py",
				"src/models/model.py",
			},
			expectDocs: []string{
				"assets/README.md",
				"assets/images/preview.jpg",
			},
			expectName: "nested-model",
		},
		{
			name: "deep nested directories",
			setupFiles: map[string]string{
				"level1/level2/level3/model.bin":      "",
				"level1/level2/level3/level4/code.py": "",
				"level1/config.json":                  "",
			},
			setupDirs: []string{
				"level1",
				"level1/level2",
				"level1/level2/level3",
				"level1/level2/level3/level4",
			},
			config: &configmodelfile.GenerateConfig{
				Name: "deep-nested",
			},
			expectError:   false,
			expectConfigs: []string{"level1/config.json"},
			expectModels:  []string{"level1/level2/level3/model.bin"},
			expectCodes:   []string{"level1/level2/level3/level4/code.py"},
			expectName:    "deep-nested",
		},
		{
			name: "hidden files and directories",
			setupFiles: map[string]string{
				"config.json":          "",
				"model.bin":            "",
				".hidden_file":         "",
				".hidden_dir/file.txt": "",
				"normal_dir/.hidden":   "",
			},
			setupDirs: []string{
				".hidden_dir",
				"normal_dir",
			},
			config: &configmodelfile.GenerateConfig{
				Name: "hidden-test",
			},
			expectError:   false,
			expectConfigs: []string{"config.json"},
			expectModels:  []string{"model.bin"},
			expectCodes:   []string{},
			expectName:    "hidden-test",
		},
		{
			name: "multiple config files in directories",
			setupFiles: map[string]string{
				"model.bin":              "",
				"models/config.json":     "",
				"models/gen_config.json": "",
			},
			setupDirs: []string{"models"},
			configJson: map[string]interface{}{
				"model_type":           "gpt2",
				"torch_dtype":          "float32",
				"transformers_version": "4.30.0",
			},
			config: &configmodelfile.GenerateConfig{
				Name:   "multi-config",
				Format: "pytorch",
			},
			expectError:     false,
			expectConfigs:   []string{"config.json", "models/config.json", "models/gen_config.json"},
			expectModels:    []string{"model.bin"},
			expectCodes:     []string{},
			expectName:      "multi-config",
			expectArch:      "transformer",
			expectFamily:    "gpt2",
			expectFormat:    "pytorch",
			expectPrecision: "float32",
		},
		{
			name: "special filename characters",
			setupFiles: map[string]string{
				"config with spaces.json":      "",
				"model-with-hyphens.bin":       "",
				"file_with_underscore.py":      "",
				"dir with spaces/model.bin":    "",
				"dir-with-hyphens/config.json": "",
				"dir_with_underscore/code.py":  "",
			},
			setupDirs: []string{
				"dir with spaces",
				"dir-with-hyphens",
				"dir_with_underscore",
			},
			config: &configmodelfile.GenerateConfig{
				Name: "special-chars",
			},
			expectError: false,
			expectConfigs: []string{
				"config with spaces.json",
				"dir-with-hyphens/config.json",
			},
			expectModels: []string{
				"model-with-hyphens.bin",
				"dir with spaces/model.bin",
			},
			expectCodes: []string{
				"file_with_underscore.py",
				"dir_with_underscore/code.py",
			},
			expectName: "special-chars",
		},
		{
			name: "mixed file types in nested dirs",
			setupFiles: map[string]string{
				"configs/main.json":     "",
				"configs/params.yaml":   "",
				"models/weights.bin":    "",
				"models/data/extra.bin": "",
				"src/main.py":           "",
				"src/utils/helpers.py":  "",
				"src/models/arch.py":    "",
			},
			setupDirs: []string{
				"configs",
				"models",
				"models/data",
				"src",
				"src/utils",
				"src/models",
			},
			config: &configmodelfile.GenerateConfig{
				Name: "mixed-types",
			},
			expectError: false,
			expectConfigs: []string{
				"configs/main.json",
				"configs/params.yaml",
			},
			expectModels: []string{
				"models/weights.bin",
				"models/data/extra.bin",
			},
			expectCodes: []string{
				"src/main.py",
				"src/utils/helpers.py",
				"src/models/arch.py",
			},
			expectName: "mixed-types",
		},
		{
			name: "same filenames across directories",
			setupFiles: map[string]string{
				"dir1/config.json": "",
				"dir2/config.json": "",
				"dir1/model.bin":   "",
				"dir2/model.bin":   "",
				"dir1/script.py":   "",
				"dir2/script.py":   "",
			},
			setupDirs: []string{
				"dir1",
				"dir2",
			},
			config: &configmodelfile.GenerateConfig{
				Name: "same-names",
			},
			expectError: false,
			expectConfigs: []string{
				"dir1/config.json",
				"dir2/config.json",
			},
			expectModels: []string{
				"dir1/model.bin",
				"dir2/model.bin",
			},
			expectCodes: []string{
				"dir1/script.py",
				"dir2/script.py",
			},
			expectName: "same-names",
		},
		{
			name: "realistic model structure",
			setupFiles: map[string]string{
				"README.md":                     "",
				"config.json":                   "",
				"generation_config.json":        "",
				"tokenizer_config.json":         "",
				"tokenizer.model":               "",
				"tokenizer.json":                "",
				"pytorch_model.bin":             "",
				"model.safetensors":             "",
				"special_tokens_map.json":       "",
				"training_args.bin":             "",
				"vocab.json":                    "",
				"merges.txt":                    "",
				"extra/usage_examples.ipynb":    "",
				"scripts/convert_weights.py":    "",
				"scripts/preprocessing/prep.py": "",
			},
			setupDirs: []string{
				"extra",
				"scripts",
				"scripts/preprocessing",
			},
			configJson: map[string]interface{}{
				"model_type":           "llama",
				"torch_dtype":          "bfloat16",
				"transformers_version": "4.32.0",
				"architectures":        []string{"LlamaForCausalLM"},
			},
			config: &configmodelfile.GenerateConfig{
				Name:      "llama-7b",
				ParamSize: "7B",
			},
			expectError: false,
			expectConfigs: []string{
				"config.json",
				"generation_config.json",
				"tokenizer_config.json",
				"tokenizer.model",
				"tokenizer.json",
				"special_tokens_map.json",
				"vocab.json",
			},
			expectModels: []string{
				"pytorch_model.bin",
				"model.safetensors",
				"training_args.bin",
			},
			expectCodes: []string{
				"extra/usage_examples.ipynb",
				"scripts/convert_weights.py",
				"scripts/preprocessing/prep.py",
			},
			expectDocs:      []string{"merges.txt", "README.md"},
			expectName:      "llama-7b",
			expectArch:      "transformer",
			expectFamily:    "llama",
			expectPrecision: "bfloat16",
			expectParamsize: "7B",
		},
		{
			name: "config file conflicts",
			setupFiles: map[string]string{
				"model.bin":              "",
				"config.json":            "",
				"generation_config.json": "",
			},
			configJson: map[string]interface{}{
				"model_type":  "gpt2",
				"torch_dtype": "float16",
			},
			genConfigJson: map[string]interface{}{
				"model_type":  "llama",
				"torch_dtype": "float32",
			},
			config: &configmodelfile.GenerateConfig{
				Name: "conflict-test",
			},
			expectError:     false,
			expectConfigs:   []string{"config.json", "generation_config.json"},
			expectModels:    []string{"model.bin"},
			expectCodes:     []string{},
			expectName:      "conflict-test",
			expectFamily:    "llama",
			expectPrecision: "float32",
		},
		{
			name: "skipping internal directories",
			setupFiles: map[string]string{
				"config.json":           "",
				".git/config":           "",
				"__pycache__/cache.pyc": "",
				".hidden/model.bin":     "",
				"normal/model.bin":      "",
				"valid_dir/model.py":    "",
			},
			setupDirs: []string{
				".git",
				"__pycache__",
				".hidden",
				"normal",
				"valid_dir",
			},
			config: &configmodelfile.GenerateConfig{
				Name: "skip-test",
			},
			expectError:   false,
			expectConfigs: []string{"config.json"},
			expectModels:  []string{"normal/model.bin"},
			expectCodes:   []string{"valid_dir/model.py"},
			expectName:    "skip-test",
		},
	}

	assert := assert.New(t)
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			// Create temporary workspace directory
			tempDir, err := os.MkdirTemp("", "modelfile-test-*")
			assert.NoError(err)
			defer os.RemoveAll(tempDir)

			// Create directories first
			for _, dir := range tc.setupDirs {
				err := os.MkdirAll(filepath.Join(tempDir, dir), 0755)
				assert.NoError(err)
			}

			// Create files
			for filename, content := range tc.setupFiles {
				// Ensure parent directory exists
				dir := filepath.Dir(filepath.Join(tempDir, filename))
				if dir != tempDir {
					err := os.MkdirAll(dir, 0755)
					assert.NoError(err)
				}

				path := filepath.Join(tempDir, filename)
				err := os.WriteFile(path, []byte(content), 0644)
				assert.NoError(err)
			}

			// Create config.json if needed
			if tc.configJson != nil {
				configData, err := json.Marshal(tc.configJson)
				assert.NoError(err)
				err = os.WriteFile(filepath.Join(tempDir, "config.json"), configData, 0644)
				assert.NoError(err)
			}

			// Create generation_config.json if needed
			if tc.genConfigJson != nil {
				configData, err := json.Marshal(tc.genConfigJson)
				assert.NoError(err)
				err = os.WriteFile(filepath.Join(tempDir, "generation_config.json"), configData, 0644)
				assert.NoError(err)
			}

			// Set workspace in config
			tc.config.Workspace = tempDir
			tc.config.IgnoreUnrecognizedFileTypes = false

			// Call the function being tested
			mf, err := NewModelfileByWorkspace(tempDir, tc.config)

			if tc.expectError {
				assert.Error(err)
				return
			}

			assert.NoError(err)
			assert.NotNil(mf)
			assert.Equal(tc.expectName, mf.GetName())
			assert.Equal(tc.expectArch, mf.GetArch())
			assert.Equal(tc.expectFamily, mf.GetFamily())
			assert.Equal(tc.expectFormat, mf.GetFormat())
			assert.Equal(tc.expectParamsize, mf.GetParamsize())
			assert.Equal(tc.expectPrecision, mf.GetPrecision())
			assert.Equal(tc.expectQuantization, mf.GetQuantization())
			assert.ElementsMatch(tc.expectConfigs, mf.GetConfigs())
			assert.ElementsMatch(tc.expectModels, mf.GetModels())
			assert.ElementsMatch(tc.expectCodes, mf.GetCodes())
			assert.ElementsMatch(tc.expectDocs, mf.GetDocs())
		})
	}
}

func TestModelfile_Content(t *testing.T) {
	testcases := []struct {
		name           string
		modelfile      *modelfile
		expectedParts  []string
		notExpectParts []string
	}{
		{
			name: "full modelfile",
			modelfile: &modelfile{
				name:         "test-model",
				arch:         "transformer",
				family:       "llama",
				format:       "safetensors",
				paramsize:    "7B",
				precision:    "float16",
				quantization: "int8",
				config:       createHashSet([]string{"config.json"}),
				model:        createHashSet([]string{"model.bin", "model.safetensors"}),
				code:         createHashSet([]string{"convert.py", "inference.py"}),
				doc:          createHashSet([]string{"README.md"}),
				dataset:      createHashSet([]string{}),
			},
			expectedParts: []string{
				"# Generated at",
				"# Model name",
				"NAME test-model",
				"# Model architecture",
				"ARCH transformer",
				"# Model family",
				"FAMILY llama",
				"# Model format",
				"FORMAT safetensors",
				"# Model paramsize",
				"PARAMSIZE 7B",
				"# Model precision",
				"PRECISION float16",
				"# Model quantization",
				"QUANTIZATION int8",
				"# Config files",
				"CONFIG config.json",
				"# Documentation files",
				"DOC README.md",
				"# Code files",
				"CODE convert.py",
				"CODE inference.py",
				"# Model files",
				"MODEL model.bin",
				"MODEL model.safetensors",
			},
			notExpectParts: []string{
				"DATASET",
			},
		},
		{
			name: "minimal modelfile",
			modelfile: &modelfile{
				name:    "minimal",
				config:  createHashSet([]string{}),
				model:   createHashSet([]string{}),
				code:    createHashSet([]string{}),
				doc:     createHashSet([]string{}),
				dataset: createHashSet([]string{}),
			},
			expectedParts: []string{
				"# Generated at",
				"# Model name",
				"NAME minimal",
			},
			notExpectParts: []string{
				"ARCH", "FAMILY", "FORMAT", "PARAMSIZE", "PRECISION", "QUANTIZATION",
				"CONFIG", "CODE", "MODEL", "DATASET", "DOC",
			},
		},
		{
			name: "tiny model",
			modelfile: &modelfile{
				name:         "tiny-gpt",
				arch:         "transformer",
				family:       "gpt2",
				format:       "pytorch",
				paramsize:    "125M",
				precision:    "float32",
				quantization: "",
				config:       createHashSet([]string{"config.json"}),
				model:        createHashSet([]string{"pytorch_model.bin"}),
				code:         createHashSet([]string{}),
				doc:          createHashSet([]string{}),
				dataset:      createHashSet([]string{}),
			},
			expectedParts: []string{
				"# Generated at",
				"# Model name",
				"NAME tiny-gpt",
				"# Model architecture",
				"ARCH transformer",
				"# Model family",
				"FAMILY gpt2",
				"# Model format",
				"FORMAT pytorch",
				"# Model paramsize",
				"PARAMSIZE 125M",
				"# Model precision",
				"PRECISION float32",
				"# Config files",
				"CONFIG config.json",
				"# Model files",
				"MODEL pytorch_model.bin",
			},
			notExpectParts: []string{
				"QUANTIZATION", "CODE", "DATASET", "DOC",
			},
		},
		{
			name: "billion parameter model",
			modelfile: &modelfile{
				name:         "llama-large",
				arch:         "transformer",
				family:       "llama",
				format:       "safetensors",
				paramsize:    "13B",
				precision:    "bfloat16",
				quantization: "",
				config:       createHashSet([]string{"config.json"}),
				model:        createHashSet([]string{"model-00001-of-00003.safetensors", "model-00002-of-00003.safetensors", "model-00003-of-00003.safetensors"}),
				code:         createHashSet([]string{}),
				doc:          createHashSet([]string{}),
				dataset:      createHashSet([]string{}),
			},
			expectedParts: []string{
				"# Generated at",
				"# Model name",
				"NAME llama-large",
				"# Model architecture",
				"ARCH transformer",
				"# Model family",
				"FAMILY llama",
				"# Model format",
				"FORMAT safetensors",
				"# Model paramsize",
				"PARAMSIZE 13B",
				"# Model precision",
				"PRECISION bfloat16",
				"# Config files",
				"CONFIG config.json",
				"# Model files",
				"MODEL model-00001-of-00003.safetensors",
				"MODEL model-00002-of-00003.safetensors",
				"MODEL model-00003-of-00003.safetensors",
			},
			notExpectParts: []string{
				"QUANTIZATION", "CODE", "DATASET", "DOC",
			},
		},
		{
			name: "quantized model",
			modelfile: &modelfile{
				name:         "mistral-quantized",
				arch:         "transformer",
				family:       "mistral",
				format:       "gguf",
				paramsize:    "7B",
				precision:    "",
				quantization: "Q4_K_M",
				config:       createHashSet([]string{"config.json"}),
				model:        createHashSet([]string{"model.gguf"}),
				code:         createHashSet([]string{}),
				doc:          createHashSet([]string{}),
			},
			expectedParts: []string{
				"# Generated at",
				"# Model name",
				"NAME mistral-quantized",
				"# Model architecture",
				"ARCH transformer",
				"# Model family",
				"FAMILY mistral",
				"# Model format",
				"FORMAT gguf",
				"# Model paramsize",
				"PARAMSIZE 7B",
				"# Model quantization",
				"QUANTIZATION Q4_K_M",
				"# Config files",
				"CONFIG config.json",
				"# Model files",
				"MODEL model.gguf",
			},
			notExpectParts: []string{
				"PRECISION", "CODE", "DATASET", "DOC",
			},
		},
		{
			name: "trillion parameter model",
			modelfile: &modelfile{
				name:         "mega-model",
				arch:         "moe",
				family:       "mixtral",
				format:       "pytorch",
				paramsize:    "1.5T",
				precision:    "float16",
				quantization: "",
				config:       createHashSet([]string{"config.json"}),
				model:        createHashSet([]string{"shard-00001.bin", "shard-00002.bin"}),
				code:         createHashSet([]string{}),
				doc:          createHashSet([]string{}),
			},
			expectedParts: []string{
				"# Generated at",
				"# Model name",
				"NAME mega-model",
				"# Model architecture",
				"ARCH moe",
				"# Model family",
				"FAMILY mixtral",
				"# Model format",
				"FORMAT pytorch",
				"# Model paramsize",
				"PARAMSIZE 1.5T",
				"# Model precision",
				"PRECISION float16",
				"# Config files",
				"CONFIG config.json",
				"# Model files",
				"MODEL shard-00001.bin",
				"MODEL shard-00002.bin",
			},
			notExpectParts: []string{
				"QUANTIZATION", "CODE", "DATASET", "DOC",
			},
		},
		{
			name: "with nested paths",
			modelfile: &modelfile{
				name:      "nested-paths-model",
				paramsize: "3B",
				config:    createHashSet([]string{"configs/main.json", "configs/tokenizer/config.json"}),
				model:     createHashSet([]string{"models/weights/pytorch_model.bin"}),
				code:      createHashSet([]string{"src/utils.py", "src/models/model.py"}),
				doc:       createHashSet([]string{}),
			},
			expectedParts: []string{
				"# Generated at",
				"# Model name",
				"NAME nested-paths-model",
				"# Model paramsize",
				"PARAMSIZE 3B",
				"# Config files",
				"CONFIG configs/main.json",
				"CONFIG configs/tokenizer/config.json",
				"# Model files",
				"MODEL models/weights/pytorch_model.bin",
				"# Code files",
				"CODE src/utils.py",
				"CODE src/models/model.py",
			},
			notExpectParts: []string{
				"ARCH", "FAMILY", "FORMAT", "PRECISION", "QUANTIZATION", "DATASET", "DOC",
			},
		},
		{
			name: "fractional paramsize",
			modelfile: &modelfile{
				name:      "fractional-size",
				paramsize: "2.7B",
				config:    createHashSet([]string{"config.json"}),
				model:     createHashSet([]string{"model.bin"}),
				code:      createHashSet([]string{}),
				doc:       createHashSet([]string{}),
			},
			expectedParts: []string{
				"# Generated at",
				"# Model name",
				"NAME fractional-size",
				"# Model paramsize",
				"PARAMSIZE 2.7B",
				"# Config files",
				"CONFIG config.json",
				"# Model files",
				"MODEL model.bin",
			},
			notExpectParts: []string{
				"ARCH", "FAMILY", "FORMAT", "PRECISION", "QUANTIZATION", "DATASET", "DOC", "CODE",
			},
		},
		{
			name: "all metadata no files",
			modelfile: &modelfile{
				name:         "metadata-only",
				arch:         "transformer",
				family:       "gpt-neox",
				format:       "pytorch",
				paramsize:    "20B",
				precision:    "bfloat16",
				quantization: "int4",
				config:       createHashSet([]string{}),
				model:        createHashSet([]string{}),
				code:         createHashSet([]string{}),
				doc:          createHashSet([]string{}),
			},
			expectedParts: []string{
				"# Generated at",
				"# Model name",
				"NAME metadata-only",
				"# Model architecture",
				"ARCH transformer",
				"# Model family",
				"FAMILY gpt-neox",
				"# Model format",
				"FORMAT pytorch",
				"# Model paramsize",
				"PARAMSIZE 20B",
				"# Model precision",
				"PRECISION bfloat16",
				"# Model quantization",
				"QUANTIZATION int4",
			},
			notExpectParts: []string{
				"CODE", "CONFIG", "DOC",
			},
		},
		{
			name: "files only no metadata",
			modelfile: &modelfile{
				name:    "files-only",
				config:  createHashSet([]string{"config.json"}),
				model:   createHashSet([]string{"model.bin"}),
				code:    createHashSet([]string{"script.py"}),
				doc:     createHashSet([]string{"README.md"}),
				dataset: createHashSet([]string{}),
			},
			expectedParts: []string{
				"# Generated at",
				"# Model name",
				"NAME files-only",
				"# Config files",
				"CONFIG config.json",
				"# Model files",
				"MODEL model.bin",
				"# Code files",
				"CODE script.py",
				"# Documentation files",
				"DOC README.md",
			},
			notExpectParts: []string{
				"ARCH", "FAMILY", "FORMAT", "PRECISION", "QUANTIZATION", "PARAMSIZE", "DATASET",
			},
		},
		{
			name: "multiple files of same type",
			modelfile: &modelfile{
				name:      "multi-file",
				paramsize: "7B",
				config:    createHashSet([]string{"config1.json", "config2.json", "config3.json"}),
				model:     createHashSet([]string{"model1.bin", "model2.bin", "model3.bin", "model4.bin"}),
				code:      createHashSet([]string{"script1.py", "script2.py"}),
				doc:       createHashSet([]string{"README1.md", "README2.md"}),
			},
			expectedParts: []string{
				"# Generated at",
				"# Model name",
				"NAME multi-file",
				"# Model paramsize",
				"PARAMSIZE 7B",
				"# Config files",
				"CONFIG config1.json",
				"CONFIG config2.json",
				"CONFIG config3.json",
				"# Model files",
				"MODEL model1.bin",
				"MODEL model2.bin",
				"MODEL model3.bin",
				"MODEL model4.bin",
				"# Code files",
				"CODE script1.py",
				"CODE script2.py",
				"# Documentation files",
				"DOC README1.md",
				"DOC README2.md",
			},
			notExpectParts: []string{
				"ARCH", "FAMILY", "FORMAT", "PRECISION", "QUANTIZATION", "DATASET",
			},
		},
		{
			name: "with special characters in paths",
			modelfile: &modelfile{
				name:      "special-chars",
				paramsize: "1B",
				config:    createHashSet([]string{"spaces.json", "weird-name!.yaml"}),
				model:     createHashSet([]string{"model-v1.0_beta.bin"}),
				code:      createHashSet([]string{"spaces/script.py"}),
				doc:       createHashSet([]string{"weird-name!.md"}),
			},
			expectedParts: []string{
				"# Generated at",
				"# Model name",
				"NAME special-chars",
				"# Model paramsize",
				"PARAMSIZE 1B",
				"# Config files",
				"CONFIG spaces.json",
				"CONFIG weird-name!.yaml",
				"# Model files",
				"MODEL model-v1.0_beta.bin",
				"# Code files",
				"CODE spaces/script.py",
				"# Documentation files",
				"DOC weird-name!.md",
			},
			notExpectParts: []string{
				"ARCH", "FAMILY", "FORMAT", "PRECISION", "QUANTIZATION", "DATASET",
			},
		},
	}

	assert := assert.New(t)
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			// Generate content.
			content := string(tc.modelfile.Content())

			// Check that all expected parts are present.
			for _, part := range tc.expectedParts {
				assert.Contains(content, part, "content: %s", content)
			}

			// Check that all non-expected parts are absent
			for _, part := range tc.notExpectParts {
				assert.NotContains(content, part, "content: %s", content)
			}
		})
	}
}

// createHashSet creates a hashset from a string slice.
func createHashSet(items []string) *hashset.Set {
	set := hashset.New()
	for _, item := range items {
		set.Add(item)
	}
	return set
}

// TestGenerateByModelConfig tests the generateByModelConfig method
func TestGenerateByModelConfig(t *testing.T) {
	testcases := []struct {
		name              string
		configFiles       map[string]map[string]interface{}
		expectedArch      string
		expectedFamily    string
		expectedPrecision string
		expectError       bool
	}{
		{
			name: "config.json with all fields",
			configFiles: map[string]map[string]interface{}{
				"config.json": {
					"model_type":           "llama",
					"torch_dtype":          "float16",
					"transformers_version": "4.30.0",
				},
			},
			expectedArch:      "transformer",
			expectedFamily:    "llama",
			expectedPrecision: "float16",
			expectError:       false,
		},
		{
			name: "generation_config.json overrides config.json",
			configFiles: map[string]map[string]interface{}{
				"config.json": {
					"model_type":  "llama",
					"torch_dtype": "float16",
				},
				"generation_config.json": {
					"model_type":  "gpt2",
					"torch_dtype": "float32",
				},
			},
			expectedFamily:    "gpt2",
			expectedPrecision: "float32",
			expectError:       false,
		},
		{
			name: "invalid json file",
			configFiles: map[string]map[string]interface{}{
				"config.json": nil, // This will create invalid JSON
			},
			expectError: false, // Invalid JSON is silently ignored
		},
		{
			name:        "no config files",
			configFiles: map[string]map[string]interface{}{},
			expectError: false,
		},
		{
			name: "partial config",
			configFiles: map[string]map[string]interface{}{
				"config.json": {
					"model_type": "bert",
				},
			},
			expectedFamily: "bert",
			expectError:    false,
		},
		{
			name: "transformers version only",
			configFiles: map[string]map[string]interface{}{
				"config.json": {
					"transformers_version": "4.25.1",
				},
			},
			expectedArch: "transformer",
			expectError:  false,
		},
	}

	assert := assert.New(t)
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			tempDir, err := os.MkdirTemp("", "model-config-test-*")
			require.NoError(t, err)
			defer os.RemoveAll(tempDir)

			// Create config files
			for filename, content := range tc.configFiles {
				if content == nil {
					// Create invalid JSON
					err = os.WriteFile(filepath.Join(tempDir, filename), []byte("invalid json"), 0644)
				} else {
					data, err := json.Marshal(content)
					require.NoError(t, err)
					err = os.WriteFile(filepath.Join(tempDir, filename), data, 0644)
				}
				require.NoError(t, err)
			}

			mf := &modelfile{workspace: tempDir}
			err = mf.generateByModelConfig()

			if tc.expectError {
				assert.Error(err)
			} else {
				assert.NoError(err)
				assert.Equal(tc.expectedArch, mf.arch)
				assert.Equal(tc.expectedFamily, mf.family)
				assert.Equal(tc.expectedPrecision, mf.precision)
			}
		})
	}
}

// TestGenerateByConfig tests the generateByConfig method
func TestGenerateByConfig(t *testing.T) {
	testcases := []struct {
		name                 string
		workspace            string
		config               *configmodelfile.GenerateConfig
		expectedName         string
		expectedArch         string
		expectedFamily       string
		expectedFormat       string
		expectedParamsize    string
		expectedPrecision    string
		expectedQuantization string
	}{
		{
			name:         "default name from workspace",
			workspace:    "/path/to/my-model",
			config:       &configmodelfile.GenerateConfig{},
			expectedName: "my-model",
		},
		{
			name:      "custom name overrides workspace",
			workspace: "/path/to/workspace",
			config: &configmodelfile.GenerateConfig{
				Name: "custom-model",
			},
			expectedName: "custom-model",
		},
		{
			name:      "all fields provided",
			workspace: "/path/to/model",
			config: &configmodelfile.GenerateConfig{
				Name:         "test-model",
				Arch:         "transformer",
				Family:       "llama",
				Format:       "safetensors",
				ParamSize:    "7B",
				Precision:    "float16",
				Quantization: "int8",
			},
			expectedName:         "test-model",
			expectedArch:         "transformer",
			expectedFamily:       "llama",
			expectedFormat:       "safetensors",
			expectedParamsize:    "7B",
			expectedPrecision:    "float16",
			expectedQuantization: "int8",
		},
		{
			name:         "empty config uses workspace name",
			workspace:    "/tmp/test-workspace",
			config:       &configmodelfile.GenerateConfig{},
			expectedName: "test-workspace",
		},
	}

	assert := assert.New(t)
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			mf := &modelfile{workspace: tc.workspace}
			mf.generateByConfig(tc.config)

			assert.Equal(tc.expectedName, mf.name)
			assert.Equal(tc.expectedArch, mf.arch)
			assert.Equal(tc.expectedFamily, mf.family)
			assert.Equal(tc.expectedFormat, mf.format)
			assert.Equal(tc.expectedParamsize, mf.paramsize)
			assert.Equal(tc.expectedPrecision, mf.precision)
			assert.Equal(tc.expectedQuantization, mf.quantization)
		})
	}
}

// TestValidateWorkspace tests the validateWorkspace method specifically
func TestValidateWorkspace(t *testing.T) {
	testcases := []struct {
		name        string
		setupFunc   func() (string, func())
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid directory workspace",
			setupFunc: func() (string, func()) {
				tempDir, err := os.MkdirTemp("", "valid-workspace-*")
				require.NoError(t, err)
				// Create a file to make it non-empty
				err = os.WriteFile(filepath.Join(tempDir, "test.txt"), []byte("test"), 0644)
				require.NoError(t, err)
				return tempDir, func() { os.RemoveAll(tempDir) }
			},
			expectError: false,
		},
		{
			name: "empty directory workspace",
			setupFunc: func() (string, func()) {
				tempDir, err := os.MkdirTemp("", "empty-workspace-*")
				require.NoError(t, err)
				return tempDir, func() { os.RemoveAll(tempDir) }
			},
			expectError: true,
			errorMsg:    "the workspace is empty",
		},
		{
			name: "non-existent workspace",
			setupFunc: func() (string, func()) {
				return "/non/existent/path", func() {}
			},
			expectError: true,
			errorMsg:    "access to workspace failed",
		},
		{
			name: "file instead of directory",
			setupFunc: func() (string, func()) {
				tempFile, err := os.CreateTemp("", "file-workspace-*")
				require.NoError(t, err)
				tempFile.Close()
				return tempFile.Name(), func() { os.Remove(tempFile.Name()) }
			},
			expectError: true,
			errorMsg:    "the workspace is not a directory",
		},
		{
			name: "symbolic link workspace",
			setupFunc: func() (string, func()) {
				tempDir, err := os.MkdirTemp("", "symlink-target-*")
				require.NoError(t, err)
				// Create a file in target
				err = os.WriteFile(filepath.Join(tempDir, "test.txt"), []byte("test"), 0644)
				require.NoError(t, err)

				symlinkPath := tempDir + "-symlink"
				err = os.Symlink(tempDir, symlinkPath)
				require.NoError(t, err)

				return symlinkPath, func() {
					os.RemoveAll(tempDir)
					os.Remove(symlinkPath)
				}
			},
			expectError: true,
			errorMsg:    "the workspace should not be a symbolic link",
		},
	}

	assert := assert.New(t)
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			workspace, cleanup := tc.setupFunc()
			defer cleanup()

			mf := &modelfile{workspace: workspace}
			err := mf.validateWorkspace()

			if tc.expectError {
				assert.Error(err)
				assert.Contains(err.Error(), tc.errorMsg)
			} else {
				assert.NoError(err)
			}
		})
	}
}

func TestWorkspaceLimits(t *testing.T) {
	testcases := []struct {
		name        string
		setupFunc   func() (string, func())
		expectError bool
		errorMsg    string
	}{
		{
			name: "exceeds file count limit",
			setupFunc: func() (string, func()) {
				tempDir, err := os.MkdirTemp("", "file-count-test-*")
				require.NoError(t, err)

				// Create more files than the limit (1024)
				for i := 0; i < MaxWorkspaceFileCount+10; i++ {
					filename := fmt.Sprintf("file_%d.txt", i)
					err = os.WriteFile(filepath.Join(tempDir, filename), []byte("test"), 0644)
					require.NoError(t, err)
				}

				return tempDir, func() { os.RemoveAll(tempDir) }
			},
			expectError: true,
			errorMsg:    "exceeds maximum file count limit",
		},
		{
			name: "normal sized files should pass",
			setupFunc: func() (string, func()) {
				tempDir, err := os.MkdirTemp("", "normal-file-test-*")
				require.NoError(t, err)

				// Create normal sized files including a model file
				normalPath := filepath.Join(tempDir, "model.bin")
				err = os.WriteFile(normalPath, []byte("test model content"), 0644)
				require.NoError(t, err)

				// Add a config file too
				configPath := filepath.Join(tempDir, "config.json")
				err = os.WriteFile(configPath, []byte(`{"model_type": "test"}`), 0644)
				require.NoError(t, err)

				return tempDir, func() { os.RemoveAll(tempDir) }
			},
			expectError: false,
		},
		{
			name: "within limits",
			setupFunc: func() (string, func()) {
				tempDir, err := os.MkdirTemp("", "within-limits-test-*")
				require.NoError(t, err)

				// Create a reasonable number of files including valid model/code files
				for i := 0; i < 8; i++ {
					filename := fmt.Sprintf("file_%d.txt", i)
					err = os.WriteFile(filepath.Join(tempDir, filename), []byte("test content"), 0644)
					require.NoError(t, err)
				}

				// Add valid model and config files
				err = os.WriteFile(filepath.Join(tempDir, "model.bin"), []byte("model content"), 0644)
				require.NoError(t, err)
				err = os.WriteFile(filepath.Join(tempDir, "config.json"), []byte(`{"model_type": "test"}`), 0644)
				require.NoError(t, err)

				return tempDir, func() { os.RemoveAll(tempDir) }
			},
			expectError: false,
		},
	}

	assert := assert.New(t)
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			workspace, cleanup := tc.setupFunc()
			defer cleanup()

			config := &configmodelfile.GenerateConfig{
				Name: "test-model",
			}

			_, err := NewModelfileByWorkspace(workspace, config)

			if tc.expectError {
				assert.Error(err)
				assert.Contains(err.Error(), tc.errorMsg)
			} else {
				assert.NoError(err)
			}
		})
	}
}

func TestFileTypeClassification(t *testing.T) {
	testcases := []struct {
		name            string
		files           map[string]int64 // filename -> size
		expectedConfigs []string
		expectedModels  []string
		expectedCodes   []string
		expectedDocs    []string
	}{
		{
			name: "various file types",
			files: map[string]int64{
				"config.json":         1024,
				"model.bin":           1024 * 1024 * 1024, // 1GB - large file
				"script.py":           2048,
				"README.md":           512,
				"tokenizer.json":      1024,
				"weights.safetensors": 2 * 1024 * 1024 * 1024, // 2GB - large file
				"inference.py":        3072,
				"LICENSE":             256,
			},
			expectedConfigs: []string{"config.json", "tokenizer.json"},
			expectedModels:  []string{"model.bin", "weights.safetensors"},
			expectedCodes:   []string{"script.py", "inference.py"},
			expectedDocs:    []string{"README.md", "LICENSE"},
		},
		{
			name: "small unknown files treated as code files",
			files: map[string]int64{
				"unknown_small_file":  1024,      // 1KB - below threshold
				"another_unknown.xyz": 50 * 1024, // 50KB - below threshold
				"config.json":         1024,      // Add a config to make workspace valid
			},
			expectedConfigs: []string{"config.json"},
			expectedModels:  []string{},
			expectedCodes:   []string{"unknown_small_file", "another_unknown.xyz"},
		},
		{
			name: "case insensitive file extensions",
			files: map[string]int64{
				"CONFIG.JSON": 1024,
				"Model.BIN":   1024,
				"Script.PY":   1024,
				"README.MD":   1024,
			},
			expectedConfigs: []string{"CONFIG.JSON"},
			expectedModels:  []string{"Model.BIN"},
			expectedCodes:   []string{"Script.PY"},
			expectedDocs:    []string{"README.MD"},
		},
		{
			name: "nested directory files",
			files: map[string]int64{
				"configs/model.json":       1024,
				"models/pytorch_model.bin": 1024,
				"src/utils.py":             1024,
				"docs/guide.md":            1024,
			},
			expectedConfigs: []string{"configs/model.json"},
			expectedModels:  []string{"models/pytorch_model.bin"},
			expectedCodes:   []string{"src/utils.py"},
			expectedDocs:    []string{"docs/guide.md"},
		},
	}

	assert := assert.New(t)
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			tempDir, err := os.MkdirTemp("", "file-type-test-*")
			require.NoError(t, err)
			defer os.RemoveAll(tempDir)

			// Create files with specified sizes
			for filename, size := range tc.files {
				fullPath := filepath.Join(tempDir, filename)

				// Create directory if needed
				dir := filepath.Dir(fullPath)
				if dir != tempDir {
					err = os.MkdirAll(dir, 0755)
					require.NoError(t, err)
				}

				// Create file with specified size
				file, err := os.Create(fullPath)
				require.NoError(t, err)

				if size > 0 {
					// For large files, we'll write a smaller amount and then seek to create the size
					// For testing purposes, just write some content
					content := strings.Repeat("x", int(min(size, 1024)))
					_, err = file.WriteString(content)
					require.NoError(t, err)
				}
				file.Close()
			}

			config := &configmodelfile.GenerateConfig{
				Name: "test-classification",
			}

			mf, err := NewModelfileByWorkspace(tempDir, config)
			require.NoError(t, err)

			assert.ElementsMatch(tc.expectedConfigs, mf.GetConfigs())
			assert.ElementsMatch(tc.expectedModels, mf.GetModels())
			assert.ElementsMatch(tc.expectedCodes, mf.GetCodes())
			assert.ElementsMatch(tc.expectedDocs, mf.GetDocs())
		})
	}
}

func TestSkippedFiles(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "skip-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create various files and directories that should be skipped
	filesToCreate := []string{
		".hidden_file",
		".git/config",
		"__pycache__/cache.pyc",
		"model.pyo",
		"script.pyd",
		"modelfile",
		"normal_file.txt",
		"valid_model.bin",
	}

	dirsToCreate := []string{
		".git",
		"__pycache__",
		".hidden_dir",
		"normal_dir",
	}

	for _, dir := range dirsToCreate {
		err = os.MkdirAll(filepath.Join(tempDir, dir), 0755)
		require.NoError(t, err)
	}

	for _, file := range filesToCreate {
		fullPath := filepath.Join(tempDir, file)
		dir := filepath.Dir(fullPath)
		if dir != tempDir {
			err = os.MkdirAll(dir, 0755)
			require.NoError(t, err)
		}
		err = os.WriteFile(fullPath, []byte("content"), 0644)
		require.NoError(t, err)
	}

	config := &configmodelfile.GenerateConfig{
		Name: "skip-test",
	}

	mf, err := NewModelfileByWorkspace(tempDir, config)
	require.NoError(t, err)

	// Only normal_file.txt and valid_model.bin should be included
	allFiles := append(append(append(mf.GetConfigs(), mf.GetModels()...), mf.GetCodes()...), mf.GetDocs()...)

	assert := assert.New(t)

	// Check that skipped files are not included
	for _, file := range allFiles {
		assert.NotContains(file, ".hidden")
		assert.NotContains(file, ".git")
		assert.NotContains(file, "__pycache__")
		assert.NotContains(file, ".pyc")
		assert.NotContains(file, ".pyo")
		assert.NotContains(file, ".pyd")
		assert.NotEqual(file, "modelfile")
	}

	// Check that normal files are included
	expectedFiles := []string{"normal_file.txt", "valid_model.bin"}
	for _, expectedFile := range expectedFiles {
		found := false
		for _, file := range allFiles {
			if strings.Contains(file, expectedFile) {
				found = true
				break
			}
		}
		assert.True(found, "Expected file %s should be included", expectedFile)
	}
}

func TestEmptyWorkspaceHandling(t *testing.T) {
	testcases := []struct {
		name        string
		files       []string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "only documentation files",
			files:       []string{"README.md", "LICENSE", "docs.txt"},
			expectError: true,
			errorMsg:    "no model/code/dataset found",
		},
		{
			name:        "only configuration files",
			files:       []string{"config.json", "settings.yaml"},
			expectError: true,
			errorMsg:    "no model/code/dataset found",
		},
		{
			name:        "has model files",
			files:       []string{"model.bin", "config.json"},
			expectError: false,
		},
		{
			name:        "has code files",
			files:       []string{"script.py", "config.json"},
			expectError: false,
		},
		{
			name:        "mixed valid files",
			files:       []string{"model.bin", "script.py", "README.md"},
			expectError: false,
		},
	}

	assert := assert.New(t)
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			tempDir, err := os.MkdirTemp("", "empty-workspace-test-*")
			require.NoError(t, err)
			defer os.RemoveAll(tempDir)

			for _, filename := range tc.files {
				err = os.WriteFile(filepath.Join(tempDir, filename), []byte("content"), 0644)
				require.NoError(t, err)
			}

			config := &configmodelfile.GenerateConfig{
				Name: "test-model",
			}

			_, err = NewModelfileByWorkspace(tempDir, config)

			if tc.expectError {
				assert.Error(err)
				assert.Contains(err.Error(), tc.errorMsg)
			} else {
				assert.NoError(err)
			}
		})
	}
}

// min returns the minimum of two integers
func min(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}
