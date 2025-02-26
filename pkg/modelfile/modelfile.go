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
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	modefilecommand "github.com/CloudNativeAI/modctl/pkg/modelfile/command"
	"github.com/CloudNativeAI/modctl/pkg/modelfile/parser"

	"github.com/emirpasic/gods/sets/hashset"
)

// Modelfile is the interface for the modelfile. It is used to parse
// the modelfile by the path and get the information of the modelfile.
type Modelfile interface {
	// GetConfigs returns the args of the config command in the modelfile,
	// and deduplicates the args. The order of the args is the same as the
	// order in the modelfile.
	GetConfigs() []string

	// GetModels returns the args of the model command in the modelfile,
	// and deduplicates the args. The order of the args is the same as the
	// order in the modelfile.
	GetModels() []string

	// GetCode returns the args of the code command in the modelfile,
	// and deduplicates the args. The order of the args is the same as the
	// order in the modelfile.
	GetCodes() []string

	// GetDatasets returns the args of the dataset command in the modelfile,
	// and deduplicates the args. The order of the args is the same as the
	// order in the modelfile.
	GetDatasets() []string

	// GetName returns the value of the name command in the modelfile.
	GetName() string

	// GetArch returns the value of the arch command in the modelfile.
	GetArch() string

	// GetFamily returns the value of the family command in the modelfile.
	GetFamily() string

	// GetFormat returns the value of the format command in the modelfile.
	GetFormat() string

	// GetParamsize returns the value of the paramsize command in the modelfile.
	GetParamsize() string

	// GetPrecision returns the value of the precision command in the modelfile.
	GetPrecision() string

	// GetQuantization returns the value of the quantization command in the modelfile.
	GetQuantization() string

	// SaveToFile saves the modelfile to the file.
	SaveToFile(path string) error
}

// modelfile is the implementation of the Modelfile interface.
type modelfile struct {
	config       *hashset.Set
	model        *hashset.Set
	code         *hashset.Set
	dataset      *hashset.Set
	name         string
	arch         string
	family       string
	format       string
	paramsize    string
	precision    string
	quantization string
}

// Config file patterns - supported configuration and resource files
var (
	configFilePatterns = []string{
		// Configuration formats
		"*.json",      // JSON configuration files
		"*.jsonl",     // JSON Lines format
		"*.yaml",      // YAML configuration files
		"*.yml",       // YAML alternative extension
		"*.toml",      // TOML configuration files
		"*.ini",       // INI configuration files
		"*.config",    // Generic config files
		"*.txt",       // Text files
		"*.modelcard", // Model card metadata
		"*.meta",      // Model metadata

		// Documentation files
		"*.md",           // Markdown documentation
		"LICENSE*",       // License files
		"README*",        // Project documentation
		"SETUP*",         // Setup instructions
		"*requirements*", // Dependency specifications

		// Image assets
		"*.jpg",  // JPEG image format
		"*.jpeg", // JPEG alternative extension
		"*.png",  // PNG image format
		"*.gif",  // GIF image format
		"*.bmp",  // Bitmap image format
		"*.tiff", // TIFF image format
		"*.ico",  // Icon format

		// Model-specific files
		"*tokenizer.model*", // Tokenizer files (e.g., Mistral v3)
		"config.json.*",     // Model configuration variants
	}

	// Model file patterns - supported model file extensions
	modelFilePatterns = []string{
		// Huggingface formats
		"*.safetensors", // Safe and efficient tensor serialization format

		// PyTorch formats
		"*.bin", // General binary format
		"*.pt",  // PyTorch model
		"*.pth", // PyTorch model (alternative extension)

		// TensorFlow formats
		"*.tflite", // TensorFlow Lite
		"*.h5",     // Keras HDF5 format
		"*.hdf",    // Hierarchical Data Format
		"*.hdf5",   // HDF5 (alternative extension)

		// Other ML frameworks
		"*.ot",      // OpenVINO format
		"*.engine",  // TensorRT format
		"*.trt",     // TensorRT format (alternative extension)
		"*.onnx",    // Open Neural Network Exchange format
		"*.gguf",    // GGML Universal Format
		"*.msgpack", // MessagePack serialization
		"*.model",   // Some NLP frameworks
	}

	// Code file patterns - supported script and notebook files
	codeFilePatterns = []string{
		"*.py",    // Python source files
		"*.sh",    // Shell scripts
		"*.ipynb", // Jupyter notebooks
	}

	// Skip patterns - files and directories to ignore during processing
	skipPatterns = []string{
		".*",          // Hidden files and directories
		"modelfile",   // Modelfile configuration
		"__pycache__", // Python bytecode cache directory
		"*.pyc",       // Python compiled bytecode
		"*.pyo",       // Python optimized bytecode
		"*.pyd",       // Python dynamic modules
	}
)

