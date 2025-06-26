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
	"fmt"
	"io"
	"strings"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

type Type = string

const (
	// Raw is the raw codec type.
	Raw Type = "raw"

	// Tar is the tar codec type.
	Tar Type = "tar"
)

// Codec is an interface for encoding and decoding the data.
type Codec interface {
	// Type returns the type of the codec.
	Type() Type

	// Encode encodes the target file into a reader.
	Encode(targetFilePath, workDirPath string) (io.Reader, error)

	// Decode reads the input reader and decodes the data into the output path.
	Decode(outputDir, filePath string, reader io.Reader, desc ocispec.Descriptor) error
}

func New(codecType Type) (Codec, error) {
	switch codecType {
	case Raw:
		return newRaw(), nil
	case Tar:
		return newTar(), nil
	default:
		return nil, fmt.Errorf("unsupported codec type: %s", codecType)
	}
}

// TypeFromMediaType returns the codec type from the media type,
// return empty string if not supported.
func TypeFromMediaType(mediaType string) Type {
	// If the mediaType ends with ".tar", return Tar.
	if strings.HasSuffix(mediaType, ".tar") {
		return Tar
	}

	// If the mediaType ends with ".raw", return Raw.
	if strings.HasSuffix(mediaType, ".raw") {
		return Raw
	}

	return ""
}

// IsRawMediaType returns true if the media type is raw.
func IsRawMediaType(mediaType string) bool {
	return strings.HasSuffix(mediaType, ".raw")
}
