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
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParse(t *testing.T) {
	assert := assert.New(t)

	testCases := []struct {
		input     string
		expectErr bool
	}{
		{
			input: `
# This is a comment
MODEL model1
`,
			expectErr: false,
		},
		{
			input: `
# This is a comment
INVALID command
`,
			expectErr: true,
		},
		{
			input: `
# This is a comment
MODEL model1
NAME foo
`,
			expectErr: false,
		},
		{
			input: `
# This is a comment
`,
			expectErr: false,
		},
		{
			input: `
MODEL model1
`,
			expectErr: false,
		},
		{
			input: `

MODEL model1
`,
			expectErr: false,
		},
	}

	for _, tc := range testCases {
		reader := strings.NewReader(tc.input)
		root, err := Parse(reader)
		if tc.expectErr {
			assert.Error(err)
			assert.Nil(root)
			continue
		}

		assert.NoError(err)
		assert.NotNil(root)
	}
}

func TestIsComment(t *testing.T) {
	testCases := []struct {
		line     string
		expected bool
	}{
		{"# This is a comment", true},
		{"  # This is also a comment", false},
		{"This is not a comment", false},
	}

	assert := assert.New(t)
	for _, tc := range testCases {
		assert.Equal(tc.expected, isComment(tc.line))
	}
}

func TestIsCommand(t *testing.T) {
	testCases := []struct {
		line     string
		expected bool
	}{
		{"MODEL foo", true},
		{"MODEL foo", true},
		{"NAME bar", true},
		{"unknown command", false},
		{"  unknown command", false},
	}

	assert := assert.New(t)
	for _, tc := range testCases {
		assert.Equal(tc.expected, isCommand(tc.line))
	}
}

func TestIsEmptyContinuationLine(t *testing.T) {
	testCases := []struct {
		line     string
		expected bool
	}{
		{"", true},
		{" ", false},
		{"not empty", false},
	}

	assert := assert.New(t)
	for _, tc := range testCases {
		assert.Equal(tc.expected, isEmptyContinuationLine(tc.line))
	}
}

func TestParseCommandLine(t *testing.T) {
	testCases := []struct {
		line      string
		start     int
		end       int
		expectErr bool
		cmd       string
		args      []string
	}{
		{"CONFIG foo", 1, 2, false, "CONFIG", []string{"foo"}},
		{"CONFIG foo", 1, 2, false, "CONFIG", []string{"foo"}},
		{"MODEL foo", 1, 2, false, "MODEL", []string{"foo"}},
		{"CODE foo", 1, 2, false, "CODE", []string{"foo"}},
		{"DATASET foo", 1, 2, false, "DATASET", []string{"foo"}},
		{"NAME bar", 3, 4, false, "NAME", []string{"bar"}},
		{"ARCH transformer", 5, 6, false, "ARCH", []string{"transformer"}},
		{"FAMILY llama3", 7, 8, false, "FAMILY", []string{"llama3"}},
		{"FORMAT onnx", 9, 10, false, "FORMAT", []string{"onnx"}},
		{"PARAMSIZE 100", 11, 12, false, "PARAMSIZE", []string{"100"}},
		{"PRECISION bf16", 13, 14, false, "PRECISION", []string{"bf16"}},
		{"QUANTIZATION awq", 15, 16, false, "QUANTIZATION", []string{"awq"}},
		{"unknown command", 5, 6, true, "", nil},
	}

	assert := assert.New(t)
	for _, tc := range testCases {
		node, err := parseCommandLine(tc.line, tc.start, tc.end)
		if tc.expectErr {
			assert.Error(err)
			assert.Nil(node)
			continue
		}

		assert.NoError(err)
		assert.NotNil(node)
		assert.Equal(tc.cmd, node.GetValue())
		assert.Equal(tc.start, node.GetStartLine())
		assert.Equal(tc.end, node.GetEndLine())

		argsNode := node.GetNext()
		assert.NotNil(argsNode)

		args := []string{}
		for argsNode := node.GetNext(); argsNode != nil; argsNode = argsNode.GetNext() {
			args = append(args, argsNode.GetValue())
		}
		assert.Equal(tc.args, args)
	}
}

func TestSplitCommand(t *testing.T) {
	testCases := []struct {
		line      string
		expectErr bool
		cmd       string
		args      []string
	}{
		{"MODEL foo", false, "MODEL", []string{"foo"}},
		{"NAME bar", false, "NAME", []string{"bar"}},
		{"invalid", true, "", nil},
	}

	assert := assert.New(t)
	for _, tc := range testCases {
		cmd, args, err := splitCommand(tc.line)
		if tc.expectErr {
			assert.Error(err)
			assert.Empty(cmd)
			assert.Nil(args)
			continue
		}

		assert.NoError(err)
		assert.Equal(tc.cmd, cmd)
		assert.Equal(tc.args, args)
	}
}

