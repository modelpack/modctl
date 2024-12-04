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

package archiver

import (
	"archive/tar"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Tar tars the target file and return the content by stream.
func Tar(path string) (io.Reader, error) {
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

// Untar untars the target stream to the destination path.
func Untar(reader io.Reader, destPath string) error {
	// uncompress gzip if it is a .tar.gz file
	// gzipReader, err := gzip.NewReader(reader)
	// if err != nil {
	//     return err
	// }
	// defer gzipReader.Close()
	// tarReader := tar.NewReader(gzipReader)

	tarReader := tar.NewReader(reader)

	if err := os.MkdirAll(destPath, 0755); err != nil {
		return err
	}

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		// sanitize filepaths to prevent directory traversal.
		cleanPath := filepath.Clean(header.Name)
		if strings.Contains(cleanPath, "..") {
			return fmt.Errorf("tar file contains invalid path: %s", cleanPath)
		}

		path := filepath.Join(destPath, cleanPath)
		// check the file type.
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(path, 0755); err != nil {
				return err
			}
		case tar.TypeReg:
			file, err := os.Create(path)
			if err != nil {
				return err
			}

			if _, err := io.Copy(file, tarReader); err != nil {
				file.Close()
				return err
			}
			file.Close()

			if err := os.Chmod(path, os.FileMode(header.Mode)); err != nil {
				return err
			}
		}
	}

	return nil
}
