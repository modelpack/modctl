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

package codec

import (
	"io"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"

	"github.com/modelpack/modctl/pkg/archiver"
)

// tar is a codec for tar files.
type tar struct{}

// newTar creates a new tar codec instance.
func newTar() *tar {
	return &tar{}
}

// Type returns the type of the codec.
func (t *tar) Type() string {
	return Tar
}

// Encode tars the target file into a reader.
func (t *tar) Encode(targetFilePath, workDirPath string) (io.Reader, error) {
	return archiver.Tar(targetFilePath, workDirPath)
}

// Decode reads the input reader and decodes the data into the output path.
func (t *tar) Decode(outputDir, filePath string, reader io.Reader, desc ocispec.Descriptor) error {
	// As the file name has been provided in the tar header,
	// so we do not care about the filePath.
	return archiver.Untar(reader, outputDir)
}
