//go:build windows

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

	"golang.org/x/sys/windows"
)

// availableSpace returns the number of bytes available to the calling user on
// the volume containing dir, taking per-user disk quotas into account.
func availableSpace(dir string) (int64, error) {
	path, err := windows.UTF16PtrFromString(dir)
	if err != nil {
		return 0, err
	}

	var freeBytesAvailable uint64
	if err := windows.GetDiskFreeSpaceEx(path, &freeBytesAvailable, nil, nil); err != nil {
		return 0, err
	}

	// Cap at MaxInt64 to avoid wrapping negative when cast to int64.
	if freeBytesAvailable > math.MaxInt64 {
		return math.MaxInt64, nil
	}

	return int64(freeBytesAvailable), nil
}
