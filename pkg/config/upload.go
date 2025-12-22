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

package config

import (
	"errors"
	"fmt"
	"path/filepath"
)

type Upload struct {
	Repo           string
	PlainHTTP      bool
	Insecure       bool
	Raw            bool
	DestinationDir string
}

func NewUpload() *Upload {
	return &Upload{
		Repo:           "",
		PlainHTTP:      false,
		Insecure:       false,
		Raw:            false,
		DestinationDir: "",
	}
}

func (u *Upload) Validate() error {
	if u.Repo == "" {
		return errors.New("repo is required")
	}

	// Check if destination directory is relative path.
	if u.DestinationDir != "" {
		if filepath.IsAbs(u.DestinationDir) {
			return fmt.Errorf("destination directory must be relative path")
		}
	}

	return nil
}
