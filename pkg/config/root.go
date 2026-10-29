/*
 *     Copyright 2024 The ModelPack Authors
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
	"os"
	"os/user"
	"path/filepath"
)

type Root struct {
	StorageDir      string
	Pprof           bool
	PprofAddr       string
	DisableProgress bool
	LogDir          string
	LogLevel        string
}

func NewRoot() (*Root, error) {
	homeDir, err := userHomeDir()
	if err != nil {
		return nil, err
	}

	return &Root{
		StorageDir:      filepath.Join(homeDir, ".modctl"),
		Pprof:           false,
		PprofAddr:       "localhost:6060",
		DisableProgress: false,
		LogDir:          filepath.Join(homeDir, ".modctl/logs"),
		LogLevel:        "info",
	}, nil
}

// userHomeDir returns the home directory of the current user. It prefers
// os.UserHomeDir, which honors $HOME (%USERPROFILE% on Windows), so that an
// overridden home directory is respected. os/user.Current queries the user
// database, which may ignore $HOME, so it is only used as a fallback when the
// environment variable is not set.
func userHomeDir() (string, error) {
	if dir, err := os.UserHomeDir(); err == nil {
		return dir, nil
	}

	u, err := user.Current()
	if err != nil {
		return "", err
	}

	return u.HomeDir, nil
}