// isFileType checks if the filename matches any of the given patterns
func isFileType(filename string, patterns []string) bool {
	// Convert filename to lowercase for case-insensitive comparison
	lowerFilename := strings.ToLower(filename)
	for _, pattern := range patterns {
		// Convert pattern to lowercase for case-insensitive comparison
		matched, err := filepath.Match(strings.ToLower(pattern), lowerFilename)
		if err == nil && matched {
			return true
		}
	}
	return false
}

// isSkippable checks if the filename matches any of the skip patterns
func isSkippable(filename string) bool {
	// Special handling for current and parent directory
	if filename == "." || filename == ".." {
		return false
	}

	// Convert filename to lowercase for case-insensitive comparison
	lowerFilename := strings.ToLower(filename)
	for _, pattern := range skipPatterns {
		// Convert pattern to lowercase for case-insensitive comparison
		matched, err := filepath.Match(strings.ToLower(pattern), lowerFilename)
		if err == nil && matched {
			return true
		}
	}
	return false
}

// cleanModelName sanitizes a string to create a valid model name
func cleanModelName(name string) string {
	// Remove any trailing slashes first
	name = strings.TrimRight(name, "/\\")

	// Replace invalid characters with underscores
	name = strings.Map(func(r rune) rune {
		// Allow alphanumeric characters
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			return r
		}
		// Replace everything else with underscore
		return '_'
	}, name)

	// Remove leading/trailing underscores
	name = strings.Trim(name, "_")

	// If name is empty after cleaning, return a default
	if name == "" {
		return "unnamed_model"
	}

	return name
}

// NewModelfile creates a new modelfile by the path of the modelfile.
// It parses the modelfile and returns the modelfile interface.
func NewModelfile(path string) (Modelfile, error) {
	mf := &modelfile{
		config:  hashset.New(),
		model:   hashset.New(),
		code:    hashset.New(),
		dataset: hashset.New(),
	}
	if err := mf.parseFile(path); err != nil {
		return nil, err
	}

	return mf, nil
}

// Parsing the model config file and update the parameters, currently only
// support the huggingface tranformers library. Considering to use library
// directly.
func parseModelConfig(path string, modelFile *modelfile) error {
	// Get config map from json files
	modelConfig := make(map[string]interface{})
	for _, file := range []string{"config.json", "generation_config.json"} {
		filename := filepath.Join(path, file)
		data, err := os.ReadFile(filename)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			} else {
				return err
			}
		}

		var config map[string]interface{}
		if err := json.Unmarshal(data, &config); err == nil {
			for k, v := range config {
				modelConfig[k] = v
			}
		}
	}

	// get precision
	if _, ok := modelConfig["torch_dtype"]; ok {
		modelFile.precision = modelConfig["torch_dtype"].(string)
	}

	// get family
	if _, ok := modelConfig["model_type"]; ok {
		modelFile.family = modelConfig["model_type"].(string)
	}

	// get architecture
	if _, ok := modelConfig["transformers_version"]; ok {
		modelFile.arch = "transformer"
	}

	return nil
}

// overwriteModelConfig overwrites the modelfile configurations with the provided config values
func overwriteModelConfig(mf *modelfile, config *ModelfileGenConfig) {
	if config == nil {
		return
	}

	// Name is handled separately in AutoModelfile
	if config.Arch != "" {
		mf.arch = config.Arch
	}
	if config.Family != "" {
		mf.family = config.Family
	}
	if config.Format != "" {
		mf.format = config.Format
	}
	if config.Paramsize != "" {
		mf.paramsize = config.Paramsize
	}
	if config.Precision != "" {
		mf.precision = config.Precision
	}
	if config.Quantization != "" {
		mf.quantization = config.Quantization
	}
}

