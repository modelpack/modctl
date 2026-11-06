//go:build unix

/*
 *     Copyright 2026 The ModelPack Authors
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

package build

import (
	"os"
	"syscall"
	"testing"

	"github.com/stretchr/testify/require"
)

// expectedFileOwner returns the numeric owner IDs reported by the OS.
func expectedFileOwner(t *testing.T, info os.FileInfo) (uid, gid uint32) {
	t.Helper()

	stat, ok := info.Sys().(*syscall.Stat_t)
	require.True(t, ok, "expected *syscall.Stat_t, got %T", info.Sys())

	return stat.Uid, stat.Gid
}
