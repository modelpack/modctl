package modelfile

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
)

type PathFilter struct {
	excludePatterns []string
	includePatterns []string
}

func NewPathFilter(excludePatterns []string, includePatterns []string) (*PathFilter, error) {
	var cleanedExclude []string
	for _, p := range excludePatterns {
		if _, err := doublestar.PathMatch(p, ""); err != nil {
			return nil, fmt.Errorf("invalid exclude pattern %q: %w", p, err)
		}
		cleanedExclude = append(cleanedExclude, strings.TrimRight(p, string(filepath.Separator)))
	}

	var cleanedInclude []string
	for _, p := range includePatterns {
		if _, err := doublestar.PathMatch(p, ""); err != nil {
			return nil, fmt.Errorf("invalid include pattern %q: %w", p, err)
		}
		cleanedInclude = append(cleanedInclude, strings.TrimRight(p, string(filepath.Separator)))
	}

	return &PathFilter{
		excludePatterns: cleanedExclude,
		includePatterns: cleanedInclude,
	}, nil
}

// Match checks if a path matches any exclude pattern.
func (pf *PathFilter) Match(path string) bool {
	if len(pf.excludePatterns) == 0 {
		return false
	}

	for _, pattern := range pf.excludePatterns {
		matched, err := doublestar.PathMatch(pattern, path)
		if err != nil {
			return false
		}
		if matched {
			return true
		}
	}

	return false
}

// MatchInclude checks if a relative path matches any include pattern.
func (pf *PathFilter) MatchInclude(relPath string) bool {
	if len(pf.includePatterns) == 0 {
		return false
	}

	for _, pattern := range pf.includePatterns {
		matched, err := doublestar.PathMatch(pattern, relPath)
		if err != nil {
			return false
		}
		if matched {
			return true
		}
	}

	return false
}

// ShouldDescend checks if a skippable directory should be entered
// because an include pattern might match files inside it.
func (pf *PathFilter) ShouldDescend(dirRelPath string) bool {
	if len(pf.includePatterns) == 0 {
		return false
	}

	for _, pattern := range pf.includePatterns {
		// Direct match: pattern matches the directory itself
		// e.g., "**/.*" matches ".weights" (** matches zero segments)
		matched, err := doublestar.PathMatch(pattern, dirRelPath)
		if err == nil && matched {
			return true
		}

		// Prefix match: pattern starts with dirRelPath/
		// e.g., ".weights/**" starts with ".weights/"
		prefix := dirRelPath + string(filepath.Separator)
		if strings.HasPrefix(pattern, prefix) {
			return true
		}
	}

	return false
}
