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

package diskspace

import (
	"fmt"
	"math"
	"os"
	"path/filepath"

	"golang.org/x/sys/unix"
)

const (
	// safetyMargin is the extra space ratio to account for metadata overhead
	// (manifests, temporary files, etc.). 10% extra required.
	safetyMargin = 1.1
)

// Check checks if the directory has enough disk space for the required bytes.
// It returns a descriptive error if space is insufficient, or nil if space is enough.
// The caller should use the returned error for warning purposes only and not
// treat it as a fatal error.
func Check(dir string, requiredBytes int64) error {
	if requiredBytes <= 0 {
		return nil
	}

	// Ensure the directory exists for statfs; walk up to find an existing parent.
	checkDir := dir
	for {
		if _, err := os.Stat(checkDir); err == nil {
			break
		}
		parent := filepath.Dir(checkDir)
		if parent == checkDir {
			// Reached filesystem root without finding an existing directory.
			return fmt.Errorf("cannot determine disk space: no existing directory found for path %s", dir)
		}
		checkDir = parent
	}

	var stat unix.Statfs_t
	if err := unix.Statfs(checkDir, &stat); err != nil {
		return fmt.Errorf("failed to check disk space for %s: %w", dir, err)
	}

	// Available space for non-root users.
	// Guard against overflow: on Linux Bavail is uint64, and values exceeding
	// math.MaxInt64 would wrap negative when cast to int64. Cap at MaxInt64.
	bavail := stat.Bavail
	bsize := uint64(stat.Bsize)
	var availableBytes int64
	if bavail > 0 && bsize > uint64(math.MaxInt64)/bavail {
		availableBytes = math.MaxInt64
	} else {
		availableBytes = int64(bavail * bsize)
	}
	requiredWithMargin := int64(float64(requiredBytes) * safetyMargin)

	if availableBytes < requiredWithMargin {
		return fmt.Errorf(
			"insufficient disk space in %s: available %s, required %s (with 10%% safety margin)",
			dir, formatBytes(availableBytes), formatBytes(requiredWithMargin),
		)
	}

	return nil
}

// formatBytes formats bytes into a human-readable string.
func formatBytes(bytes int64) string {
	if bytes < 0 {
		return "0 B"
	}

	const (
		kb = 1024
		mb = kb * 1024
		gb = mb * 1024
		tb = gb * 1024
	)

	switch {
	case bytes >= tb:
		return fmt.Sprintf("%.2f TB", float64(bytes)/float64(tb))
	case bytes >= gb:
		return fmt.Sprintf("%.2f GB", float64(bytes)/float64(gb))
	case bytes >= mb:
		return fmt.Sprintf("%.2f MB", float64(bytes)/float64(mb))
	case bytes >= kb:
		return fmt.Sprintf("%.2f KB", float64(bytes)/float64(kb))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}
