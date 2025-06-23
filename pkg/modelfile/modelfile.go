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

	configmodelfile "github.com/CloudNativeAI/modctl/pkg/config/modelfile"
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

	// GetDocs returns the args of the doc command in the modelfile,
	// and deduplicates the args. The order of the args is the same as the
	// order in the modelfile.
	GetDocs() []string

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

	// Content returns the content of the modelfile.
	Content() []byte
}

// modelfile is the implementation of the Modelfile interface.
type modelfile struct {
	workspace    string
	config       *hashset.Set
	model        *hashset.Set
	code         *hashset.Set
	dataset      *hashset.Set
	doc          *hashset.Set
	name         string
	arch         string
	family       string
	format       string
	paramsize    string
	precision    string
	quantization string
}

// NewModelfile creates a new modelfile by the path of the modelfile.
// It parses the modelfile and returns the modelfile interface.
func NewModelfile(path string) (Modelfile, error) {
	mf := &modelfile{
		config:  hashset.New(),
		model:   hashset.New(),
		code:    hashset.New(),
		dataset: hashset.New(),
		doc:     hashset.New(),
	}

	if err := mf.parseFile(path); err != nil {
		return nil, err
	}

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
		case modefilecommand.DOC:
			mf.doc.Add(child.GetNext().GetValue())
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

// NewModelfileByWorkspace creates a new modelfile by the workspace.
//
// It generates the modelfile by the following steps:
//  1. It walks the workspace and gets the files, and generates the modelfile by the files.
//  2. It generates the modelfile by the model config, such as config.json and generation_config.json.
//  3. It generates the modelfile by the generate config, such as name, arch, family, format,
//     paramsize, precision, and quantization.
func NewModelfileByWorkspace(workspace string, config *configmodelfile.GenerateConfig) (Modelfile, error) {
	mf := &modelfile{
		workspace: workspace,
		config:    hashset.New(),
		model:     hashset.New(),
		code:      hashset.New(),
		dataset:   hashset.New(),
		doc:       hashset.New(),
	}

	if err := mf.validateWorkspace(); err != nil {
		return nil, err
	}

	if err := mf.generateByWorkspace(); err != nil {
		return nil, err
	}

	if err := mf.generateByModelConfig(); err != nil {
		return nil, err
	}

	mf.generateByConfig(config)
	return mf, nil
}

// validateWorkspace validates the workspace directory
func (mf *modelfile) validateWorkspace() error {
	// check if the workspace is a directory, symbolic link, or empty
	info, err := os.Lstat(mf.workspace)
	if err != nil {
		return fmt.Errorf("access to workspace failed: %s", err)
	}

	// check if the workspace is a symbolic link
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("for simplicity, the workspace should not be a symbolic link: %s", mf.workspace)
	}

	// check if the workspace is a directory
	if !info.IsDir() {
		return fmt.Errorf("the workspace is not a directory: %s", mf.workspace)
	}

	// check if the workspace is empty by reading directory contents
	entries, err := os.ReadDir(mf.workspace)
	if err != nil {
		return fmt.Errorf("failed to read workspace directory: %s", err)
	}
	if len(entries) == 0 {
		return fmt.Errorf("the workspace is empty: %s", mf.workspace)
	}

	return nil
}

// generateByWorkspace generates the modelfile by the workspace's files.
func (mf *modelfile) generateByWorkspace() error {
	// Initialize counters for workspace limits validation
	var fileCount int
	var totalSize int64

	// Walk the path and get the files.
	if err := filepath.Walk(mf.workspace, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		filename := info.Name()

		// Skip hidden and skippable files/directories.
		if isSkippable(filename) {
			if info.IsDir() {
				return filepath.SkipDir
			}

			return nil
		}

		if info.IsDir() {
			return nil
		}

		// Check workspace limits for regular files
		fileCount++
		fileSize := info.Size()
		totalSize += fileSize

		// Check single file size limit
		if fileSize > MaxSingleFileSize {
			return fmt.Errorf("file %s exceeds maximum single file size limit of %d bytes (%s)", path, MaxSingleFileSize, formatBytes(MaxSingleFileSize))
		}

		// Check file count limit
		if fileCount > MaxWorkspaceFileCount {
			return fmt.Errorf("workspace exceeds maximum file count limit of %d files", MaxWorkspaceFileCount)
		}

		// Check total workspace size limit
		if totalSize > MaxTotalWorkspaceSize {
			return fmt.Errorf("workspace exceeds maximum total size limit of %d bytes (%s)", MaxTotalWorkspaceSize, formatBytes(MaxTotalWorkspaceSize))
		}

		// Get relative path from the base directory.
		relPath, err := filepath.Rel(mf.workspace, path)
		if err != nil {
			return err
		}

		switch {
		case IsFileType(filename, ConfigFilePatterns):
			mf.config.Add(relPath)
		case IsFileType(filename, ModelFilePatterns):
			mf.model.Add(relPath)
		case IsFileType(filename, CodeFilePatterns):
			mf.code.Add(relPath)
		case IsFileType(filename, DocFilePatterns):
			mf.doc.Add(relPath)
		default:
			// If the file is large, usually it is a weight file.
			if SizeShouldBeWeightFile(info.Size()) {
				mf.model.Add(relPath)
			} else {
				mf.code.Add(relPath)
			}

			return nil
		}

		return nil
	}); err != nil {
		return err
	}

	if mf.model.Size() == 0 && mf.code.Size() == 0 && mf.dataset.Size() == 0 {
		return fmt.Errorf("no model/code/dataset found - you have to create the Modelfile by yourself")
	}

	return nil
}

