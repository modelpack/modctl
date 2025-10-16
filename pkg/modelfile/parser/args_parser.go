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
	"errors"
	"strings"
)

// parseStringArgs parses the string type of args and returns a Node, for example:
// "MODEL foo" args' value is "foo".
// If multiple args are provided (due to unquoted spaces), they are joined with spaces.
// This handles cases like: CONFIG path with spaces/file.json
func parseStringArgs(args []string, start, end int) (Node, error) {
	if len(args) == 0 {
		return nil, errors.New("empty args")
	}

	// Join all arguments with spaces to handle unquoted file paths with spaces
	joined := strings.Join(args, " ")

	if strings.TrimSpace(joined) == "" {
		return nil, errors.New("empty args")
	}

	return NewNode(joined, start, end), nil
}
