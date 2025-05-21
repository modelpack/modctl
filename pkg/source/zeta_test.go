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

func TestZeta(t *testing.T) {
	parser := &zeta{}
	info, err := parser.Parse("testdata/zeta-repo")
	assert.NoError(t, err)
	assert.Equal(t, "https://zeta.com/test/zeta-repo", info.URL, "source url should be equal to expected")
	assert.Equal(t, "928f09e2c2162f7c94e8be8e36c2f9a3bd978c2756ab6815dba1a4f4228f279a", info.Commit, "commit should be equal to expected")
	assert.Equal(t, true, info.Dirty, "dirty should be equal to expected")
}
