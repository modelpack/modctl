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
	"os"
	"path/filepath"
	"testing"
	"time"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGit(t *testing.T) {
	// Create a temporary directory for the test git repository
	tempDir, err := os.MkdirTemp("", "git-test-repo")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Initialize a new git repository
	repo, err := gogit.PlainInit(tempDir, false)
	require.NoError(t, err)

	// Create a remote "origin" with a test URL
	expectedURL := "https://github.com/octocat/Hello-World.git"
	_, err = repo.CreateRemote(&config.RemoteConfig{
		Name: "origin",
		URLs: []string{expectedURL},
	})
	require.NoError(t, err)

	// Create a test file and commit it
	testFile := filepath.Join(tempDir, "README.md")
	err = os.WriteFile(testFile, []byte("# Hello World"), 0644)
	require.NoError(t, err)

	worktree, err := repo.Worktree()
	require.NoError(t, err)

	_, err = worktree.Add("README.md")
	require.NoError(t, err)

	commit, err := worktree.Commit("Initial commit", &gogit.CommitOptions{
		Author: &object.Signature{
			Name:  "Test User",
			Email: "test@example.com",
			When:  time.Now(),
		},
	})
	require.NoError(t, err)

	// Now test the git parser
	parser := &git{}
	info, err := parser.Parse(tempDir)
	assert.NoError(t, err)
	assert.Equal(t, expectedURL, info.URL, "source url should be equal to expected")
	assert.Equal(t, commit.String(), info.Commit, "commit should be equal to expected")
	assert.Equal(t, false, info.Dirty, "dirty should be equal to expected")
}
