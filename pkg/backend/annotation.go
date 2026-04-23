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

package backend

import (
	legacymodelspec "github.com/dragonflyoss/model-spec/specs-go/v1"
	modelspec "github.com/modelpack/model-spec/specs-go/v1"
)

// getAnnotationFilepath returns the filepath stored on a descriptor's
// annotations, preferring the modelpack key and falling back to the legacy
// dragonflyoss key so older artifacts remain readable. Returns empty string
// when neither key is present.
func getAnnotationFilepath(annotations map[string]string) string {
	if annotations == nil {
		return ""
	}
	if path := annotations[modelspec.AnnotationFilepath]; path != "" {
		return path
	}
	return annotations[legacymodelspec.AnnotationFilepath]
}
