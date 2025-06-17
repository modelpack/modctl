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
	"testing"

	configmodelfile "github.com/CloudNativeAI/modctl/pkg/config/modelfile"
	modelspec "github.com/CloudNativeAI/model-spec/specs-go/v1"
	"github.com/emirpasic/gods/sets/hashset"
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

func TestValidateWorkspace(t *testing.T) {
	tests := []struct {
		name          string
		setupFunc     func() (string, func()) // returns workspace path and cleanup function
		expectedError string
	}{
		{
			name: "valid_directory",
			setupFunc: func() (string, func()) {
				tmpDir, err := os.MkdirTemp("", "validate_workspace_test")
				if err != nil {
					t.Fatalf("Failed to create temp dir: %v", err)
				}

				// Create a test file to make directory non-empty
				testFile := filepath.Join(tmpDir, "test.txt")
				err = os.WriteFile(testFile, []byte("test content"), 0644)
				if err != nil {
					os.RemoveAll(tmpDir)
					t.Fatalf("Failed to create test file: %v", err)
				}

				return tmpDir, func() { os.RemoveAll(tmpDir) }
			},
			expectedError: "",
		},
		{
			name: "non_existent_directory",
			setupFunc: func() (string, func()) {
				return "/non/existent/path", func() {}
			},
			expectedError: "access to workspace failed:",
		},
		{
			name: "file_instead_of_directory",
			setupFunc: func() (string, func()) {
				tmpDir, err := os.MkdirTemp("", "validate_workspace_test")
				if err != nil {
					t.Fatalf("Failed to create temp dir: %v", err)
				}

				testFile := filepath.Join(tmpDir, "test.txt")
				err = os.WriteFile(testFile, []byte("test content"), 0644)
				if err != nil {
					os.RemoveAll(tmpDir)
					t.Fatalf("Failed to create test file: %v", err)
				}

				return testFile, func() { os.RemoveAll(tmpDir) }
			},
			expectedError: "the workspace is not a directory:",
		},
		{
			name: "empty_directory",
			setupFunc: func() (string, func()) {
				tmpDir, err := os.MkdirTemp("", "validate_workspace_test")
				if err != nil {
					t.Fatalf("Failed to create temp dir: %v", err)
				}
				return tmpDir, func() { os.RemoveAll(tmpDir) }
			},
			expectedError: "the workspace is empty:",
		},
		{
			name: "symbolic_link",
			setupFunc: func() (string, func()) {
				tmpDir, err := os.MkdirTemp("", "validate_workspace_test")
				if err != nil {
					t.Fatalf("Failed to create temp dir: %v", err)
				}

				// Create target directory with content
				targetDir := filepath.Join(tmpDir, "target")
				err = os.Mkdir(targetDir, 0755)
				if err != nil {
					os.RemoveAll(tmpDir)
					t.Fatalf("Failed to create target dir: %v", err)
				}

				testFile := filepath.Join(targetDir, "test.txt")
				err = os.WriteFile(testFile, []byte("test content"), 0644)
				if err != nil {
					os.RemoveAll(tmpDir)
					t.Fatalf("Failed to create test file: %v", err)
				}

				// Create symbolic link
				linkPath := filepath.Join(tmpDir, "link")
				err = os.Symlink(targetDir, linkPath)
				if err != nil {
					os.RemoveAll(tmpDir)
					t.Fatalf("Failed to create symlink: %v", err)
				}

				return linkPath, func() { os.RemoveAll(tmpDir) }
			},
			expectedError: "for simplicity, the workspace should not be a symbolic link:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workspace, cleanup := tt.setupFunc()
			defer cleanup()

			mf := &modelfile{
				workspace: workspace,
				config:    hashset.New(),
				model:     hashset.New(),
				code:      hashset.New(),
				dataset:   hashset.New(),
				doc:       hashset.New(),
			}

			err := mf.validateWorkspace()

			if tt.expectedError == "" {
				assert.NoError(t, err, "Expected no error for test case: %s", tt.name)
			} else {
				assert.Error(t, err, "Expected error for test case: %s", tt.name)
				assert.Contains(t, err.Error(), tt.expectedError, "Error message should contain expected text for test case: %s", tt.name)
			}
		})
	}
}

