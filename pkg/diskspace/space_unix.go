//go:build linux || darwin

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

package diskspace

import (
	"math"

	"golang.org/x/sys/unix"
)

// availableSpace returns the number of bytes available to non-root users on
// the filesystem containing dir.
func availableSpace(dir string) (int64, error) {
	var stat unix.Statfs_t
	if err := unix.Statfs(dir, &stat); err != nil {
		return 0, err
	}

	// Guard against overflow: on Linux Bavail is uint64, and values exceeding
	// math.MaxInt64 would wrap negative when cast to int64. Cap at MaxInt64.
	bavail := stat.Bavail
	bsize := uint64(stat.Bsize)
	if bavail > 0 && bsize > uint64(math.MaxInt64)/bavail {
		return math.MaxInt64, nil
	}

	return int64(bavail * bsize), nil
}
