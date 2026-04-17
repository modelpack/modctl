package modelfile

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPathFilter(t *testing.T) {
	testcases := []struct {
		name            string
		exclude         []string
		include         []string
		expectedExclude []string
		expectedInclude []string
		expectError     bool
		errorMsg        string
	}{
		{
			name:            "normal exclude patterns",
			exclude:         []string{"*.log", "checkpoint*/"},
			expectedExclude: []string{"*.log", "checkpoint*"},
		},
		{
			name:        "invalid exclude pattern",
			exclude:     []string{"*.log", "[invalid"},
			expectError: true,
			errorMsg:    `invalid exclude pattern "[invalid"`,
		},
		{
			name:            "valid include patterns",
			include:         []string{"**/.*", ".weights/**"},
			expectedInclude: []string{"**/.*", ".weights/**"},
		},
		{
			name:        "invalid include pattern",
			include:     []string{"[invalid"},
			expectError: true,
			errorMsg:    `invalid include pattern "[invalid"`,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			filter, err := NewPathFilter(tc.exclude, tc.include)

			if tc.expectError {
				require.Error(t, err, "Expected an error")
				assert.Contains(t, err.Error(), tc.errorMsg)
				assert.Nil(t, filter)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, filter)
			if tc.expectedExclude != nil {
				assert.Equal(t, tc.expectedExclude, filter.excludePatterns)
			}
			if tc.expectedInclude != nil {
				assert.Equal(t, tc.expectedInclude, filter.includePatterns)
			}
		})
	}
}

func TestPathFilter_Match(t *testing.T) {
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
			// With doublestar, ** now matches across path separators
			filterName: "Directory_Double_Asterisk_Filter",
			patterns:   []string{"build/**"},
			tests: []struct {
				desc     string
				path     string
				expected bool
			}{
				{"matches file directly inside", "build/app", true},
				{"matches hidden file inside", "build/.config", true},
				{"matches the directory itself with doublestar", "build", true},
				{"matches nested files with doublestar", "build/assets/icon.png", true},
			},
		},
		{
			filterName: "Recursive_Glob_Filter",
			patterns:   []string{"**/*.log"},
			tests: []struct {
				desc     string
				path     string
				expected bool
			}{
				{"matches root level log", "debug.log", true},
				{"matches nested log", "subdir/debug.log", true},
				{"matches deeply nested log", "a/b/c/debug.log", true},
				{"does not match non-log", "main.go", false},
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
			filter, err := NewPathFilter(tc.patterns, nil)
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

func TestPathFilter_MatchInclude(t *testing.T) {
	testcases := []struct {
		name     string
		includes []string
		tests    []struct {
			desc     string
			path     string
			expected bool
		}
	}{
		{
			name:     "empty include patterns",
			includes: nil,
			tests: []struct {
				desc     string
				path     string
				expected bool
			}{
				{"any path returns false", "subdir/.hidden", false},
				{"root hidden returns false", ".hidden", false},
			},
		},
		{
			name:     "root-only dot pattern",
			includes: []string{".*"},
			tests: []struct {
				desc     string
				path     string
				expected bool
			}{
				{"matches root hidden file", ".hidden", true},
				{"matches root .env", ".env", true},
				{"does not match nested hidden", "subdir/.hidden", false},
				{"does not match normal file", "normal.txt", false},
			},
		},
		{
			name:     "recursive dot pattern",
			includes: []string{"**/.*"},
			tests: []struct {
				desc     string
				path     string
				expected bool
			}{
				{"matches root hidden", ".hidden", true},
				{"matches nested hidden", "subdir/.hidden", true},
				{"matches deeply nested", "a/b/.config", true},
				{"does not match normal", "subdir/normal", false},
			},
		},
		{
			name:     "specific dir pattern",
			includes: []string{".weights/**"},
			tests: []struct {
				desc     string
				path     string
				expected bool
			}{
				{"matches file inside", ".weights/model.bin", true},
				{"matches nested file inside", ".weights/sub/data.bin", true},
				{"does not match other hidden", ".config/file", false},
			},
		},
		{
			name:     "brace expansion",
			includes: []string{".{weights,config}/**"},
			tests: []struct {
				desc     string
				path     string
				expected bool
			}{
				{"matches .weights/x", ".weights/model.bin", true},
				{"matches .config/y", ".config/settings.json", true},
				{"does not match .other/z", ".other/file", false},
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			filter, err := NewPathFilter(nil, tc.includes)
			require.NoError(t, err)

			for _, test := range tc.tests {
				t.Run(test.desc, func(t *testing.T) {
					result := filter.MatchInclude(test.path)
					assert.Equal(t, test.expected, result, "Path: %q", test.path)
				})
			}
		})
	}
}

func TestPathFilter_ShouldDescend(t *testing.T) {
	testcases := []struct {
		name     string
		includes []string
		tests    []struct {
			desc     string
			dirPath  string
			expected bool
		}
	}{
		{
			name:     "no include patterns",
			includes: nil,
			tests: []struct {
				desc     string
				dirPath  string
				expected bool
			}{
				{"any dir returns false", ".weights", false},
			},
		},
		{
			name:     "recursive dot pattern enters dot dirs only",
			includes: []string{"**/.*"},
			tests: []struct {
				desc     string
				dirPath  string
				expected bool
			}{
				{"enters root hidden dir", ".weights", true},
				{"enters nested hidden dir", "subdir/.hidden", true},
				{"does not enter __pycache__", "__pycache__", false},
				{"does not enter normal skippable", "modelfile", false},
			},
		},
		{
			name:     "pycache pattern enters pycache only",
			includes: []string{"**/__pycache__/**"},
			tests: []struct {
				desc     string
				dirPath  string
				expected bool
			}{
				{"enters __pycache__", "__pycache__", true},
				{"enters nested __pycache__", "src/__pycache__", true},
				{"does not enter hidden dir", ".hidden", false},
			},
		},
		{
			name:     "prefix match for specific dir",
			includes: []string{".weights/**"},
			tests: []struct {
				desc     string
				dirPath  string
				expected bool
			}{
				{"enters .weights", ".weights", true},
				{"does not enter .weird", ".weird", false},
				{"does not enter .config", ".config", false},
			},
		},
		{
			name:     "prefix boundary",
			includes: []string{".weights/*"},
			tests: []struct {
				desc     string
				dirPath  string
				expected bool
			}{
				{"enters .weights", ".weights", true},
				{"does not enter .weird", ".weird", false},
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			filter, err := NewPathFilter(nil, tc.includes)
			require.NoError(t, err)

			for _, test := range tc.tests {
				t.Run(test.desc, func(t *testing.T) {
					result := filter.ShouldDescend(test.dirPath)
					assert.Equal(t, test.expected, result, "Dir: %q", test.dirPath)
				})
			}
		})
	}
}

func TestPathFilter_IncludeExcludeInteraction(t *testing.T) {
	// File matches both include and exclude → excluded (exclude wins)
	filter, err := NewPathFilter([]string{"**/.env"}, []string{"**/.*"})
	require.NoError(t, err)

	// .env matches include pattern **/.*
	assert.True(t, filter.MatchInclude(".env"))
	// .env also matches exclude pattern **/.env
	assert.True(t, filter.Match(".env"))

	// .hidden matches include but not exclude
	assert.True(t, filter.MatchInclude(".hidden"))
	assert.False(t, filter.Match(".hidden"))
}
