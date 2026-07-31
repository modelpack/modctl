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

//go:build linux || darwin

package diskspace

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCheck_ZeroBytes(t *testing.T) {
	assert.NoError(t, Check("/tmp", 0))
}

func TestCheck_NegativeBytes(t *testing.T) {
	assert.NoError(t, Check("/tmp", -1))
}

func TestCheck_SmallSize(t *testing.T) {
	// 1 byte should always have enough space
	assert.NoError(t, Check("/tmp", 1))
}

func TestCheck_ExtremelyLargeSize(t *testing.T) {
	// 1 exabyte should always fail
	err := Check("/tmp", 1<<60)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "insufficient disk space")
}

func TestCheck_NonExistentDirWalksUp(t *testing.T) {
	// Should walk up to find an existing parent directory
	assert.NoError(t, Check("/tmp/nonexistent-modctl-test-dir-12345/subdir", 1))
}

func TestCheck_EmptyPath(t *testing.T) {
	// Empty string should walk up to root and succeed for small size
	assert.NoError(t, Check("", 1))
}

func TestCheck_RootDir(t *testing.T) {
	assert.NoError(t, Check("/", 1))
}

func TestCheck_DeeplyNestedNonExistentDir(t *testing.T) {
	assert.NoError(t, Check("/tmp/a/b/c/d/e/f/g/h", 1))
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		bytes    int64
		expected string
	}{
		{-1, "0 B"},
		{-1024, "0 B"},
		{0, "0 B"},
		{500, "500 B"},
		{1024, "1.00 KB"},
		{1536, "1.50 KB"},
		{1048576, "1.00 MB"},
		{1073741824, "1.00 GB"},
		{1099511627776, "1.00 TB"},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.expected, formatBytes(tt.bytes), "formatBytes(%d)", tt.bytes)
	}
}
