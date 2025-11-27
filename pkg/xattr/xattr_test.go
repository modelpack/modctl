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

package xattr

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMakeKey(t *testing.T) {
	assert.Equal(t, "user.modctl", MakeKey("modctl"))
	assert.Equal(t, "user.modctl.size", MakeKey("modctl", "size"))
	assert.Equal(t, "user.modctl.file.digest", MakeKey("modctl", "file", "digest"))
}

func TestSetAndGet(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test_file")
	err := os.WriteFile(filePath, []byte("test"), 0644)
	require.NoError(t, err)

	// Check if filesystem supports xattrs.
	key := "user.test"
	testValue := []byte("test value")
	if err := Set(filePath, key, testValue); err != nil {
		t.Skip("Filesystem does not support extended attributes")
	}

	// Test set and get.
	value := []byte("hello world")
	err = Set(filePath, key, value)
	require.NoError(t, err)

	retrieved, err := Get(filePath, key)
	require.NoError(t, err)
	assert.Equal(t, value, retrieved)
}

func TestGetNonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test_file")
	err := os.WriteFile(filePath, []byte("test"), 0644)
	require.NoError(t, err)

	_, err = Get(filePath, "user.nonexistent")
	assert.Error(t, err)
}
