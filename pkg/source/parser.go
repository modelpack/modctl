/*
 *     Copyright 2025 The CNAI Authors
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

package source

import "fmt"

const (
	// ParserTypeGit is the type of parser for git repositories.
	ParserTypeGit = "git"

	// ParserTypeZeta is the type of parser for zeta repositories.
	ParserTypeZeta = "zeta"
)

// Parser is an interface for parsing the source information.
type Parser interface {
	Parse(workspace string) (*Info, error)
}

// Info is a struct that holds the source information.
type Info struct {
	// URL is the URL of the source.
	// e.g git is the origin remote URL.
	URL string
	// Commit is the commit hash of the source.
	// e.g git is the HEAD commit hash.
	Commit string
	// Dirty is true if the source is dirty.
	// e.g git is indicating whether the workspace is dirty.
	Dirty bool
}

func NewParser(typ string) (Parser, error) {
	switch typ {
	case ParserTypeGit:
		return &git{}, nil
	case ParserTypeZeta:
		return &zeta{}, nil
	default:
		return nil, fmt.Errorf("unsupported parser type: %s", typ)
	}
}
