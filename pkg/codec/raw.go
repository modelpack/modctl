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
	"os"
	"path/filepath"
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
func (r *raw) Decode(reader io.Reader, outputDir, filePath string) error {
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

	return nil
}
