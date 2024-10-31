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

package processor

import (
	"context"
	"os"

	"github.com/CloudNativeAI/modctl/pkg/storage"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

// Processor is the interface to recognize and process the identified file.
type Processor interface {
	// Name returns the name of the processor.
	Name() string
	// Identify identifies the file, returns true if the file is identified,
	// then the Process will be called to process the file, otherwise it will be skipped.
	Identify(ctx context.Context, path string, info os.FileInfo) bool
	// Process processes the file.
	Process(ctx context.Context, store storage.Storage, repo, path string, info os.FileInfo) (ocispec.Descriptor, error)
}
