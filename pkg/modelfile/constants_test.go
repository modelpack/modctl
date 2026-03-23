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
		{"events.out.tfevents.1679012345.hostname", true}, // *.tfevents* matches via filepath.Match (wildcards match dots)
		{"training.db", true},

		// Sharded/variant patterns.
		{"model.bin.1", true},
		{"model.bin.part2", true},
		{"model.gguf.part1", true},
		{"model.gguf.00001-of-00003", true},
		{"model.llamafile.zip", true},
		{"model.llamafile.gz", true},

		// Existing patterns still work.
		{"model.safetensors", true},
		{"model.bin", true},
		{"model.gguf", true},
		{"model.llamafile", true},

		// Non-matching files.
		{"readme.txt", false},
		{"script.py", false},
	}

	assert := assert.New(t)
	for _, tc := range testCases {
		assert.Equal(tc.expected, IsFileType(tc.filename, ModelFilePatterns), "filename: %s", tc.filename)
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
