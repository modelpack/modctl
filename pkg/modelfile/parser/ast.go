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

// Node is the interface for the AST node.
//
// Walk the AST to get the information by the root node.
// The root node is the entry point of the AST. Use the Node
// interface to build the AST for the modelfile.
type Node interface {
	// GetNext returns the next node of the current node.
	GetNext() Node

	// GetChildren returns the children nodes of the current node.
	GetChildren() []Node

	// GetValue returns the value of the current node, the value is
	// the command or the args of the line by parsing the modelfile.
	GetValue() string

	// GetAttributes returns the attributes of the current node, such as key-value pairs.
	// Attributes can be used to expand the information of the node.
	GetAttributes() map[string]string

	// GetStartLine returns the start line of the current node, it will help to
	// locate the start of the node.
	GetStartLine() int

	// GetEndLine returns the end line of the current node, it will help to
	// locate the end of the node.
	GetEndLine() int

	// AddChild adds a child node to the current node.
	AddChild(child Node)

	// AddNext adds a next node to the current node.
	AddNext(next Node)

	// AddAttribute adds an attribute to the current node.
	AddAttribute(key, value string)
}

// node is the implementation of the Node interface.
type node struct {
	// value is the value of the node, it is the command or the args of the line.
	value string

	// Next is the next node of the current node.
	next Node

	// children is the children nodes of the current node.
	children []Node

	// startLine is the start line of the current node.
	startLine int

	// endLine is the end line of the current node.
	endLine int

	// attributes is the attributes of the current node.
	attributes map[string]string
}

// NewRootNode creates a new root node of the AST. It is used as the entry
// point of the AST, so do not use it to create the node of the AST.
func NewRootNode() Node {
	return &node{
		value:      "",
		next:       nil,
		children:   nil,
		startLine:  0,
		endLine:    0,
		attributes: nil,
	}
}

// NewNode creates a new node of the AST with the value, start line, and end line.
// It is used to create the node of the AST, so need to create the node with the
// value by parsing the modelfile.
func NewNode(value string, start, end int) Node {
	return &node{
		value:      value,
		next:       nil,
		children:   nil,
		startLine:  start,
		endLine:    end,
		attributes: nil,
	}
}

// GetNext returns the next node of the current node.
func (n *node) GetNext() Node {
	return n.next
}

// GetChildren returns the children nodes of the current node.
func (n *node) GetChildren() []Node {
	return n.children
}

// GetValue returns the value of the current node, the value is
// the command or the args of the line by parsing the modelfile.
func (n *node) GetValue() string {
	return n.value
}

// GetStartLine returns the start line of the current node, it will help to
// locate the start of the node.
func (n *node) GetStartLine() int {
	return n.startLine
}

// GetEndLine returns the end line of the current node, it will help to
// locate the end of the node.
func (n *node) GetEndLine() int {
	return n.endLine
}

// GetAttributes returns the attributes of the current node, such as key-value pairs.
// Attributes can be used to expand the information of the node.
func (n *node) GetAttributes() map[string]string {
	return n.attributes
}

// AddChild adds a child node to the current node.
func (n *node) AddChild(child Node) {
	n.children = append(n.children, child)
}

// AddNext adds a next node to the current node.
func (n *node) AddNext(next Node) {
	n.next = next
}

// AddAttribute adds an attribute to the current node.
func (n *node) AddAttribute(key, value string) {
	if n.attributes == nil {
		n.attributes = make(map[string]string)
	}

	n.attributes[key] = value
}
