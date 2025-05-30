//go:build !disable_libgit2

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

package source

import (
	"fmt"
	"path/filepath"
	"strings"

	git2go "github.com/libgit2/git2go/v34"
)

const (
	// The error returned by libgit2 when the user is not the owner of the git repository.
	safeDirectoryNotFoundErrorMsg = "config value 'safe.directory' was not found"
)

// isSafeDirectoryNotFoundError checks if the error is a safe.directory not found error.
func isSafeDirectoryNotFoundError(err error) bool {
	if err != nil {
		return strings.Contains(err.Error(), safeDirectoryNotFoundErrorMsg)
	}

	return false
}

type git struct{}

func (g *git) Parse(workspace string) (*Info, error) {
	repo, err := git2go.OpenRepository(workspace)
	if err != nil {
		// Try to set safe.directory manually if it is not found, and try to open repository again.
		if isSafeDirectoryNotFoundError(err) {
			config, err := git2go.OpenDefault()
			if err != nil {
				return nil, fmt.Errorf("failed to open config: %w", err)
			}
			defer config.Free()

			absWorkspace, err := filepath.Abs(workspace)
			if err != nil {
				return nil, fmt.Errorf("failed to get absolute path of workspace: %w", err)
			}

			if err := config.SetString("safe.directory", absWorkspace); err != nil {
				return nil, fmt.Errorf("failed to set safe.directory: %w", err)
			}
		}

		repo, err = git2go.OpenRepository(workspace)
		if err != nil {
			return nil, fmt.Errorf("failed to open repository at %s: %w", workspace, err)
		}
	}
	defer repo.Free()

	// Get remote URL(Source URL).
	remote, err := repo.Remotes.Lookup("origin")
	if err != nil {
		return nil, fmt.Errorf("failed to lookup remote: %w", err)
	}
	defer remote.Free()

	url := remote.Url()
	if len(url) == 0 {
		return nil, fmt.Errorf("failed to get remote URL")
	}

	// Get HEAD commit.
	head, err := repo.Head()
	if err != nil {
		return nil, fmt.Errorf("failed to get HEAD: %w", err)
	}
	defer head.Free()

	commitSHA := head.Target().String()
	if len(commitSHA) == 0 {
		return nil, fmt.Errorf("failed to get HEAD commit")
	}

	// Check whether the workspace is dirty.
	statusOpts := git2go.StatusOptions{}
	statusOpts.Show = git2go.StatusShowIndexAndWorkdir
	statusOpts.Flags = git2go.StatusOptIncludeUntracked | git2go.StatusOptRenamesHeadToIndex | git2go.StatusOptSortCaseSensitively

	statusList, err := repo.StatusList(&statusOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to get status list: %w", err)
	}
	defer statusList.Free()

	entryCount, err := statusList.EntryCount()
	if err != nil {
		return nil, fmt.Errorf("failed to get status entry count: %w", err)
	}

	isDirty := entryCount > 0

	return &Info{
		URL:    url,
		Commit: commitSHA,
		Dirty:  isDirty,
	}, nil
}
