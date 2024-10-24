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

func TestNewRootNode(t *testing.T) {
	root := NewRootNode()

	assert := assert.New(t)
	assert.Equal("", root.GetValue())
	assert.Nil(root.GetNext())
	assert.Empty(root.GetChildren())
	assert.Equal(0, root.GetStartLine())
	assert.Equal(0, root.GetEndLine())
	assert.Nil(root.GetAttributes())
}

func TestNewNode(t *testing.T) {
	testCases := []struct {
		value     string
		startLine int
		endLine   int
	}{
		{"test1", 1, 2},
		{"test2", 3, 4},
		{"test3", 5, 6},
	}

	assert := assert.New(t)
	for _, tc := range testCases {
		node := NewNode(tc.value, tc.startLine, tc.endLine)

		assert.Equal(tc.value, node.GetValue(), "expected value %s", tc.value)
		assert.Nil(node.GetNext(), "expected nil next node")
		assert.Empty(node.GetChildren(), "expected no children")
		assert.Equal(tc.startLine, node.GetStartLine(), "expected start line %d", tc.startLine)
		assert.Equal(tc.endLine, node.GetEndLine(), "expected end line %d", tc.endLine)
		assert.Nil(node.GetAttributes(), "expected nil attributes")
	}
}

func TestAddChild(t *testing.T) {
	parent := NewNode("parent", 1, 2)
	children := []Node{
		NewNode("child1", 3, 4),
		NewNode("child2", 5, 6),
		NewNode("child3", 7, 8),
	}

	for _, child := range children {
		parent.AddChild(child)
	}

	assert := assert.New(t)
	assert.Len(parent.GetChildren(), len(children))
	for i, child := range children {
		assert.Equal(child, parent.GetChildren()[i])
	}
}

func TestAddNext(t *testing.T) {
	node1 := NewNode("node1", 1, 2)
	node2 := NewNode("node2", 3, 4)
	node1.AddNext(node2)

	assert := assert.New(t)
	assert.Equal(node2, node1.GetNext())
}

func TestAddAttribute(t *testing.T) {
	node := NewNode("node", 1, 2)
	attributes := map[string]string{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	}

	for key, value := range attributes {
		node.AddAttribute(key, value)
	}

	assert := assert.New(t)
	assert.Equal(attributes, node.GetAttributes())
}
