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

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseReference(t *testing.T) {
	tests := []struct {
		input    string
		expected string
		hasError bool
	}{
		{"example.com/repo:tag", "example.com/repo", false},
		{"example.com/repo@sha256:1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef", "example.com/repo", false},
		{"invalid reference", "", true},
	}

	for _, test := range tests {
		ref, err := ParseReference(test.input)
		if test.hasError {
			assert.Error(t, err)
		} else {
			assert.NoError(t, err)
			assert.Equal(t, test.expected, ref.Repository())
		}
	}
}

func TestReferencer_Repository(t *testing.T) {
	ref, err := ParseReference("example.com/repo:tag")
	assert.NoError(t, err)
	assert.Equal(t, "example.com/repo", ref.Repository())
}

func TestReferencer_Tag(t *testing.T) {
	ref, err := ParseReference("example.com/repo:tag")
	assert.NoError(t, err)
	assert.Equal(t, "tag", ref.Tag())

	ref, err = ParseReference("example.com/repo@sha256:1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef")
	assert.NoError(t, err)
	assert.Equal(t, "", ref.Tag())
}

func TestReferencer_Digest(t *testing.T) {
	ref, err := ParseReference("example.com/repo@sha256:1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef")
	assert.NoError(t, err)
	assert.Equal(t, "sha256:1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef", ref.Digest())

	ref, err = ParseReference("example.com/repo:tag")
	assert.NoError(t, err)
	assert.Equal(t, "", ref.Digest())
}

func TestReferencer_Domain(t *testing.T) {
	ref, err := ParseReference("example.com/repo:tag")
	assert.NoError(t, err)
	assert.Equal(t, "example.com", ref.Domain())

	ref, err = ParseReference("example.com/repo@sha256:1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef")
	assert.NoError(t, err)
	assert.Equal(t, "example.com", ref.Domain())
}

func TestReferencer(t *testing.T) {
	ref, err := ParseReference("example.com/repo:tag@sha256:1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef")
	assert.NoError(t, err)
	assert.Equal(t, "example.com/repo", ref.Repository())
	assert.Equal(t, "tag", ref.Tag())
	assert.Equal(t, "sha256:1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef", ref.Digest())
}
