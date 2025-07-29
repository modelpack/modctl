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
	"encoding/json"
	"io"
	"os"
	"path/filepath"

	legacymodelspec "github.com/dragonflyoss/model-spec/specs-go/v1"
	modelspec "github.com/modelpack/model-spec/specs-go/v1"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

// raw is a codec that for raw files.
type raw struct{}

// newRaw creates a new raw codec instance.
func newRaw() *raw {
	return &raw{}
}

// Type returns the type of the codec.
func (r *raw) Type() string {
	return Raw
}

// Encode reads the target file into a reader.
func (r *raw) Encode(targetFilePath, workDirPath string) (io.Reader, error) {
	return os.Open(targetFilePath)
}

// Decode reads the input reader and decodes the data into the output path.
func (r *raw) Decode(outputDir, filePath string, reader io.Reader, desc ocispec.Descriptor) error {
	fullPath := filepath.Join(outputDir, filePath)
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	file, err := os.Create(fullPath)
	if err != nil {
		return err
	}
	defer file.Close()

	if _, err := io.Copy(file, reader); err != nil {
		return err
	}

	var fileMetadata *modelspec.FileMetadata
	// Try to retrieve the file metadata from annotation for raw file.
	if desc.Annotations != nil {
		fileMetadataStr := desc.Annotations[modelspec.AnnotationFileMetadata]
		if fileMetadataStr == "" {
			fileMetadataStr = desc.Annotations[legacymodelspec.AnnotationFileMetadata]
		}

		if fileMetadataStr != "" {
			if err := json.Unmarshal([]byte(fileMetadataStr), &fileMetadata); err != nil {
				return err
			}
		}
	}

	// Restore file metadata if available.
	if fileMetadata != nil {
		// Restore file mode (convert from decimal to octal).
		if fileMetadata.Mode != 0 {
			if err := file.Chmod(os.FileMode(fileMetadata.Mode)); err != nil {
				return err
			}
		}
	}

	// Restore modification time if available.
	if fileMetadata != nil && !fileMetadata.ModTime.IsZero() {
		if err := os.Chtimes(fullPath, fileMetadata.ModTime, fileMetadata.ModTime); err != nil {
			return err
		}
	}

	return nil
}
