package modelfile

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsFileType(t *testing.T) {
	testCases := []struct {
		filename string
		patterns []string
		expected bool
	}{
		{"config.json", []string{"*.json", "*.yaml"}, true},
		{"config.yaml", []string{"*.json", "*.yaml"}, true},
		{"config.txt", []string{"*.json", "*.yaml"}, false},
		{"image.JPG", []string{"*.jpg", "*.png"}, true},
		{"image.jpeg", []string{"*.jpg", "*.png"}, false},
		{"script.py", []string{"*.py", "*.sh"}, true},
		{"script.sh", []string{"*.py", "*.sh"}, true},
		{"script.bash", []string{"*.py", "*.sh"}, false},
		{"folder/config.json", []string{"*.json", "*.yaml"}, true},
		{"FOLDER/config.json", []string{"*.json", "*.yaml"}, true},
		{"folder/CONFIG.JSON", []string{"*.json", "*.yaml"}, true},
		{"folder\\config.json", []string{"*.json", "*.yaml"}, true},
	}

	assert := assert.New(t)
	for _, tc := range testCases {
		assert.Equal(tc.expected, IsFileType(tc.filename, tc.patterns))
	}
}

func TestIsFileTypeModelPatterns(t *testing.T) {
	testCases := []struct {
		filename string
		expected bool
	}{
		// New data/dataset formats.
		{"dataset.arrow", true},
		{"train.parquet", true},
		{"model.ftz", true},
		{"feats.ark", true},
		{"training.db", true},

		// Sharded/variant patterns.
		{"model.bin.1", true},
		{"model.bin.part2", true},
		{"model.gguf.part1", true},
		{"model.gguf.00001-of-00003", true},
		{"model.onnx_data", true},
		{"model.onnx_data_1", true},
		{"ckpt-0/tensor00001_000", true},
		{"model.llamafile.zip", true},
		{"model.llamafile.gz", true},

		// Existing patterns still work.
		{"model.safetensors", true},
		{"model.bin", true},
		{"model.gguf", true},
		{"model.mil", true},
		{"model.llamafile", true},

		// Non-matching files.
		{"merges.txt", false},
		{"readme.txt", false},
		{"script.py", false},
		{"events.out.tfevents.1679012345.hostname", false}, // tfevents moved to DocFilePatterns
	}

	assert := assert.New(t)
	for _, tc := range testCases {
		assert.Equal(tc.expected, IsFileType(tc.filename, ModelFilePatterns), "filename: %s", tc.filename)
	}
}

func TestIsFileTypeConfigPatterns(t *testing.T) {
	testCases := []struct {
		filename string
		expected bool
	}{
		{"vocab.txt", true},
		{"merges.txt", true},
		{"added_tokens.txt", true},
		{"chat_template.jinja", true},
		{"tokenizer.tiktoken", true},
		{"spiece.model", true},
		{"sentencepiece.bpe.model", true},
		{"sentencepiece.bpe.vocab", true},
		{"tiktoken.model", true},
		{"weights.model", false},
		{"readme.txt", false},
	}

	assert := assert.New(t)
	for _, tc := range testCases {
		assert.Equal(tc.expected, IsFileType(tc.filename, ConfigFilePatterns), "filename: %s", tc.filename)
	}
}

func TestIsFileTypeDocPatternsTfevents(t *testing.T) {
	testCases := []struct {
		filename string
		expected bool
	}{
		{"events.out.tfevents.1679012345.hostname", true}, // *.tfevents* matches via filepath.Match (wildcards match dots)
		{"train.tfevents", true},
		{"model.safetensors", false}, // model files should not match doc patterns
	}

	assert := assert.New(t)
	for _, tc := range testCases {
		assert.Equal(tc.expected, IsFileType(tc.filename, DocFilePatterns), "filename: %s", tc.filename)
	}
}

func TestInferFileType(t *testing.T) {
	testCases := []struct {
		name     string
		filename string
		fileSize int64
		expected FileType
	}{
		// Known extensions - size should not matter
		{"config json", "config.json", 1024, FileTypeConfig},
		{"config yaml", "settings.yaml", 1024, FileTypeConfig},
		{"model safetensors", "model.safetensors", 1024, FileTypeModel},
		{"model bin", "weights.bin", 1024, FileTypeModel},
		{"model onnx external data", "model.onnx_data_1", 1024, FileTypeModel},
		{"model coreml mil", "model.mil", 1024, FileTypeModel},
		{"checkpoint tensor shard", "ckpt-0/tensor00001_000", 1024, FileTypeModel},
		{"code python", "script.py", 1024, FileTypeCode},
		{"code go", "main.go", 1024, FileTypeCode},
		{"doc markdown", "README.md", 1024, FileTypeDoc},
		{"doc pdf", "guide.pdf", 1024, FileTypeDoc},
		{"tokenizer vocab txt", "vocab.txt", 1024, FileTypeConfig},
		{"tokenizer merges txt", "merges.txt", 1024, FileTypeConfig},
		{"tokenizer added tokens txt", "added_tokens.txt", 1024, FileTypeConfig},
		{"sentencepiece spiece model", "spiece.model", 1024, FileTypeConfig},
		{"sentencepiece bpe model", "sentencepiece.bpe.model", 1024, FileTypeConfig},
		{"tiktoken model", "tiktoken.model", 1024, FileTypeConfig},
		{"chat template jinja", "chat_template.jinja", 1024, FileTypeConfig},

		// Dotfile with known secondary extension
		{".cache.json is config", ".cache.json", 1024, FileTypeConfig},
		{".hidden.py is code", ".hidden.py", 1024, FileTypeCode},

		// Unrecognized - small files fallback to code
		{"dotfile small", ".metadata", 1024, FileTypeCode},
		{"no extension small", "unknown_file", 1024, FileTypeCode},
		{"unknown ext small", "data.xyz", 50 * 1024, FileTypeCode},

		// Unrecognized - large files fallback to model
		{"dotfile large", ".metadata", 200 * 1024 * 1024, FileTypeModel},
		{"no extension large", "unknown_file", 200 * 1024 * 1024, FileTypeModel},
		{"unknown ext large", "data.xyz", 200 * 1024 * 1024, FileTypeModel},

		// Edge case: exactly at threshold (WeightFileSizeThreshold = 128*1000*1000) should be code
		{"at threshold", "borderline", WeightFileSizeThreshold, FileTypeCode},
		// Just above threshold should be model
		{"above threshold", "borderline", WeightFileSizeThreshold + 1, FileTypeModel},
	}

	assert := assert.New(t)
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(tc.expected, InferFileType(tc.filename, tc.fileSize),
				"InferFileType(%q, %d)", tc.filename, tc.fileSize)
		})
	}
}

func TestIsSkippable(t *testing.T) {
	testCases := []struct {
		filename string
		expected bool
	}{
		{".hiddenfile", true},
		{"modelfile", true},
		{"__pycache__", true},
		{"file.pyc", true},
		{"file.pyo", true},
		{"file.pyd", true},
		{"visiblefile.txt", false},
		{"directory", false},
	}

	assert := assert.New(t)
	for _, tc := range testCases {
		assert.Equal(tc.expected, isSkippable(tc.filename))
	}
}
