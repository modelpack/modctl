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
	"fmt"
	"strings"
	"testing"

	modelspec "github.com/CloudNativeAI/model-spec/specs-go/v1"
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
		flags     []string
	}{
		{"MODEL foo", false, "MODEL", []string{"foo"}, []string{}},
		{"NAME bar", false, "NAME", []string{"bar"}, []string{}},
		{"MODEL --label=key=value /home/user/model.safetensors", false, "MODEL", []string{"/home/user/model.safetensors"}, []string{"key=value"}},
		{"MODEL --untested --experimental=test model.safetensors", false, "MODEL", []string{"model.safetensors"}, []string{"test"}},
		{"CONFIG --format=json config.yaml", false, "CONFIG", []string{"config.yaml"}, []string{"json"}},
		{"MODEL --flag1=value1 --flag2=value2 model.bin", false, "MODEL", []string{"model.bin"}, []string{"value1", "value2"}},
		{"MODEL --untested model.safetensors", false, "MODEL", []string{"model.safetensors"}, []string{}}, // flag without value
		{"invalid", true, "", nil, nil},
	}

	assert := assert.New(t)
	for _, tc := range testCases {
		cmd, args, flags, err := splitCommand(tc.line)
		if tc.expectErr {
			assert.Error(err)
			assert.Empty(cmd)
			assert.Nil(args)
			assert.Nil(flags)
			continue
		}

		assert.NoError(err)
		assert.Equal(tc.cmd, cmd)
		assert.Equal(tc.args, args)
		assert.Equal(tc.flags, flags)
	}
}

func TestExtractFlagValue(t *testing.T) {
	testCases := []struct {
		name          string
		flag          string
		expectedValue string
	}{
		{
			name:          "flag with key=value",
			flag:          "--label=key=value",
			expectedValue: "key=value",
		},
		{
			name:          "flag with simple value",
			flag:          "--format=json",
			expectedValue: "json",
		},
		{
			name:          "flag without value",
			flag:          "--untested",
			expectedValue: "",
		},
		{
			name:          "flag with empty value",
			flag:          "--label=",
			expectedValue: "",
		},
		{
			name:          "complex flag value",
			flag:          fmt.Sprintf("--label=%s=true", modelspec.AnnotationMediaTypeUntested),
			expectedValue: fmt.Sprintf("%s=true", modelspec.AnnotationMediaTypeUntested),
		},
	}

	assert := assert.New(t)
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			value := extractFlagValue(tc.flag)
			assert.Equal(tc.expectedValue, value, "Value mismatch for test case: %s", tc.name)
		})
	}
}
