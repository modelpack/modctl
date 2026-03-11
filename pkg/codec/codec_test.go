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
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Raw Codec Tests ---

func TestRawEncodeDecode(t *testing.T) {
	dir := t.TempDir()
	content := []byte("hello world raw codec test")

	// Write source file.
	srcPath := filepath.Join(dir, "input.bin")
	require.NoError(t, os.WriteFile(srcPath, content, 0644))

	r := newRaw()

	// Encode: should return a reader with the file's content.
	reader, err := r.Encode(srcPath, dir)
	require.NoError(t, err)

	encoded, err := io.ReadAll(reader)
	require.NoError(t, err)
	assert.Equal(t, content, encoded)

	// Decode: write the encoded bytes to an output directory.
	outputDir := filepath.Join(dir, "output")
	require.NoError(t, os.MkdirAll(outputDir, 0755))

	desc := ocispec.Descriptor{Size: int64(len(content))}
	err = r.Decode(outputDir, "decoded.bin", bytes.NewReader(encoded), desc)
	require.NoError(t, err)

	decoded, err := os.ReadFile(filepath.Join(outputDir, "decoded.bin"))
	require.NoError(t, err)
	assert.Equal(t, content, decoded)
}

func TestRawEncodeEmpty(t *testing.T) {
	dir := t.TempDir()
	content := []byte{}

	srcPath := filepath.Join(dir, "empty.bin")
	require.NoError(t, os.WriteFile(srcPath, content, 0644))

	r := newRaw()

	reader, err := r.Encode(srcPath, dir)
	require.NoError(t, err)

	encoded, err := io.ReadAll(reader)
	require.NoError(t, err)
	assert.Empty(t, encoded)
}

func TestRawDecodeInvalidInput(t *testing.T) {
	dir := t.TempDir()
	r := newRaw()

	// Decode from a reader that always errors; the error should propagate.
	badReader := &errorReader{}
	desc := ocispec.Descriptor{Size: 10}

	err := r.Decode(dir, "out.bin", badReader, desc)
	assert.Error(t, err)
}

// errorReader is a reader that always returns an error.
type errorReader struct{}

func (e *errorReader) Read([]byte) (int, error) {
	return 0, io.ErrUnexpectedEOF
}

// --- Tar Codec Tests ---

func TestTarArchiveSingleFile(t *testing.T) {
	srcDir := t.TempDir()
	content := []byte("single file content")

	filePath := filepath.Join(srcDir, "file.txt")
	require.NoError(t, os.WriteFile(filePath, content, 0644))

	c := newTar()

	reader, err := c.Encode(filePath, srcDir)
	require.NoError(t, err)

	// Read the tar stream fully so it can be used for extraction.
	tarData, err := io.ReadAll(reader)
	require.NoError(t, err)
	assert.NotEmpty(t, tarData)

	// Extract and verify.
	extractDir := t.TempDir()
	desc := ocispec.Descriptor{}
	err = c.Decode(extractDir, "file.txt", bytes.NewReader(tarData), desc)
	require.NoError(t, err)

	extracted, err := os.ReadFile(filepath.Join(extractDir, "file.txt"))
	require.NoError(t, err)
	assert.Equal(t, content, extracted)
}

func TestTarArchiveMultipleFiles(t *testing.T) {
	srcDir := t.TempDir()

	files := map[string]string{
		"a.txt":     "content of a",
		"b.txt":     "content of b",
		"sub/c.txt": "content of c in sub",
	}

	for name, data := range files {
		p := filepath.Join(srcDir, name)
		require.NoError(t, os.MkdirAll(filepath.Dir(p), 0755))
		require.NoError(t, os.WriteFile(p, []byte(data), 0644))
	}

	c := newTar()

	// Archive the entire directory.
	reader, err := c.Encode(srcDir, filepath.Dir(srcDir))
	require.NoError(t, err)

	tarData, err := io.ReadAll(reader)
	require.NoError(t, err)
	assert.NotEmpty(t, tarData)

	// Extract.
	extractDir := t.TempDir()
	desc := ocispec.Descriptor{}
	err = c.Decode(extractDir, "", bytes.NewReader(tarData), desc)
	require.NoError(t, err)

	// The archive was created relative to filepath.Dir(srcDir), so the
	// extracted tree includes the base name of srcDir as a prefix.
	base := filepath.Base(srcDir)
	for name, expected := range files {
		got, err := os.ReadFile(filepath.Join(extractDir, base, name))
		require.NoError(t, err, "reading extracted file %s", name)
		assert.Equal(t, expected, string(got))
	}
}

func TestTarExtractRoundtrip(t *testing.T) {
	srcDir := t.TempDir()
	content := []byte("roundtrip data 1234567890")

	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "data.bin"), content, 0644))

	c := newTar()

	// Encode.
	reader, err := c.Encode(filepath.Join(srcDir, "data.bin"), srcDir)
	require.NoError(t, err)

	tarData, err := io.ReadAll(reader)
	require.NoError(t, err)

	// Decode.
	extractDir := t.TempDir()
	desc := ocispec.Descriptor{}
	require.NoError(t, c.Decode(extractDir, "data.bin", bytes.NewReader(tarData), desc))

	got, err := os.ReadFile(filepath.Join(extractDir, "data.bin"))
	require.NoError(t, err)
	assert.Equal(t, content, got)
}

func TestTarInvalidArchive(t *testing.T) {
	c := newTar()
	extractDir := t.TempDir()
	desc := ocispec.Descriptor{}

	// Feed garbage data as a tar stream.
	err := c.Decode(extractDir, "file.txt", strings.NewReader("this is not a tar"), desc)
	assert.Error(t, err)
}
