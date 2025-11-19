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
	"strings"

	"golang.org/x/sys/unix"
)

const (
	// Prefix for all xattr keys to ensure compatibility across platforms.
	// Linux requires "user." prefix for user-space xattrs, while macOS allows any key.
	Prefix = "user."

	// Common xattr keys.
	KeySize   = "modctl.size"
	KeyMtime  = "modctl.mtime"
	KeySha256 = "modctl.sha256"
)

// Get retrieves an xattr value for a given key.
func Get(path, key string) ([]byte, error) {
	var value []byte
	sz, err := unix.Getxattr(path, key, value)
	if err != nil {
		return nil, err
	}

	value = make([]byte, sz)
	_, err = unix.Getxattr(path, key, value)
	if err != nil {
		return nil, err
	}

	return value, nil
}

// Set sets an xattr value for a given key.
func Set(path, key string, value []byte) error {
	return unix.Setxattr(path, key, value, 0)
}

// MakeKey creates a fully-qualified xattr key with the user prefix.
func MakeKey(parts ...string) string {
	return Prefix + strings.Join(parts, ".")
}
