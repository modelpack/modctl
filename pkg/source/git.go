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

	gogit "github.com/go-git/go-git/v5"
)

type git struct{}

func (g *git) Parse(workspace string) (*Info, error) {
	repo, err := gogit.PlainOpen(workspace)
	if err != nil {
		return nil, fmt.Errorf("failed to open repo: %w", err)
	}

	// By default, use the origin as the remote.
	remote, err := repo.Remote("origin")
	if err != nil {
		return nil, fmt.Errorf("failed to get remote: %w", err)
	}
	remoteURLs := remote.Config().URLs
	if len(remoteURLs) == 0 {
		return nil, fmt.Errorf("no URLs found for remote 'origin'")
	}
	url := remoteURLs[0]

	// Fetch the head commit.
	head, err := repo.Head()
	if err != nil {
		return nil, fmt.Errorf("failed to get HEAD: %w", err)
	}
	commitHash := head.Hash().String()

	// Check if the workspace is dirty.
	worktree, err := repo.Worktree()
	if err != nil {
		return nil, fmt.Errorf("failed to get worktree: %w", err)
	}
	status, err := worktree.Status()
	if err != nil {
		return nil, fmt.Errorf("failed to get status: %w", err)
	}
	isDirty := !status.IsClean()

	return &Info{
		URL:    url,
		Commit: commitHash,
		Dirty:  isDirty,
	}, nil
}
