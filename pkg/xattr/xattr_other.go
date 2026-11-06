//go:build !linux && !darwin

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

import (
	"errors"
	"fmt"
)

// errNotSupported is returned on platforms without extended attribute support.
var errNotSupported = fmt.Errorf("xattr: %w on this platform", errors.ErrUnsupported)

// Get always fails on this platform because extended attributes are not
// supported. Callers treat the error as a cache miss.
func Get(path, key string) ([]byte, error) {
	return nil, errNotSupported
}

// Set always fails on this platform because extended attributes are not
// supported. Callers treat the error as a best-effort failure.
func Set(path, key string, value []byte) error {
	return errNotSupported
}
