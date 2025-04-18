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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGit(t *testing.T) {
	parser := &git{}
	info, err := parser.Parse("testdata/git-repo")
	assert.NoError(t, err)
	assert.Equal(t, "https://github.com/octocat/Hello-World.git", info.URL, "source url should be equal to expected")
	assert.Equal(t, "7fd1a60b01f91b314f59955a4e4d4e80d8edf11d", info.Commit, "commit should be equal to expected")
	assert.Equal(t, false, info.Dirty, "dirty should be equal to expected")
}
