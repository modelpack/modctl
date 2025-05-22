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
	"context"
	"fmt"

	pkgzeta "github.com/antgroup/hugescm/pkg/zeta"
)

type zeta struct{}

func (z *zeta) Parse(workspace string) (*Info, error) {
	ctx := context.Background()
	repo, err := pkgzeta.Open(ctx, &pkgzeta.OpenOptions{
		Worktree: workspace,
		Quiet:    true,
		Verbose:  false,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to open repo: %w", err)
	}
	defer repo.Close()

	url := repo.Core.Remote
	if len(url) == 0 {
		return nil, fmt.Errorf("no remote URL found")
	}

	rev, err := repo.Revision(ctx, "HEAD")
	if err != nil {
		return nil, fmt.Errorf("failed to get HEAD revision: %w", err)
	}
	commitHash := rev.String()

	status, err := repo.Worktree().Status(ctx, false)
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
