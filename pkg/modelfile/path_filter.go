package modelfile

import (
	"fmt"
	"path/filepath"
	"strings"
)

type PathFilter struct {
	patterns []string
}

func NewPathFilter(patterns ...string) (*PathFilter, error) {
	var cleaned []string
	for _, p := range patterns {
		// validate the pattern
		if _, err := filepath.Match(p, ""); err != nil {
			return nil, fmt.Errorf("invalid exclude pattern: %q", p)
		}
		// since filepath.Walk never returns a path with trailing separator, we need to remove separator from patterns
		cleaned = append(cleaned, strings.TrimRight(p, string(filepath.Separator)))
	}
	return &PathFilter{patterns: cleaned}, nil
}

func (pf *PathFilter) Match(path string) bool {
	if len(pf.patterns) == 0 {
		return false
	}

	for _, pattern := range pf.patterns {
		matched, err := filepath.Match(pattern, path)
		if err != nil {
			// The only possible returned error is ErrBadPattern
			// which we checked when creating the filter
			return false
		}
		if matched {
			return true
		}
	}

	return false
}
