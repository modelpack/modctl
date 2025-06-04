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
	ConfigFilePatterns = []string{
		"*.json",       // JSON configuration files
		"*.jsonl",      // JSON Lines format
		"*.yaml",       // YAML configuration files
		"*.yml",        // YAML alternative extension
		"*.toml",       // TOML configuration files
		"*.ini",        // INI configuration files
		"*.config",     // Generic config files
		"*.cfg",        // Generic config files
		"*.conf",       // Generic config files
		"*.properties", // Generic config files
		"*.props",      // Generic config files
		"*.prop",       // Generic config files
		"*.xml",        // XML configuration files
		"*.xsd",        // XML Schema Definition
		"*.rng",        // XML Schema Relax NG

		// Model-specific config files.
		"*.modelcard",       // Model card metadata
		"*.meta",            // Model metadata
		"*tokenizer.model*", // Tokenizer files (e.g., Mistral v3)
		"config.json.*",     // Model configuration variants
	}

	// Model file patterns - supported model file extensions.
	ModelFilePatterns = []string{
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
		"*.ot",         // OpenVINO format
		"*.engine",     // TensorRT format
		"*.trt",        // TensorRT format (alternative extension)
		"*.onnx",       // Open Neural Network Exchange format
		"*.gguf",       // GGML Universal Format
		"*.msgpack",    // MessagePack serialization
		"*.model",      // Some NLP frameworks
		"*.pkl",        // Pickle format
		"*.pickle",     // Pickle format (alternative extension)
		"*.ckpt",       // Checkpoint format
		"*.checkpoint", // Checkpoint format (alternative extension)
	}

	// Code file patterns - supported script and notebook files.
	CodeFilePatterns = []string{
		"*.py",     // Python source files
		"*.ipynb",  // Jupyter notebooks
		"*.sh",     // Shell scripts
		"*.patch",  // Patch files
		"*.c",      // C source files
		"*.h",      // C header files
		"*.hxx",    // C++ header files
		"*.cpp",    // C++ source files
		"*.cc",     // C++ source files
		"*.hpp",    // C++ header files
		"*.hh",     // C++ header files
		"*.java",   // Java source files
		"*.js",     // JavaScript source files
		"*.ts",     // TypeScript source files
		"*.go",     // Go source files
		"*.rs",     // Rust source files
		"*.swift",  // Swift source files
		"*.rb",     // Ruby source files
		"*.php",    // PHP source files
		"*.scala",  // Scala source files
		"*.kt",     // Kotlin source files
		"*.r",      // R source files
		"*.m",      // MATLAB/Objective-C source files
		"*.f",      // Fortran source files
		"*.f90",    // Fortran 90 source files
		"*.jl",     // Julia source files
		"*.lua",    // Lua source files
		"*.pl",     // Perl source files
		"*.cs",     // C# source files
		"*.vb",     // Visual Basic source files
		"*.dart",   // Dart source files
		"*.groovy", // Groovy source files
		"*.elm",    // Elm source files
		"*.erl",    // Erlang source files
		"*.ex",     // Elixir source files
		"*.hs",     // Haskell source files
		"*.clj",    // Clojure source files
		"*.cljs",   // ClojureScript source files
		"*.cljc",   // Clojure Common Lisp source files
		"*.cl",     // Common Lisp source files
		"*.lisp",   // Lisp source files
		"*.scm",    // Scheme source files
		"*.cu",     // CUDA source files
		"*.cuh",    // CUDA header files

		// Library files.
		"*.so",    // Shared object files
		"*.dll",   // Dynamic Link Library
		"*.dylib", // Dynamic Library
		"*.lib",   // Library files
		"*.a",     // Static Library
	}

	// Doc file patterns - supported documentation files
	DocFilePatterns = []string{
		// Documentation files.
		"*.txt",          // Text files
		"*.md",           // Markdown documentation
		"*.pdf",          // PDF files
		"LICENSE*",       // License files
		"README*",        // Project documentation
		"SETUP*",         // Setup instructions
		"*requirements*", // Dependency specifications
		"*.log",          // Log files

		// Image assets.
		"*.jpg",  // JPEG image format
		"*.jpeg", // JPEG alternative extension
		"*.png",  // PNG image format
		"*.gif",  // GIF image format
		"*.bmp",  // Bitmap image format
		"*.tiff", // TIFF image format
		"*.ico",  // Icon format
		"*.webp", // WebP image format
		"*.heic", // HEIC image format
		"*.heif", // HEIF image format
		"*.hevc", // HEVC image format
		"*.svg",  // SVG image format

		// Video assets.
		"*.mp4",  // MPEG-4 video format
		"*.mov",  // QuickTime video format
		"*.avi",  // AVI video format
		"*.mkv",  // Matroska video format
		"*.webm", // WebM video format
		"*.m4v",  // MPEG-4 video format
		"*.flv",  // Flash Video format
		"*.wmv",  // Windows Media Video format
		"*.mpg",  // MPEG-1 video format
		"*.mpeg", // MPEG-2 video format
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

// IsFileType checks if the filename matches any of the given patterns
func IsFileType(filename string, patterns []string) bool {
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
