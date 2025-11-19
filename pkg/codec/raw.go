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
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"

	legacymodelspec "github.com/dragonflyoss/model-spec/specs-go/v1"
	modelspec "github.com/modelpack/model-spec/specs-go/v1"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/sirupsen/logrus"

	"github.com/modelpack/modctl/pkg/xattr"
)

// ErrAlreadyUpToDate is returned when the target output already matches the descriptor metadata.
var ErrAlreadyUpToDate = errors.New("codec: target already up-to-date")

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

// fileNeedsUpdate checks if the file exists and whether its size and digest match.
// Returns true if the file needs to be updated/written, false if it can be skipped.
func (r *raw) fileNeedsUpdate(fullPath string, desc ocispec.Descriptor) (bool, error) {
	// Check if file exists.
	info, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist, needs to be written.
			return true, nil
		}

		// Other error occurred.
		return true, err
	}

	// File exists, check size first (quick check).
	if info.Size() != desc.Size {
		return true, nil
	}

	// Check xattrs for stored size and digest.
	sizeKey := xattr.MakeKey(xattr.KeySize)
	storedSize, err := xattr.Get(fullPath, sizeKey)
	if err != nil {
		// xattr not found or error reading, needs update.
		return true, nil
	}

	digestKey := xattr.MakeKey(xattr.KeySha256)
	storedDigest, err := xattr.Get(fullPath, digestKey)
	if err != nil {
		// xattr not found or error reading, needs update.
		return true, nil
	}

	// Compare stored values with descriptor.
	expectedSize := strconv.FormatInt(desc.Size, 10)
	expectedDigest := desc.Digest.String()

	if string(storedSize) == expectedSize && string(storedDigest) == expectedDigest {
		// File is up-to-date, no need to write.
		logrus.Debugf("file %s is up-to-date", fullPath)
		return false, nil
	}

	// Values don't match, needs update.
	return true, nil
}

// storeFileMetadata stores the size and digest in xattrs.
func (r *raw) storeFileMetadata(fullPath string, desc ocispec.Descriptor) error {
	sizeKey := xattr.MakeKey(xattr.KeySize)
	if err := xattr.Set(fullPath, sizeKey, []byte(strconv.FormatInt(desc.Size, 10))); err != nil {
		return fmt.Errorf("failed to set size xattr: %w", err)
	}

	digestKey := xattr.MakeKey(xattr.KeySha256)
	if err := xattr.Set(fullPath, digestKey, []byte(desc.Digest.String())); err != nil {
		return fmt.Errorf("failed to set digest xattr: %w", err)
	}

	return nil
}

// Decode reads the input reader and decodes the data into the output path.
func (r *raw) Decode(outputDir, filePath string, reader io.Reader, desc ocispec.Descriptor) error {
	fullPath := filepath.Join(outputDir, filePath)
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Check if file needs update.
	needsUpdate, err := r.fileNeedsUpdate(fullPath, desc)
	if err != nil {
		logrus.Errorf("failed to check whether the file %s needs to be updated: %s", fullPath, err)
		needsUpdate = true
	}

	if !needsUpdate {
		return ErrAlreadyUpToDate
	}

	// File needs to be written/updated.
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

	// Store size and digest in xattrs after successful write.
	// Ignore errors as xattrs might not be supported on all filesystems.
	if err := r.storeFileMetadata(fullPath, desc); err != nil {
		logrus.Errorf("failed to store file metadata of %s: %s", fullPath, err)
	}

	return nil
}
