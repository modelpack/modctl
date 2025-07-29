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
	"path/filepath"

	"github.com/modelpack/modctl/pkg/storage/distribution"
)

const (
	// contentV1Dir is the content v1 directory.
	contentV1Dir = "content.v1"
)

// Type is the type of storage.
type Type = string

// New gets the storage by the type.
func New(storageType Type, storageDir string, opts ...Option) (Storage, error) {
	storageOpts := &Options{}
	for _, opt := range opts {
		opt(storageOpts)
	}

	storageOpts.RootDir = filepath.Join(storageDir, contentV1Dir)
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
