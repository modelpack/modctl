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
		"*.json5",      // JSON5 files
		"*.jsonc",      // JSON with comments
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
		"*.hparams",         // Hyperparameter files
		"*.params",          // Parameter files
		"*.hyperparams",     // Hyperparameter configuration
		"*.wandb",           // Weights & Biases configuration
		"*.mlflow",          // MLflow configuration
		"*.tensorboard",     // TensorBoard configuration
	}

	// Model file patterns - supported model file extensions.
	ModelFilePatterns = []string{
		// Huggingface formats.
		"*.safetensors", // Safe and efficient tensor serialization format

		// PyTorch formats.
		"*.bin", // General binary format
		"*.pt",  // PyTorch model
		"*.pth", // PyTorch model (alternative extension)
		"*.mar", // PyTorch Model Archive
		"*.pte", // PyTorch ExecuTorch format
		"*.pt2", // PyTorch 2.0 export format
		"*.ptl", // PyTorch Mobile format

		// TensorFlow formats.
		"*.tflite", // TensorFlow Lite
		"*.h5",     // Keras HDF5 format
		"*.hdf",    // Hierarchical Data Format
		"*.hdf5",   // HDF5 (alternative extension)
		"*.pb",     // TensorFlow SavedModel/Frozen Graph
		"*.meta",   // TensorFlow checkpoint metadata
		"*.data-*", // TensorFlow checkpoint data files
		"*.index",  // TensorFlow checkpoint index

		// GGML formats.
		"*.gguf", // GGML Universal Format
		"*.ggml", // GGML format (legacy)
		"*.ggmf", // GGMF format (deprecated)
		"*.ggjt", // GGJT format (deprecated)
		"*.q4_0", // GGML Q4_0 quantization
		"*.q4_1", // GGML Q4_1 quantization
		"*.q5_0", // GGML Q5_0 quantization
		"*.q5_1", // GGML Q5_1 quantization
		"*.q8_0", // GGML Q8_0 quantization
		"*.f16",  // GGML F16 format
		"*.f32",  // GGML F32 format

		// checkpoint formats.
		"*.ckpt",       // Checkpoint format
		"*.checkpoint", // Checkpoint format (alternative extension)
		"*.dist_ckpt",  // Distributed checkpoint format

		// Semantics-specific formats
		"*.tensor",    // Generic tensor format
		"*.weights",   // Generic weights format
		"*.state",     // State files
		"*.embedding", // Embedding files
		"*.vocab",     // Vocabulary files (when binary)

		// Other ML frameworks.
		"*.ot",         // OpenVINO format
		"*.engine",     // TensorRT format
		"*.trt",        // TensorRT format (alternative extension)
		"*.onnx",       // Open Neural Network Exchange format
		"*.msgpack",    // MessagePack serialization
		"*.model",      // Some NLP frameworks
		"*.pkl",        // Pickle format
		"*.pickle",     // Pickle format (alternative extension)
		"*.keras",      // Keras native format
		"*.joblib",     // Joblib serialization (scikit-learn)
		"*.npy",        // NumPy array format
		"*.npz",        // NumPy compressed archive
		"*.nc",         // NetCDF format
		"*.mlmodel",    // Apple Core ML format
		"*.coreml",     // Apple Core ML format (alternative)
		"*.mleap",      // MLeap format (Spark ML)
		"*.surml",      // SurrealML format
		"*.llamafile",  // Llamafile format
		"*.caffemodel", // Caffe model format
		"*.prototxt",   // Caffe model definition
		"*.dlc",        // Qualcomm Deep Learning Container
		"*.circle",     // Samsung Circle format
		"*.nb",         // Neural Network Binary format
	}

	// Code file patterns - supported script and notebook files.
	CodeFilePatterns = []string{
		// language source files
		"*.py",     // Python source files
		"*.ipynb",  // Jupyter notebooks
		"*.sh",     // Shell scripts
		"*.patch",  // Patch files
		"*.c",      // C source files
		"*.h",      // C header files
		"*.hxx",    // C++ header files
		"*.cpp",    // C++ source files
		"*.cc",     // C++ source files
		"*.cxx",    // C++ source files (alternative)
		"*.c++",    // C++ source files (alternative)
		"*.hpp",    // C++ header files
		"*.hh",     // C++ header files
		"*.h++",    // C++ header files (alternative)
		"*.java",   // Java source files
		"*.js",     // JavaScript source files
		"*.mjs",    // JavaScript ES6 modules
		"*.cjs",    // CommonJS modules
		"*.jsx",    // React JSX files
		"*.ts",     // TypeScript source files
		"*.tsx",    // TypeScript JSX files
		"*.go",     // Go source files
		"*.rs",     // Rust source files
		"*.swift",  // Swift source files
		"*.rb",     // Ruby source files
		"*.php",    // PHP source files
		"*.scala",  // Scala source files
		"*.kt",     // Kotlin source files
		"*.kts",    // Kotlin script files
		"*.r",      // R source files
		"*.R",      // R source files (alternative)
		"*.m",      // MATLAB/Objective-C source files
		"*.mm",     // Objective-C++ source files
		"*.f",      // Fortran source files
		"*.f90",    // Fortran 90 source files
		"*.f95",    // Fortran 95 source files
		"*.f03",    // Fortran 2003 source files
		"*.f08",    // Fortran 2008 source files
		"*.jl",     // Julia source files
		"*.lua",    // Lua source files
		"*.pl",     // Perl source files
		"*.pm",     // Perl modules
		"*.cs",     // C# source files
		"*.vb",     // Visual Basic source files
		"*.dart",   // Dart source files
		"*.groovy", // Groovy source files
		"*.elm",    // Elm source files
		"*.erl",    // Erlang source files
		"*.hrl",    // Erlang header files
		"*.ex",     // Elixir source files
		"*.exs",    // Elixir script files
		"*.hs",     // Haskell source files
		"*.lhs",    // Literate Haskell source files
		"*.clj",    // Clojure source files
		"*.cljs",   // ClojureScript source files
		"*.cljc",   // Clojure Common source files
		"*.cl",     // Common Lisp source files
		"*.lisp",   // Lisp source files
		"*.lsp",    // Lisp source files (alternative)
		"*.scm",    // Scheme source files
		"*.ss",     // Scheme source files (alternative)
		"*.rkt",    // Racket source files
		"*.sql",    // SQL files
		"*.psql",   // PostgreSQL files
		"*.mysql",  // MySQL files
		"*.sqlite", // SQLite files
		"*.zig",    // Zig source files
		"*.cu",     // CUDA source files
		"*.cuh",    // CUDA header files

		// Scripting and automation
		"*.bash",        // Bash scripts
		"*.zsh",         // Zsh scripts
		"*.fish",        // Fish shell scripts
		"*.csh",         // C shell scripts
		"*.tcsh",        // TC shell scripts
		"*.ksh",         // Korn shell scripts
		"*.ps1",         // PowerShell scripts
		"*.psm1",        // PowerShell modules
		"*.psd1",        // PowerShell data files
		"*.bat",         // Windows batch files
		"*.cmd",         // Windows command files
		"*.vbs",         // VBScript files
		"*.wsf",         // Windows Script Files
		"*.applescript", // AppleScript files
		"*.scpt",        // AppleScript compiled files
		"*.awk",         // AWK scripts
		"*.sed",         // sed scripts
		"*.expect",      // Expect scripts

		// Build and project files
		"*.env",             // Environment variable files
		"*.env.*",           // Environment files with suffixes
		".env*",             // Environment files (hidden)
		"Makefile*",         // Makefile variants
		"*.dockerfile",      // Dockerfile configurations
		"Dockerfile*",       // Dockerfile variants
		"*.mk",              // Make include files
		"*.cmake",           // CMake files
		"CMakeLists.txt",    // CMake configuration
		"*.gradle",          // Gradle build files
		"*.gradle.kts",      // Kotlin DSL Gradle files
		"build.gradle*",     // Gradle build files
		"settings.gradle*",  // Gradle settings files
		"*.sbt",             // SBT build files
		"*.mill",            // Mill build files
		"*.bazel",           // Bazel build files
		"*.bzl",             // Bazel extension files
		"BUILD*",            // Bazel BUILD files
		"WORKSPACE*",        // Bazel WORKSPACE files
		"*.buck",            // Buck build files
		"BUCK*",             // Buck BUILD files
		"*.ninja",           // Ninja build files
		"*.gyp",             // GYP build files
		"*.gypi",            // GYP include files
		"*.waf",             // Waf build files
		"wscript*",          // Waf build scripts
		"package.json",      // Node.js package file
		"package-lock.json", // Node.js lock file
		"yarn.lock",         // Yarn lock file
		"pnpm-lock.yaml",    // PNPM lock file
		"requirements*.txt", // Python requirements
		"Pipfile*",          // Python Pipenv files
		"pyproject.toml",    // Python project configuration
		"setup.cfg",         // Python setup configuration
		"tox.ini",           // Python tox configuration
		"poetry.lock",       // Python Poetry lock file
		"Cargo.toml",        // Rust package configuration
		"Cargo.lock",        // Rust lock file
		"go.mod",            // Go module file
		"go.sum",            // Go checksum file
		"composer.json",     // PHP Composer file
		"composer.lock",     // PHP Composer lock file
		"Gemfile*",          // Ruby Gemfile
		"*.gemspec",         // Ruby gem specification
		"mix.exs",           // Elixir Mix file
		"mix.lock",          // Elixir Mix lock file
		"rebar.config",      // Erlang Rebar config
		"rebar.lock",        // Erlang Rebar lock file

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

	// Large file size threshold
	WeightFileSizeThreshold int64 = 128 * 1024 * 1024
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

// For large unknown file type, usually it is a weight file.
func SizeShouldBeWeightFile(size int64) bool {
	return size > WeightFileSizeThreshold
}
