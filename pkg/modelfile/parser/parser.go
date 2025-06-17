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
	"errors"
	"fmt"
	"io"
	"strings"
	"unicode"

	"github.com/CloudNativeAI/modctl/pkg/modelfile/command"
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
	cmd, args, flags, err := splitCommand(line)
	if err != nil {
		return nil, err
	}

	switch cmd {
	case command.CONFIG, command.DOC, command.DATASET, command.NAME, command.ARCH, command.FAMILY, command.FORMAT, command.PARAMSIZE, command.PRECISION, command.QUANTIZATION:
		if len(args) != 1 {
			return nil, errors.New("command " + cmd + " requires exactly one argument")
		}

		argsNode, err := parseStringArgs(args, start, end)
		if err != nil {
			return nil, err
		}
		cmdNode := NewNode(cmd, start, end)
		cmdNode.AddNext(argsNode)

		return cmdNode, nil
	case command.MODEL, command.CODE:
		argsNode, err := parseStringArgs(args, start, end)
		if err != nil {
			return nil, err
		}

		cmdNode := NewNode(cmd, start, end)
		cmdNode.AddNext(argsNode)

		// Add flags as attributes if any exist
		if len(flags) > 0 {
			for _, flag := range flags {
				// Parse the flag to get key and value
				key, value := parseFlagKeyValue(flag)
				if key != "" {
					cmdNode.AddAttribute(key, value)
				}
			}
		}
		return cmdNode, nil
	default:
		return nil, fmt.Errorf("invalid command: %s", cmd)
	}
}

// parseFlagKeyValue parses a flag string and returns the key and value
// Example: "key=value" returns "key", "value"
// Example: "org.cnai.model.file.mediatype.untested=true" returns "org.cnai.model.file.mediatype.untested", "true"
func parseFlagKeyValue(flag string) (string, string) {
	// For flags that are just values (like "key=value" from "--label=key=value"),
	// we need to determine if this is already a key=value pair or if we need to add a prefix
	if idx := strings.Index(flag, "="); idx != -1 {
		return flag[:idx], flag[idx+1:]
	}

	// If no "=" found, treat the whole thing as a key with empty value
	return flag, ""
}

// splitCommand splits the command line into the command, args, and flags. Returns the
// command, the args, the flags, and an error if the command line is invalid.
// Example: "MODEL --label=key=value /home/user/model.safetensors" returns "MODEL", ["/home/user/model.safetensors"], ["key=value"], nil.
func splitCommand(line string) (string, []string, []string, error) {
	parts := strings.Fields(line)
	if len(parts) < 2 {
		return "", nil, nil, fmt.Errorf("invalid command line: %s", line)
	}

	cmd := strings.ToUpper(parts[0])

	// Extract flags and remaining args from the rest of the line
	restOfLine := strings.TrimSpace(line[len(parts[0]):])
	remaining, flags, err := extractCommandFlags(restOfLine)
	if err != nil {
		return "", nil, nil, err
	}

	// Parse remaining content as args
	var args []string
	if remaining != "" {
		args = strings.Fields(remaining)
	}

	return cmd, args, flags, nil
}

// extractCommandFlags parses the command flags and returns the remaining part of the line
// and the command flags (with values only, without the flag names).
func extractCommandFlags(line string) (string, []string, error) {
	flags := []string{}
	var i int

	// Skip leading spaces and process flags
	for i < len(line) {
		// Skip spaces
		for i < len(line) && unicode.IsSpace(rune(line[i])) {
			i++
		}

		// Check if we've reached the end or found a non-flag
		if i >= len(line) || !isFlag(line, i) {
			break
		}

		// Extract the flag
		start := i
		for i < len(line) && !unicode.IsSpace(rune(line[i])) {
			i++
		}

		flag := line[start:i]
		if flag == "--" {
			// Double dash terminator, return remaining content starting from the space after "--"
			return line[start+2:], flags, nil
		}

		if flag != "" {
			// Extract the value part from --flag=value format
			flagValue := extractFlagValue(flag)
			if flagValue != "" {
				flags = append(flags, flagValue)
			}
		}
	}

	// Return remaining content after flags
	return line[i:], flags, nil
}

// isFlag checks if the content at position i starts with "--"
func isFlag(line string, i int) bool {
	return i+1 < len(line) && line[i] == '-' && line[i+1] == '-'
}

// extractFlagValue extracts the value from a flag string
// Example: "--label=key=value" returns "key=value"
// Example: "--untested" returns ""
func extractFlagValue(flag string) string {
	// Remove the leading "--"
	if strings.HasPrefix(flag, "--") {
		flag = flag[2:]
	}

	// Find the first "=" to get the value part
	if idx := strings.Index(flag, "="); idx != -1 {
		return flag[idx+1:]
	}

	// If no "=" found, return empty string
	return ""
}
