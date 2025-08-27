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
