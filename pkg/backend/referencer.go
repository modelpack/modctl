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

package backend

import "github.com/distribution/reference"

// Referencer is the interface for the reference.
type Referencer interface {
	// Repository returns the repository of the reference.
	Repository() string
	// Tag returns the tag of the reference.
	Tag() string
	// Digest returns the digest of the reference.
	Digest() string
}

type referencer struct {
	named reference.Named
}

// ParseReference parses the reference.
func ParseReference(ref string) (Referencer, error) {
	named, err := reference.ParseNamed(ref)
	if err != nil {
		return nil, err
	}

	return &referencer{named: named}, nil
}

// Repository returns the repository of the reference.
func (r *referencer) Repository() string {
	return reference.TrimNamed(r.named).String()
}

// Tag returns the tag of the reference.
func (r *referencer) Tag() string {
	if tagged, ok := r.named.(reference.Tagged); ok {
		return tagged.Tag()
	}

	return ""
}

// Digest returns the digest of the reference.
func (r *referencer) Digest() string {
	if digested, ok := r.named.(reference.Digested); ok {
		return digested.Digest().String()
	}

	return ""
}