// TestSplitCommandWithSpaces tests the splitCommand function with spaces and quotes
func TestSplitCommandWithSpaces(t *testing.T) {
	testCases := []struct {
		name         string
		line         string
		expectedCmd  string
		expectedArgs []string
		expectError  bool
	}{
		{
			name:         "simple command",
			line:         "MODEL model.bin",
			expectedCmd:  "MODEL",
			expectedArgs: []string{"model.bin"},
			expectError:  false,
		},
		{
			name:         "quoted argument with spaces",
			line:         "MODEL \"model weights.bin\"",
			expectedCmd:  "MODEL",
			expectedArgs: []string{"model weights.bin"},
			expectError:  false,
		},
		{
			name:         "quoted argument with path separators",
			line:         "CONFIG \"nested dir/config.json\"",
			expectedCmd:  "CONFIG",
			expectedArgs: []string{"nested dir/config.json"},
			expectError:  false,
		},
		{
			name:         "multiple quoted arguments",
			line:         "CODE \"script one.py\" \"script two.py\"",
			expectedCmd:  "CODE",
			expectedArgs: []string{"script one.py", "script two.py"},
			expectError:  false,
		},
		{
			name:         "mixed quoted and unquoted",
			line:         "DOC README.md \"user guide.pdf\"",
			expectedCmd:  "DOC",
			expectedArgs: []string{"README.md", "user guide.pdf"},
			expectError:  false,
		},
		{
			name:         "escaped quotes",
			line:         "NAME \"model \\\"v2\\\"\"",
			expectedCmd:  "NAME",
			expectedArgs: []string{"model \"v2\""},
			expectError:  false,
		},
		{
			name:        "unclosed quotes",
			line:        "MODEL \"unclosed",
			expectError: true,
		},
		{
			name:        "empty command",
			line:        "",
			expectError: true,
		},
		{
			name:        "no arguments",
			line:        "MODEL",
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cmd, args, err := splitCommand(tc.line)

			if tc.expectError {
				assert.Error(t, err, "Expected error for input: %s", tc.line)
			} else {
				assert.NoError(t, err, "Unexpected error for input: %s", tc.line)
				assert.Equal(t, tc.expectedCmd, cmd, "Command mismatch")
				assert.Equal(t, tc.expectedArgs, args, "Arguments mismatch")
			}
		})
	}
}

// TestParseArgsFunction tests the parseArgs function directly
func TestParseArgsFunction(t *testing.T) {
	testCases := []struct {
		name        string
		argsStr     string
		expected    []string
		expectError bool
	}{
		{
			name:        "single unquoted argument",
			argsStr:     "model.bin",
			expected:    []string{"model.bin"},
			expectError: false,
		},
		{
			name:        "single quoted argument",
			argsStr:     "\"model weights.bin\"",
			expected:    []string{"model weights.bin"},
			expectError: false,
		},
		{
			name:        "multiple mixed arguments",
			argsStr:     "model.bin \"config file.json\" script.py",
			expected:    []string{"model.bin", "config file.json", "script.py"},
			expectError: false,
		},
		{
			name:        "argument with tabs and spaces",
			argsStr:     "\"file\twith\ttabs\tand  spaces\"",
			expected:    []string{"file\twith\ttabs\tand  spaces"},
			expectError: false,
		},
		{
			name:        "escaped quotes",
			argsStr:     "\"file with \\\"quotes\\\" inside\"",
			expected:    []string{"file with \"quotes\" inside"},
			expectError: false,
		},
		{
			name:        "empty quoted string",
			argsStr:     "\"\"",
			expected:    []string{""},
			expectError: false,
		},
		{
			name:        "unclosed quotes",
			argsStr:     "\"unclosed quote",
			expectError: true,
		},
		{
			name:        "multiple spaces between args",
			argsStr:     "arg1    arg2     \"arg 3\"",
			expected:    []string{"arg1", "arg2", "arg 3"},
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := parseArgs(tc.argsStr)

			if tc.expectError {
				assert.Error(t, err, "Expected error for input: %s", tc.argsStr)
			} else {
				assert.NoError(t, err, "Unexpected error for input: %s", tc.argsStr)
				assert.Equal(t, tc.expected, result, "Result mismatch")
			}
		})
	}
}
