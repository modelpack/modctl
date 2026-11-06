/*
 *     Copyright 2025 The ModelPack Authors
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

import "strings"

const (
	// Prefix for all xattr keys to ensure compatibility across platforms.
	// Linux requires "user." prefix for user-space xattrs, while macOS allows any key.
	Prefix = "user."

	// Common xattr keys.
	KeySize   = "modctl.size"
	KeyMtime  = "modctl.mtime"
	KeySha256 = "modctl.sha256"
)

// MakeKey creates a fully-qualified xattr key with the user prefix.
func MakeKey(parts ...string) string {
	return Prefix + strings.Join(parts, ".")
}
