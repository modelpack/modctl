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

package modelfile

import (
	"path/filepath"
	"strings"
)

var (
	// Config file patterns - supported configuration file extensions.
	configFilePatterns = []string{
		"*.json",      // JSON configuration files
		"*.jsonl",     // JSON Lines format
		"*.yaml",      // YAML configuration files
		"*.yml",       // YAML alternative extension
		"*.toml",      // TOML configuration files
		"*.ini",       // INI configuration files
		"*.config",    // Generic config files
		"*.modelcard", // Model card metadata
		"*.meta",      // Model metadata

		// Model-specific files.
		"*tokenizer.model*", // Tokenizer files (e.g., Mistral v3)
		"config.json.*",     // Model configuration variants
	}

	// Model file patterns - supported model file extensions.
	modelFilePatterns = []string{
		// Huggingface formats.
		"*.safetensors", // Safe and efficient tensor serialization format

		// PyTorch formats.
		"*.bin", // General binary format
		"*.pt",  // PyTorch model
		"*.pth", // PyTorch model (alternative extension)

		// TensorFlow formats.
		"*.tflite", // TensorFlow Lite
		"*.h5",     // Keras HDF5 format
		"*.hdf",    // Hierarchical Data Format
		"*.hdf5",   // HDF5 (alternative extension)

		// Other ML frameworks.
		"*.ot",      // OpenVINO format
		"*.engine",  // TensorRT format
		"*.trt",     // TensorRT format (alternative extension)
		"*.onnx",    // Open Neural Network Exchange format
		"*.gguf",    // GGML Universal Format
		"*.msgpack", // MessagePack serialization
		"*.model",   // Some NLP frameworks
	}

	// Code file patterns - supported script and notebook files.
	codeFilePatterns = []string{
		"*.py",    // Python source files
		"*.sh",    // Shell scripts
		"*.ipynb", // Jupyter notebooks
		"*.patch", // Patch files
	}

	// Doc file patterns - supported documentation files
	docFilePatterns = []string{
		// Documentation files.
		"*.txt",          // Text files
		"*.md",           // Markdown documentation
		"*.pdf",          // PDF files
		"LICENSE*",       // License files
		"README*",        // Project documentation
		"SETUP*",         // Setup instructions
		"*requirements*", // Dependency specifications

		// Image assets.
		"*.jpg",  // JPEG image format
		"*.jpeg", // JPEG alternative extension
		"*.png",  // PNG image format
		"*.gif",  // GIF image format
		"*.bmp",  // Bitmap image format
		"*.tiff", // TIFF image format
		"*.ico",  // Icon format
	}

	// Skip patterns - files and directories to ignore during processing.
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
