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

package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseStringArgs(t *testing.T) {
	testCases := []struct {
		args      []string
		start     int
		end       int
		expectErr bool
		expected  string
	}{
		{[]string{"foo"}, 1, 2, false, "foo"},
		{[]string{"bar"}, 3, 4, false, "bar"},
		{[]string{}, 5, 6, true, ""},
		{[]string{"foo", "bar"}, 7, 8, false, "foo bar"}, // Now handles multiple args by joining
		{[]string{""}, 9, 10, true, ""},
		// Additional test cases for spaces in file paths
		{[]string{"path", "with", "spaces/file.json"}, 11, 12, false, "path with spaces/file.json"},
		{[]string{"example", "workflows_Wan2.1/image_to_video_wan_480p_example.json"}, 13, 14, false, "example workflows_Wan2.1/image_to_video_wan_480p_example.json"},
		// Test cases for whitespace handling
		{[]string{" "}, 15, 16, true, ""},                                               // Whitespace-only argument should be rejected
		{[]string{"", ""}, 17, 18, true, ""},                                            // Multiple empty string arguments should be rejected
		{[]string{" a "}, 19, 20, false, " a "},                                         // Arguments with leading/trailing spaces should be preserved
		{[]string{"  ", "   "}, 21, 22, true, ""},                                       // Multiple whitespace-only arguments should be rejected
		{[]string{" path ", "with", " spaces "}, 23, 24, false, " path  with  spaces "}, // Mixed whitespace should be preserved
	}

	assert := assert.New(t)
	for _, tc := range testCases {
		node, err := parseStringArgs(tc.args, tc.start, tc.end)
		if tc.expectErr {
			assert.Error(err)
			assert.Nil(node)
			continue
		}

		assert.NoError(err)
		assert.NotNil(node)
		assert.Equal(tc.expected, node.GetValue())
		assert.Equal(tc.start, node.GetStartLine())
		assert.Equal(tc.end, node.GetEndLine())
	}
}