// generateByModelConfig generates the modelfile by the model config, such as config.json and generation_config.json.
func (mf *modelfile) generateByModelConfig() error {
	// Get config map from json files. Collect all the keys and values from the config files
	// and store them in the modelConfig map.
	configFiles := []string{"config.json", "generation_config.json"}
	modelConfig := make(map[string]interface{})
	for _, filename := range configFiles {
		path := filepath.Join(mf.workspace, filename)
		data, err := os.ReadFile(path)
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

	if torchDtype, ok := modelConfig["torch_dtype"].(string); ok {
		mf.precision = torchDtype
	}

	if modelType, ok := modelConfig["model_type"].(string); ok {
		mf.family = modelType
	}

	if _, ok := modelConfig["transformers_version"]; ok {
		mf.arch = "transformer"
	}

	return nil
}

// generateByConfig generates the modelfile by the generate config, such as name, arch, family, format,
// paramsize, precision, and quantization.
func (mf *modelfile) generateByConfig(config *configmodelfile.GenerateConfig) {
	if config.Name == "" {
		mf.name = filepath.Base(mf.workspace)
	} else {
		mf.name = config.Name
	}

	if config.Arch != "" {
		mf.arch = config.Arch
	}

	if config.Family != "" {
		mf.family = config.Family
	}

	if config.Format != "" {
		mf.format = config.Format
	}

	if config.ParamSize != "" {
		mf.paramsize = config.ParamSize
	}

	if config.Precision != "" {
		mf.precision = config.Precision
	}

	if config.Quantization != "" {
		mf.quantization = config.Quantization
	}
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

// GetDocs returns the args of the doc command in the modelfile,
// and deduplicates the args. The order of the args is the same as the
// order in the modelfile.
func (mf *modelfile) GetDocs() []string {
	var docs []string
	for _, rawDoc := range mf.doc.Values() {
		doc, ok := rawDoc.(string)
		if !ok {
			continue
		}

		docs = append(docs, doc)
	}

	return docs
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

// Content returns the content of the modelfile.
func (mf *modelfile) Content() []byte {
	content := ""
	content += fmt.Sprintf("# Generated at %s\n", time.Now().Format(time.RFC3339))

	// Add single-value commands.
	content += mf.writeField("Model name", modefilecommand.NAME, mf.name)
	content += mf.writeField("Model architecture (Generated from transformers_version in config.json)", modefilecommand.ARCH, mf.arch)
	content += mf.writeField("Model family (Generated from model_type in config.json)", modefilecommand.FAMILY, mf.family)
	content += mf.writeField("Model format", modefilecommand.FORMAT, mf.format)
	content += mf.writeField("Model paramsize", modefilecommand.PARAMSIZE, mf.paramsize)
	content += mf.writeField("Model precision (Generated from torch_dtype in config.json)", modefilecommand.PRECISION, mf.precision)
	content += mf.writeField("Model quantization", modefilecommand.QUANTIZATION, mf.quantization)

	// Add multi-value commands.
	content += mf.writeMultiField("Config files (Generated from the files in the workspace directory)", modefilecommand.CONFIG, mf.GetConfigs(), ConfigFilePatterns)
	content += mf.writeMultiField("Code files (Generated from the files in the workspace directory)", modefilecommand.CODE, mf.GetCodes(), CodeFilePatterns)
	content += mf.writeMultiField("Model files (Generated from the files in the workspace directory)", modefilecommand.MODEL, mf.GetModels(), ModelFilePatterns)
	content += mf.writeMultiField("Documentation files (Generated from the files in the workspace directory)", modefilecommand.DOC, mf.GetDocs(), DocFilePatterns)
	return []byte(content)
}

func (mf *modelfile) writeField(comment, cmd, value string) string {
	if value == "" {
		return ""
	}

	return fmt.Sprintf("\n# %s\n%s %s\n", comment, cmd, value)
}

func (mf *modelfile) writeMultiField(comment, cmd string, values []string, patterns []string) string {
	if len(values) == 0 {
		return ""
	}

	content := fmt.Sprintf("\n# %s\n", comment)
	content += fmt.Sprintf("# Supported file types: %s\n", strings.Join(patterns, ", "))

	sort.Strings(values)
	for _, value := range values {
		// Quote the value if it contains spaces or special characters
		quotedValue := mf.quoteIfNeeded(value)
		content += fmt.Sprintf("%s %s\n", cmd, quotedValue)
	}

	return content
}

// quoteIfNeeded adds quotes around a value if it contains spaces or special characters
func (mf *modelfile) quoteIfNeeded(value string) string {
	// Check if the value contains spaces or other characters that need quoting
	if strings.ContainsAny(value, " \t\n\r") {
		// Escape any existing quotes in the value
		escaped := strings.ReplaceAll(value, `"`, `\"`)
		return fmt.Sprintf(`"%s"`, escaped)
	}
	return value
}
