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
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/modelpack/modctl/pkg/modelfile/command"
)

// Parse parses the modelfile and returns the root node of the AST,
// and the root node is the entry point of the AST. Walk the AST to
// get the information of the modelfile.
func Parse(reader io.Reader) (Node, error) {
	root := NewRootNode()
	currentLine := 0

	scanner := bufio.NewScanner(reader)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		bytes := scanner.Bytes()
		trimmedLine := strings.TrimSpace(string(bytes))

		// If the line is empty, continue to the next line.
		if isEmptyContinuationLine(trimmedLine) {
			currentLine++
			continue
		}

		// If the line is a comment, do not to record it and
		// continue to the next line.
		if isComment(trimmedLine) {
			currentLine++
			continue
		}

		// If the line is a command, parse the command line, and add
		// the command node and the args node to the root node.
		if isCommand(trimmedLine) {
			node, err := parseCommandLine(trimmedLine, currentLine, currentLine)
			if err != nil {
				return nil, fmt.Errorf("parse command line error on line %d: %w", currentLine, err)
			}

			root.AddChild(node)
			currentLine++
			continue
		}

		// If the line is not a comment, empty continuation, or a command, return an error.
		return nil, fmt.Errorf("parse error on line %d: %s", currentLine, string(bytes))
	}

	return root, nil
}

// isComment checks if the line is a comment.
func isComment(line string) bool {
	return strings.HasPrefix(line, "#")
}

// isCommand checks if the line is a command.
func isCommand(line string) bool {
	line = strings.ToUpper(line)
	for _, cmd := range command.Commands {
		if strings.HasPrefix(line, cmd) {
			return true
		}
	}

	return false
}

// isEmptyContinuationLine checks if the line is an empty continuation line.
func isEmptyContinuationLine(line string) bool {
	return len(line) == 0
}

// parseCommandLine parses the command line and returns the command node with the args node.
// Need to walk the next node of the command node to get the args node.
func parseCommandLine(line string, start, end int) (Node, error) {
	cmd, args, err := splitCommand(line)
	if err != nil {
		return nil, err
	}

	switch cmd {
	case command.CONFIG, command.MODEL, command.CODE, command.DATASET, command.DOC, command.NAME, command.ARCH, command.FAMILY, command.FORMAT, command.PARAMSIZE, command.PRECISION, command.QUANTIZATION:
		argsNode, err := parseStringArgs(args, start, end)
		if err != nil {
			return nil, err
		}

		cmdNode := NewNode(cmd, start, end)
		cmdNode.AddNext(argsNode)
		return cmdNode, nil
	default:
		return nil, fmt.Errorf("invalid command: %s", cmd)
	}
}

// splitCommand splits the command line into the command and the args. Returns the
// command and the args, and an error if the command line is invalid.
// Example: "MODEL foo" returns "MODEL", ["foo"] and nil.
// Example: "MODEL \"foo bar\"" returns "MODEL", ["foo bar"] and nil.
func splitCommand(line string) (string, []string, error) {
	// First, split to get the command
	firstSpace := strings.Index(line, " ")
	if firstSpace == -1 {
		return "", nil, fmt.Errorf("invalid command line: %s", line)
	}

	command := strings.ToUpper(strings.TrimSpace(line[:firstSpace]))
	argsStr := strings.TrimSpace(line[firstSpace+1:])

	if argsStr == "" {
		return "", nil, fmt.Errorf("invalid command line: %s", line)
	}

	// Parse the arguments, handling quoted strings
	args, err := parseArgs(argsStr)
	if err != nil {
		return "", nil, fmt.Errorf("invalid args in command line: %s", line)
	}

	return command, args, nil
}

// parseArgs parses argument string, handling quoted strings with spaces
func parseArgs(argsStr string) ([]string, error) {
	var args []string
	var current strings.Builder
	var inQuotes bool
	var escape bool
	var hadQuotes bool // Track if we've seen quotes for the current argument

	for _, r := range argsStr {
		switch {
		case escape:
			// Previous character was escape, add this character literally
			current.WriteRune(r)
			escape = false
		case r == '\\':
			// Escape next character
			escape = true
		case r == '"':
			// Toggle quote mode
			inQuotes = !inQuotes
			hadQuotes = true
		case r == ' ' || r == '\t':
			if inQuotes {
				// Inside quotes, preserve the space
				current.WriteRune(r)
			} else {
				// Outside quotes, this ends the current argument
				if current.Len() > 0 || hadQuotes {
					args = append(args, current.String())
					current.Reset()
					hadQuotes = false
				}
			}
		default:
			current.WriteRune(r)
		}
	}

	// Check for unclosed quotes
	if inQuotes {
		return nil, fmt.Errorf("unclosed quotes in arguments")
	}

	// Add the final argument if any
	if current.Len() > 0 || hadQuotes {
		args = append(args, current.String())
	}

	return args, nil
}
