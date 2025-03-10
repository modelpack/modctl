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
	"os"
	"path/filepath"
	"sort"
	"testing"

	configmodelfile "github.com/CloudNativeAI/modctl/pkg/config/modelfile"
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
config config1
model model1
code code1
dataset dataset1
doc doc1
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
config        config1
model         model1
code          code1
dataset       dataset1
doc           doc1
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
config config1
config config2
model model1
model model2
code code1
code code2
dataset dataset1
dataset dataset2
doc doc1
doc doc2
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
doc doc1
doc doc1
doc doc2
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
		name                       string
		setupFiles                 map[string]string
		setupDirs                  []string
		configJson                 map[string]interface{}
		genConfigJson              map[string]interface{}
		config                     *configmodelfile.GenerateConfig
		ignoreUnrecognizedFileType bool
		expectError                bool
		expectConfigs              []string
		expectModels               []string
		expectCodes                []string
		expectDocs                 []string
		expectName                 string
		expectArch                 string
		expectFamily               string
		expectFormat               string
		expectParamsize            string
		expectPrecision            string
		expectQuantization         string
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
			ignoreUnrecognizedFileType: false,
			expectError:                false,
			expectConfigs:              []string{"config.json"},
			expectModels:               []string{"model.bin"},
			expectCodes:                []string{"model.py", "tokenizer.py"},
			expectDocs:                 []string{"README.md", "LICENSE"},
			expectName:                 "test-model",
		},
		{
			name:       "empty workspace",
			setupFiles: map[string]string{},
			config: &configmodelfile.GenerateConfig{
				Name: "empty-model",
			},
			ignoreUnrecognizedFileType: false,
			expectError:                true,
			expectConfigs:              []string{},
			expectModels:               []string{},
			expectCodes:                []string{},
			expectName:                 "empty-model",
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
			ignoreUnrecognizedFileType: false,
			expectError:                false,
			expectConfigs:              []string{"config.json"},
			expectModels:               []string{"model.bin"},
			expectCodes:                []string{},
			expectName:                 "config-model",
			expectArch:                 "transformer",
			expectFamily:               "llama",
			expectPrecision:            "float16",
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
			ignoreUnrecognizedFileType: false,
			expectError:                false,
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
			ignoreUnrecognizedFileType: false,
			expectError:                false,
			expectConfigs:              []string{"level1/config.json"},
			expectModels:               []string{"level1/level2/level3/model.bin"},
			expectCodes:                []string{"level1/level2/level3/level4/code.py"},
			expectName:                 "deep-nested",
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
			ignoreUnrecognizedFileType: false,
			expectError:                false,
			expectConfigs:              []string{"config.json"},
			expectModels:               []string{"model.bin"},
			expectCodes:                []string{},
			expectName:                 "hidden-test",
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
			ignoreUnrecognizedFileType: false,
			expectError:                false,
			expectConfigs:              []string{"config.json", "models/config.json", "models/gen_config.json"},
			expectModels:               []string{"model.bin"},
			expectCodes:                []string{},
			expectName:                 "multi-config",
			expectArch:                 "transformer",
			expectFamily:               "gpt2",
			expectFormat:               "pytorch",
			expectPrecision:            "float32",
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
			ignoreUnrecognizedFileType: false,
			expectError:                false,
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
			ignoreUnrecognizedFileType: false,
			expectError:                false,
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
			ignoreUnrecognizedFileType: false,
			expectError:                false,
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
			ignoreUnrecognizedFileType: false,
			expectError:                false,
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
			ignoreUnrecognizedFileType: false,
			expectError:                false,
			expectConfigs:              []string{"config.json", "generation_config.json"},
			expectModels:               []string{"model.bin"},
			expectCodes:                []string{},
			expectName:                 "conflict-test",
			expectFamily:               "llama",
			expectPrecision:            "float32",
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
			ignoreUnrecognizedFileType: false,
			expectError:                false,
			expectConfigs:              []string{"config.json"},
			expectModels:               []string{"normal/model.bin"},
			expectCodes:                []string{"valid_dir/model.py"},
			expectName:                 "skip-test",
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
			tc.config.IgnoreUnrecognizedFileTypes = tc.ignoreUnrecognizedFileType

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
				"name test-model",
				"# Model architecture",
				"arch transformer",
				"# Model family",
				"family llama",
				"# Model format",
				"format safetensors",
				"# Model paramsize",
				"paramsize 7B",
				"# Model precision",
				"precision float16",
				"# Model quantization",
				"quantization int8",
				"# Config files",
				"config config.json",
				"# Documentation files",
				"doc README.md",
				"# Code files",
				"code convert.py",
				"code inference.py",
				"# Model files",
				"model model.bin",
				"model model.safetensors",
			},
			notExpectParts: []string{
				"dataset",
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
				"name minimal",
			},
			notExpectParts: []string{
				"arch", "family", "format", "paramsize", "precision", "quantization",
				"config", "code", "model", "dataset", "doc",
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
				"name tiny-gpt",
				"# Model architecture",
				"arch transformer",
				"# Model family",
				"family gpt2",
				"# Model format",
				"format pytorch",
				"# Model paramsize",
				"paramsize 125M",
				"# Model precision",
				"precision float32",
				"# Config files",
				"config config.json",
				"# Model files",
				"model pytorch_model.bin",
			},
			notExpectParts: []string{
				"quantization", "code", "dataset", "doc",
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
				"name llama-large",
				"# Model architecture",
				"arch transformer",
				"# Model family",
				"family llama",
				"# Model format",
				"format safetensors",
				"# Model paramsize",
				"paramsize 13B",
				"# Model precision",
				"precision bfloat16",
				"# Config files",
				"config config.json",
				"# Model files",
				"model model-00001-of-00003.safetensors",
				"model model-00002-of-00003.safetensors",
				"model model-00003-of-00003.safetensors",
			},
			notExpectParts: []string{
				"quantization", "code", "dataset", "doc",
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
				"name mistral-quantized",
				"# Model architecture",
				"arch transformer",
				"# Model family",
				"family mistral",
				"# Model format",
				"format gguf",
				"# Model paramsize",
				"paramsize 7B",
				"# Model quantization",
				"quantization Q4_K_M",
				"# Config files",
				"config config.json",
				"# Model files",
				"model model.gguf",
			},
			notExpectParts: []string{
				"precision", "code", "dataset", "doc",
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
				"name mega-model",
				"# Model architecture",
				"arch moe",
				"# Model family",
				"family mixtral",
				"# Model format",
				"format pytorch",
				"# Model paramsize",
				"paramsize 1.5T",
				"# Model precision",
				"precision float16",
				"# Config files",
				"config config.json",
				"# Model files",
				"model shard-00001.bin",
				"model shard-00002.bin",
			},
			notExpectParts: []string{
				"quantization", "code", "dataset", "doc",
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
				"name nested-paths-model",
				"paramsize 3B",
				"# Config files",
				"config configs/main.json",
				"config configs/tokenizer/config.json",
				"# Model files",
				"model models/weights/pytorch_model.bin",
				"# Code files",
				"code src/utils.py",
				"code src/models/model.py",
			},
			notExpectParts: []string{
				"arch", "family", "format", "precision", "quantization", "dataset", "doc",
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
				"name fractional-size",
				"paramsize 2.7B",
				"# Config files",
				"config config.json",
				"# Model files",
				"model model.bin",
			},
			notExpectParts: []string{
				"arch", "family", "format", "precision", "quantization", "code", "dataset", "doc",
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
				"name metadata-only",
				"# Model architecture",
				"arch transformer",
				"# Model family",
				"family gpt-neox",
				"# Model format",
				"format pytorch",
				"# Model paramsize",
				"paramsize 20B",
				"# Model precision",
				"precision bfloat16",
				"# Model quantization",
				"quantization int4",
			},
			notExpectParts: []string{
				"code", "doc", "dataset",
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
				"name files-only",
				"# Config files",
				"config config.json",
				"# Model files",
				"model model.bin",
				"# Code files",
				"code script.py",
				"# Documentation files",
				"doc README.md",
			},
			notExpectParts: []string{
				"arch", "family", "format", "paramsize", "precision", "quantization", "dataset",
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
				"name multi-file",
				"# Model paramsize",
				"paramsize 7B",
				"# Config files",
				"config config1.json",
				"config config2.json",
				"config config3.json",
				"# Model files",
				"model model1.bin",
				"model model2.bin",
				"model model3.bin",
				"model model4.bin",
				"# Code files",
				"code script1.py",
				"code script2.py",
				"# Documentation files",
				"doc README1.md",
				"doc README2.md",
			},
			notExpectParts: []string{
				"aarch", "family", "format", "precision", "quantization", "dataset",
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
				"name special-chars",
				"# Model paramsize",
				"paramsize 1B",
				"# Config files",
				"config spaces.json",
				"config weird-name!.yaml",
				"# Model files",
				"model model-v1.0_beta.bin",
				"# Code files",
				"code spaces/script.py",
				"# Documentation files",
				"doc weird-name!.md",
			},
			notExpectParts: []string{
				"arch", "family", "format", "precision", "quantization", "dataset",
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
