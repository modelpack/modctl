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

package build

import (
	"archive/tar"
	"fmt"
	"io"
	"os"
)

// TarFileToStream tars the target file and return the content by stream.
func TarFileToStream(path string) (io.Reader, error) {
	pr, pw := io.Pipe()
	go func() {
		defer pw.Close()
		// create the tar writer.
		tw := tar.NewWriter(pw)
		defer tw.Close()

		file, err := os.Open(path)
		if err != nil {
			pw.CloseWithError(fmt.Errorf("failed to open file: %w", err))
			return
		}

		defer file.Close()
		info, err := file.Stat()
		if err != nil {
			pw.CloseWithError(fmt.Errorf("failed to stat file: %w", err))
			return
		}

		header, err := tar.FileInfoHeader(info, info.Name())
		if err != nil {
			pw.CloseWithError(fmt.Errorf("failed to create tar file info header: %w", err))
			return
		}

		if err := tw.WriteHeader(header); err != nil {
			pw.CloseWithError(fmt.Errorf("failed to write header to tar writer: %w", err))
			return
		}

		_, err = io.Copy(tw, file)
		if err != nil {
			pw.CloseWithError(fmt.Errorf("failed to copy file to tar writer: %w", err))
			return
		}
	}()

	return pr, nil
}
