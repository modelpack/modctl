/*
 *     Copyright 2024 The ModelPack Authors
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

package config

import (
	"os/user"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRootRespectsHomeEnv(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	// os.UserHomeDir reads USERPROFILE on Windows.
	t.Setenv("USERPROFILE", home)

	root, err := NewRoot()
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(home, ".modctl"), root.StorageDir)
	assert.Equal(t, filepath.Join(home, ".modctl", "logs"), root.LogDir)
}

func TestNewRootFallsBackToCurrentUser(t *testing.T) {
	u, err := user.Current()
	require.NoError(t, err)

	t.Setenv("HOME", "")
	t.Setenv("USERPROFILE", "")

	root, err := NewRoot()
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(u.HomeDir, ".modctl"), root.StorageDir)
	assert.Equal(t, filepath.Join(u.HomeDir, ".modctl", "logs"), root.LogDir)
}
