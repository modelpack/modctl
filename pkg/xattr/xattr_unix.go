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

package xattr

import "golang.org/x/sys/unix"

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
