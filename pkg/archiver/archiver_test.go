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
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestTar(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "archiver_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	filePath := filepath.Join(tmpDir, "testfile.txt")
	if err := os.WriteFile(filePath, []byte("hello"), 0644); err != nil {
		t.Fatalf("write file error: %v", err)
	}

	tarReader, err := Tar(filePath, tmpDir)
	if err != nil {
		t.Fatalf("Tar error: %v", err)
	}

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, tarReader); err != nil {
		t.Fatalf("copy tar error: %v", err)
	}

	if buf.Len() == 0 {
		t.Fatal("tar archive is empty")
	}
}

func TestUntar(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "archiver_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	filePath := filepath.Join(tmpDir, "testfile.txt")
	if err := os.WriteFile(filePath, []byte("hello"), 0644); err != nil {
		t.Fatalf("write file error: %v", err)
	}

	tarReader, err := Tar(filePath, tmpDir)
	if err != nil {
		t.Fatalf("Tar error: %v", err)
	}

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, tarReader); err != nil {
		t.Fatalf("copy tar error: %v", err)
	}

	extractDir, err := os.MkdirTemp("", "archiver_extracted")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(extractDir)

	if err := Untar(bytes.NewReader(buf.Bytes()), extractDir); err != nil {
		t.Fatalf("Untar error: %v", err)
	}

	extractedFile := filepath.Join(extractDir, filepath.Base(filePath))
	data, err := os.ReadFile(extractedFile)
	if err != nil {
		t.Fatalf("read extracted file error: %v", err)
	}

	if string(data) != "hello" {
		t.Errorf("expected 'hello', got '%s'", string(data))
	}
}
