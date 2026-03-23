/*
 *     Copyright 2024 The CNAI Authors
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

package diskspace

import (
	"strings"
	"testing"
)

func TestCheck_ZeroBytes(t *testing.T) {
	err := Check("/tmp", 0)
	if err != nil {
		t.Errorf("expected nil error for zero bytes, got: %v", err)
	}
}

func TestCheck_NegativeBytes(t *testing.T) {
	err := Check("/tmp", -1)
	if err != nil {
		t.Errorf("expected nil error for negative bytes, got: %v", err)
	}
}

func TestCheck_SmallSize(t *testing.T) {
	// 1 byte should always have enough space
	err := Check("/tmp", 1)
	if err != nil {
		t.Errorf("expected nil error for 1 byte, got: %v", err)
	}
}

func TestCheck_ExtremelyLargeSize(t *testing.T) {
	// 1 exabyte should always fail
	err := Check("/tmp", 1<<60)
	if err == nil {
		t.Error("expected error for extremely large size, got nil")
	}
	if !strings.Contains(err.Error(), "insufficient disk space") {
		t.Errorf("expected 'insufficient disk space' in error, got: %v", err)
	}
}

func TestCheck_NonExistentDirWalksUp(t *testing.T) {
	// Should walk up to find an existing parent directory
	err := Check("/tmp/nonexistent-modctl-test-dir-12345/subdir", 1)
	if err != nil {
		t.Errorf("expected nil error when parent exists, got: %v", err)
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		bytes    int64
		expected string
	}{
		{0, "0 B"},
		{500, "500 B"},
		{1024, "1.00 KB"},
		{1536, "1.50 KB"},
		{1048576, "1.00 MB"},
		{1073741824, "1.00 GB"},
		{1099511627776, "1.00 TB"},
	}

	for _, tt := range tests {
		result := formatBytes(tt.bytes)
		if result != tt.expected {
			t.Errorf("formatBytes(%d) = %s, want %s", tt.bytes, result, tt.expected)
		}
	}
}
