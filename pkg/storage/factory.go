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

package storage

import (
	"os/user"
	"path/filepath"

	"github.com/CloudNativeAI/modctl/pkg/storage/distribution"
)

const (
	// contentV1Dir is the content v1 directory.
	contentV1Dir = "/.modctl/content.v1/"
)

// Type is the type of storage.
type Type = string

// New gets the storage by the type.
func New(storageType Type, opts ...Option) (Storage, error) {
	storageOpts := &Options{}
	for _, opt := range opts {
		opt(storageOpts)
	}
	// apply default option if not set.
	if storageOpts.RootDir == "" {
		contentDir, err := GetDefaultContentDir()
		if err != nil {
			return nil, err
		}

		storageOpts.RootDir = contentDir
	}

	switch storageType {
	case distribution.StorageTypeDistribution:
		return distribution.NewStorage(storageOpts.RootDir)
	// extend more storage types here.
	// case "other":
	default:
		//  currently by default we are using distribution as storage.
		return distribution.NewStorage(storageOpts.RootDir)
	}
}

// getHomeDir returns the current user's home directory.
func getHomeDir() (string, error) {
	usr, err := user.Current()
	if err != nil {
		return "", err
	}

	return usr.HomeDir, nil
}

// GetDefaultContentDir returns the default content directory.
func GetDefaultContentDir() (string, error) {
	homeDir, err := getHomeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(homeDir, contentV1Dir), nil
}