func TestWorkspaceLimits(t *testing.T) {
	tests := []struct {
		name          string
		setupFunc     func() (string, func()) // returns workspace path and cleanup function
		expectedError string
	}{
		{
			name: "single_file_exceeds_128GB_limit",
			setupFunc: func() (string, func()) {
				tmpDir, err := os.MkdirTemp("", "workspace_limits_test")
				if err != nil {
					t.Fatalf("Failed to create temp dir: %v", err)
				}

				// Create a test file that simulates exceeding 128GB
				// We'll use a sparse file to avoid actually creating 128GB+ of data
				testFile := filepath.Join(tmpDir, "large_model.bin")
				file, err := os.Create(testFile)
				if err != nil {
					os.RemoveAll(tmpDir)
					t.Fatalf("Failed to create test file: %v", err)
				}

				// Seek to position that would make file appear larger than 128GB
				largeSize := MaxSingleFileSize + 1
				_, err = file.Seek(largeSize-1, 0)
				if err != nil {
					file.Close()
					os.RemoveAll(tmpDir)
					t.Fatalf("Failed to seek in file: %v", err)
				}

				// Write one byte at the end to make the file that size
				_, err = file.Write([]byte{0})
				if err != nil {
					file.Close()
					os.RemoveAll(tmpDir)
					t.Fatalf("Failed to write to file: %v", err)
				}
				file.Close()

				return tmpDir, func() { os.RemoveAll(tmpDir) }
			},
			expectedError: "exceeds maximum single file size limit",
		},
		{
			name: "file_count_exceeds_2048_limit",
			setupFunc: func() (string, func()) {
				tmpDir, err := os.MkdirTemp("", "workspace_limits_test")
				if err != nil {
					t.Fatalf("Failed to create temp dir: %v", err)
				}

				// Create more than 2048 files
				for i := 0; i <= MaxWorkspaceFileCount; i++ {
					testFile := filepath.Join(tmpDir, fmt.Sprintf("file_%d.txt", i))
					err = os.WriteFile(testFile, []byte("test"), 0644)
					if err != nil {
						os.RemoveAll(tmpDir)
						t.Fatalf("Failed to create test file %d: %v", i, err)
					}
				}

				return tmpDir, func() { os.RemoveAll(tmpDir) }
			},
			expectedError: "exceeds maximum file count limit",
		},
		{
			name: "total_workspace_size_exceeds_8TB_limit",
			setupFunc: func() (string, func()) {
				tmpDir, err := os.MkdirTemp("", "workspace_limits_test")
				if err != nil {
					t.Fatalf("Failed to create temp dir: %v", err)
				}

				// Create a few large files that together exceed 8TB
				// Each file will be just under 128GB (single file limit)
				// We'll create 70 files of ~120GB each to exceed 8TB total
				fileSize := MaxSingleFileSize - (1024 * 1024 * 1024) // 127GB per file
				numFiles := 70                                       // 70 * 127GB = ~8.9TB

				for i := 0; i < numFiles; i++ {
					testFile := filepath.Join(tmpDir, fmt.Sprintf("file_%d.bin", i))
					file, err := os.Create(testFile)
					if err != nil {
						os.RemoveAll(tmpDir)
						t.Fatalf("Failed to create test file %d: %v", i, err)
					}

					// Use sparse file technique
					_, err = file.Seek(fileSize-1, 0)
					if err != nil {
						file.Close()
						os.RemoveAll(tmpDir)
						t.Fatalf("Failed to seek in file %d: %v", i, err)
					}

					_, err = file.Write([]byte{0})
					if err != nil {
						file.Close()
						os.RemoveAll(tmpDir)
						t.Fatalf("Failed to write to file %d: %v", i, err)
					}
					file.Close()
				}

				return tmpDir, func() { os.RemoveAll(tmpDir) }
			},
			expectedError: "exceeds maximum total size limit",
		},
		{
			name: "workspace_within_all_limits",
			setupFunc: func() (string, func()) {
				tmpDir, err := os.MkdirTemp("", "workspace_limits_test")
				if err != nil {
					t.Fatalf("Failed to create temp dir: %v", err)
				}

				// Create a reasonable number of small files
				for i := 0; i < 10; i++ {
					testFile := filepath.Join(tmpDir, fmt.Sprintf("small_file_%d.txt", i))
					err = os.WriteFile(testFile, []byte("small content"), 0644)
					if err != nil {
						os.RemoveAll(tmpDir)
						t.Fatalf("Failed to create test file %d: %v", i, err)
					}
				}

				// Add a config file to make it a valid workspace
				configFile := filepath.Join(tmpDir, "config.json")
				err = os.WriteFile(configFile, []byte(`{"model_type": "test"}`), 0644)
				if err != nil {
					os.RemoveAll(tmpDir)
					t.Fatalf("Failed to create config file: %v", err)
				}

				// Add a model file to make it a valid workspace
				modelFile := filepath.Join(tmpDir, "model.safetensors")
				err = os.WriteFile(modelFile, []byte("fake model data"), 0644)
				if err != nil {
					os.RemoveAll(tmpDir)
					t.Fatalf("Failed to create model file: %v", err)
				}

				return tmpDir, func() { os.RemoveAll(tmpDir) }
			},
			expectedError: "",
		},
		{
			name: "exactly_at_file_count_limit",
			setupFunc: func() (string, func()) {
				tmpDir, err := os.MkdirTemp("", "workspace_limits_test")
				if err != nil {
					t.Fatalf("Failed to create temp dir: %v", err)
				}

				// Create exactly 2048 files (should be allowed)
				// Include one model file to make it valid
				modelFile := filepath.Join(tmpDir, "model.safetensors")
				err = os.WriteFile(modelFile, []byte("fake model data"), 0644)
				if err != nil {
					os.RemoveAll(tmpDir)
					t.Fatalf("Failed to create model file: %v", err)
				}

				// Create the remaining files to reach exactly 2048
				for i := 1; i < MaxWorkspaceFileCount; i++ {
					testFile := filepath.Join(tmpDir, fmt.Sprintf("file_%d.txt", i))
					err = os.WriteFile(testFile, []byte("test"), 0644)
					if err != nil {
						os.RemoveAll(tmpDir)
						t.Fatalf("Failed to create test file %d: %v", i, err)
					}
				}

				return tmpDir, func() { os.RemoveAll(tmpDir) }
			},
			expectedError: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workspace, cleanup := tt.setupFunc()
			defer cleanup()

			// Create a modelfile instance and try to generate by workspace
			config := &configmodelfile.GenerateConfig{}
			_, err := NewModelfileByWorkspace(workspace, config)

			if tt.expectedError == "" {
				assert.NoError(t, err, "Expected no error for test case: %s", tt.name)
			} else {
				assert.Error(t, err, "Expected error for test case: %s", tt.name)
				assert.Contains(t, err.Error(), tt.expectedError, "Error message should contain expected text for test case: %s", tt.name)
			}
		})
	}
}

