package modelfile

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPathFilter(t *testing.T) {
	testcases := []struct {
		name        string
		input       []string
		expected    []string
		expectError bool
		errorMsg    string
	}{
		{
			name:     "normal patterns",
			input:    []string{"*.log", "checkpoint*/"},
			expected: []string{"*.log", "checkpoint*"},
		},
		{
			name:        "invalid pattern",
			input:       []string{"*.log", "[invalid"},
			expectError: true,
			errorMsg:    `invalid exclude pattern: "[invalid"`,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			filter, err := NewPathFilter(tc.input...)

			if tc.expectError {
				require.Error(t, err, "Expected an error for input: %q", tc.input)
				assert.Contains(t, err.Error(), tc.errorMsg)
				assert.Nil(t, filter)
				return
			}

			require.NoError(t, err, "Did not expect an error for input: %q", tc.input)
			require.NotNil(t, filter)
			assert.Equal(t, tc.expected, filter.patterns)
		})
	}
}

func TestPathFilter_Matches(t *testing.T) {
	testcases := []struct {
		filterName string
		patterns   []string
		tests      []struct {
			desc     string
			path     string
			expected bool
		}
	}{
		{
			filterName: "Empty_Filter",
			patterns:   []string{},
			tests: []struct {
				desc     string
				path     string
				expected bool
			}{
				{"any file", "any/path/file.txt", false},
				{"root file", "main.go", false},
				{"empty path", "", false},
			},
		},
		{
			filterName: "Single_Asterisk_Filter",
			patterns:   []string{"*.log"},
			tests: []struct {
				desc     string
				path     string
				expected bool
			}{
				{"matches a simple log file", "debug.log", true},
				{"matches a hidden log file", ".config.log", true},
				{"does not match if not at end", "debug.log.old", false},
				{"does not match different extension", "main.go", false},
				{"does not cross path separator", "logs/debug.log", false},
			},
		},
		{
			filterName: "Directory_Wildcard_Filter",
			patterns:   []string{"build/*"},
			tests: []struct {
				desc     string
				path     string
				expected bool
			}{
				{"matches file directly inside", "build/app", true},
				{"matches hidden file inside", "build/.config", true},
				{"does not match the directory itself", "build", false},
				{"does not match nested files", "build/assets/icon.png", false},
			},
		},
		{
			// Since filepath.Match does not support **, the behavior is the same as Directory_Wildcard_Filter
			filterName: "Directory_Double_Asterisk_Filter",
			patterns:   []string{"build/**"},
			tests: []struct {
				desc     string
				path     string
				expected bool
			}{
				{"matches file directly inside", "build/app", true},
				{"matches hidden file inside", "build/.config", true},
				{"does not match the directory itself", "build", false},
				{"does not match nested files", "build/assets/icon.png", false},
			},
		},
		{
			filterName: "Directory_Filter",
			patterns:   []string{"checkpoint/"},
			tests: []struct {
				desc     string
				path     string
				expected bool
			}{
				{"matches the directory itself", "checkpoint", true},
				{"match file inside", "checkpoint/file.py", false},
			},
		},
		{
			filterName: "Complex_Filter_With_Multiple_Patterns",
			patterns:   []string{"*.tmp", ".git*", "build/"},
			tests: []struct {
				desc     string
				path     string
				expected bool
			}{
				{"matches a .tmp file", "temp.tmp", true},
				{"matches .git directory", ".git", true},
				{"matches .gitignore file", ".gitignore", true},
				{"matches build directory exactly", "build", true},
				{"does not cross separator", "data/cache.tmp", false},
				{"does not match file inside build/", "build/app.js", false},
				{"does not match src file", "src/main.go", false},
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.filterName, func(t *testing.T) {
			filter, err := NewPathFilter(tc.patterns...)
			require.NoError(t, err, "Filter creation with patterns %q failed", tc.patterns)
			require.NotNil(t, filter)

			for _, test := range tc.tests {
				t.Run(test.desc, func(t *testing.T) {
					result := filter.Match(test.path)
					assert.Equal(t, test.expected, result, fmt.Sprintf("Path: %q", test.path))
				})
			}
		})
	}
}