// AutoModelfile creates a new modelfile by the path of the model directory.
// It walks the directory and returns the auto-generated modelfile interface.
func AutoModelfile(path string, config *ModelfileGenConfig) (Modelfile, error) {
	// check if the config is nil
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	mf := &modelfile{
		config:  hashset.New(),
		model:   hashset.New(),
		code:    hashset.New(),
		dataset: hashset.New(),
	}

	// use directory name as model name if config.name is empty
	if config.Name == "" {
		mf.name = cleanModelName(filepath.Base(path))
	} else {
		mf.name = config.Name
	}

	// walk the path and get the files
	err := filepath.Walk(path, func(fullPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		filename := info.Name()

		// skip hidden and skippable files/directories
		if isSkippable(filename) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if info.IsDir() {
			return nil
		}

		// get relative path from the base directory
		relPath, err := filepath.Rel(path, fullPath)
		if err != nil {
			return err
		}

		switch {
		case isFileType(filename, configFilePatterns):
			mf.config.Add(relPath)
		case isFileType(filename, modelFilePatterns):
			mf.model.Add(relPath)
		case isFileType(filename, codeFilePatterns):
			mf.code.Add(relPath)
		default:
			// skip unrecognized files if IgnoreUnrecognized is true
			if config.IgnoreUnrecognized {
				return nil
			}
			return fmt.Errorf("unknown file type: %s - use --ignore-unrecognized to ignore, and edit the Modelfile manually", filename)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// check if model files are found
	if mf.model.Size() == 0 {
		return nil, fmt.Errorf("no recognized model files found in directory - you may need to edit the Modelfile manually")
	}

	// get the model config from the config.json file
	if err := parseModelConfig(path, mf); err != nil {
		return nil, err
	}

	// overwrite the modelfile configurations with the provided config values
	overwriteModelConfig(mf, config)

	return mf, nil
}

// parseFile parses the modelfile by the path, and validates the args of the commands.
func (mf *modelfile) parseFile(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	ast, err := parser.Parse(f)
	if err != nil {
		return err
	}

	for _, child := range ast.GetChildren() {
		switch child.GetValue() {
		case modefilecommand.CONFIG:
			mf.config.Add(child.GetNext().GetValue())
		case modefilecommand.MODEL:
			mf.model.Add(child.GetNext().GetValue())
		case modefilecommand.CODE:
			mf.code.Add(child.GetNext().GetValue())
		case modefilecommand.DATASET:
			mf.dataset.Add(child.GetNext().GetValue())
		case modefilecommand.NAME:
			if mf.name != "" {
				return fmt.Errorf("duplicate name command on line %d", child.GetStartLine())
			}
			mf.name = child.GetNext().GetValue()
		case modefilecommand.ARCH:
			if mf.arch != "" {
				return fmt.Errorf("duplicate arc command on line %d", child.GetStartLine())
			}
			mf.arch = child.GetNext().GetValue()
		case modefilecommand.FAMILY:
			if mf.family != "" {
				return fmt.Errorf("duplicate family command on line %d", child.GetStartLine())
			}
			mf.family = child.GetNext().GetValue()
		case modefilecommand.FORMAT:
			if mf.format != "" {
				return fmt.Errorf("duplicate format command on line %d", child.GetStartLine())
			}
			mf.format = child.GetNext().GetValue()
		case modefilecommand.PARAMSIZE:
			if mf.paramsize != "" {
				return fmt.Errorf("duplicate paramsize command on line %d", child.GetStartLine())
			}
			mf.paramsize = child.GetNext().GetValue()
		case modefilecommand.PRECISION:
			if mf.precision != "" {
				return fmt.Errorf("duplicate precision command on line %d", child.GetStartLine())
			}
			mf.precision = child.GetNext().GetValue()
		case modefilecommand.QUANTIZATION:
			if mf.quantization != "" {
				return fmt.Errorf("duplicate quantization command on line %d", child.GetStartLine())
			}
			mf.quantization = child.GetNext().GetValue()
		default:
			return fmt.Errorf("unknown command %s on line %d", child.GetValue(), child.GetStartLine())
		}
	}

	return nil
}

// GetConfigs returns the args of the config command in the modelfile,
// and deduplicates the args. The order of the args is the same as the
// order in the modelfile.
func (mf *modelfile) GetConfigs() []string {
	var configs []string
	for _, rawConfig := range mf.config.Values() {
		config, ok := rawConfig.(string)
		if !ok {
			continue
		}

		configs = append(configs, config)
	}

	return configs
}

// GetModels returns the args of the model command in the modelfile,
// and deduplicates the args. The order of the args is the same as the
// order in the modelfile.
func (mf *modelfile) GetModels() []string {
	var models []string
	for _, rawModel := range mf.model.Values() {
		model, ok := rawModel.(string)
		if !ok {
			continue
		}

		models = append(models, model)
	}

	return models
}

// GetCode returns the args of the code command in the modelfile,
// and deduplicates the args. The order of the args is the same as the
// order in the modelfile.
func (mf *modelfile) GetCodes() []string {
	var codes []string
	for _, rawCode := range mf.code.Values() {
		code, ok := rawCode.(string)
		if !ok {
			continue
		}

		codes = append(codes, code)
	}

	return codes
}

// GetDatasets returns the args of the dataset command in the modelfile,
// and deduplicates the args. The order of the args is the same as the
// order in the modelfile.
func (mf *modelfile) GetDatasets() []string {
	var datasets []string
	for _, rawDataset := range mf.dataset.Values() {
		dataset, ok := rawDataset.(string)
		if !ok {
			continue
		}

		datasets = append(datasets, dataset)
	}

	return datasets
}

// GetName returns the value of the name command in the modelfile.
func (mf *modelfile) GetName() string {
	return mf.name
}

// GetArch returns the value of the arch command in the modelfile.
func (mf *modelfile) GetArch() string {
	return mf.arch
}

// GetFamily returns the value of the family command in the modelfile.
func (mf *modelfile) GetFamily() string {
	return mf.family
}

// GetFormat returns the value of the format command in the modelfile.
func (mf *modelfile) GetFormat() string {
	return mf.format
}

// GetParamsize returns the value of the paramsize command in the modelfile.
func (mf *modelfile) GetParamsize() string {
	return mf.paramsize
}

// GetPrecision returns the value of the precision command in the modelfile.
func (mf *modelfile) GetPrecision() string {
	return mf.precision
}

// GetQuantization returns the value of the quantization command in the modelfile.
func (mf *modelfile) GetQuantization() string {
	return mf.quantization
}

// SaveToFile saves the modelfile content to the specified path
func (mf *modelfile) SaveToFile(path string) error {
	content := ""

	// generate time in the first line
	content += fmt.Sprintf("# Generated at %s\n", time.Now().Format(time.RFC3339))

	// add single value commands
	if mf.name != "" {
		content += "\n# Model name\n"
		content += fmt.Sprintf("NAME %s\n", mf.name)
	}
	if mf.arch != "" {
		content += "\n# Model architecture (Generated from \"transformers_version\" in config.json)\n"
		content += fmt.Sprintf("ARCH %s\n", mf.arch)
	}
	if mf.family != "" {
		content += "\n# Model family (Generated from \"model_type\" in config.json)\n"
		content += fmt.Sprintf("FAMILY %s\n", mf.family)
	}
	if mf.format != "" {
		content += "\n# Model format\n"
		content += fmt.Sprintf("FORMAT %s\n", mf.format)
	}
	if mf.paramsize != "" {
		content += "\n# Model paramsize\n"
		content += fmt.Sprintf("PARAMSIZE %s\n", mf.paramsize)
	}
	if mf.precision != "" {
		content += "\n# Model precision (Generated from \"torch_dtype\" in config.json)\n"
		content += fmt.Sprintf("PRECISION %s\n", mf.precision)
	}
	if mf.quantization != "" {
		content += "\n# Model quantization\n"
		content += fmt.Sprintf("QUANTIZATION %s\n", mf.quantization)
	}

	// add multi-value commands
	content += "\n# Config files (Generated from the files in the model directory)\n"
	content += "# Supported file types: " + strings.Join(configFilePatterns, ", ") + "\n"
	configs := mf.GetConfigs()
	sort.Strings(configs)
	for _, config := range configs {
		content += fmt.Sprintf("CONFIG %s\n", config)
	}

	content += "\n# Code files (Generated from the files in the model directory)\n"
	content += "# Supported file types: " + strings.Join(codeFilePatterns, ", ") + "\n"
	codes := mf.GetCodes()
	sort.Strings(codes)
	for _, code := range codes {
		content += fmt.Sprintf("CODE %s\n", code)
	}

	content += "\n# Model files (Generated from the files in the model directory)\n"
	content += "# Supported file types: " + strings.Join(modelFilePatterns, ", ") + "\n"
	models := mf.GetModels()
	sort.Strings(models)
	for _, model := range models {
		content += fmt.Sprintf("MODEL %s\n", model)
	}

	// write to file
	return os.WriteFile(path, []byte(content), 0644)
}