func TestDefaultBranchUntestedFlag(t *testing.T) {
	// Create a temporary directory
	tempDir := t.TempDir()

	// Create files that will fall into the default branch
	// Use file extensions that don't match any known patterns
	unknownLargeFile := filepath.Join(tempDir, "unknown_large.unknown")
	unknownSmallFile := filepath.Join(tempDir, "unknown_small.unknown")

	// Create a large file (>128MB) that should go to model
	largeContent := make([]byte, 129*1024*1024) // 129MB
	err := os.WriteFile(unknownLargeFile, largeContent, 0644)
	assert.NoError(t, err)

	// Create a small file that should go to code
	smallContent := []byte("some unknown file content")
	err = os.WriteFile(unknownSmallFile, smallContent, 0644)
	assert.NoError(t, err)

	// Generate modelfile from workspace
	config := &configmodelfile.GenerateConfig{
		Name: "test-untested-flags",
	}

	mf, err := NewModelfileByWorkspace(tempDir, config)
	assert.NoError(t, err)

	// Cast to concrete type to access flags
	modelfile := mf.(*modelfile)

	// Check that the large file was added to models with the untested flag
	assert.Contains(t, mf.GetModels(), "unknown_large.unknown")
	modelFlags := modelfile.GetModelFlags()
	assert.Contains(t, modelFlags, "unknown_large.unknown")
	assert.Equal(t, "true", modelFlags["unknown_large.unknown"][modelspec.AnnotationMediaTypeUntested])

	// Check that the small file was added to codes with the untested flag
	assert.Contains(t, mf.GetCodes(), "unknown_small.unknown")
	codeFlags := modelfile.GetCodeFlags()
	assert.Contains(t, codeFlags, "unknown_small.unknown")
	assert.Equal(t, "true", codeFlags["unknown_small.unknown"][modelspec.AnnotationMediaTypeUntested])

	// Check that the generated content includes the flags
	content := string(mf.Content())
	assert.Contains(t, content, fmt.Sprintf("MODEL --label=%s=true unknown_large.unknown", modelspec.AnnotationMediaTypeUntested))
	assert.Contains(t, content, fmt.Sprintf("CODE --label=%s=true unknown_small.unknown", modelspec.AnnotationMediaTypeUntested))
}
