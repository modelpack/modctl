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

package interceptor

import (
	"archive/tar"
	"context"
	"encoding/json"
	"fmt"
	"hash/crc32"
	"io"
	"strings"

	"github.com/CloudNativeAI/modctl/pkg/codec"
	modelspec "github.com/CloudNativeAI/model-spec/specs-go/v1"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

const (
	CrcsKey              = "org.cnai.nydus.crcs"
	DefaultFileChunkSize = 4 * 1024 * 1024
)

var mediaTypeChunkSizeMap = map[string]int{
	modelspec.MediaTypeModelWeight:  64 * 1024 * 1024,
	modelspec.MediaTypeModelDataset: 64 * 1024 * 1024,
}

var table = crc32.MakeTable(crc32.Castagnoli)

type nydus struct{}

type FileCrcList struct {
	Files []FileCrcInfo `json:"files"`
}

type FileCrcInfo struct {
	FilePath  string `json:"file_path"`
	ChunkCrcs string `json:"chunk_crcs"`
}

func NewNydus() *nydus {
	return &nydus{}
}

func (n *nydus) Intercept(ctx context.Context, mediaType string, filepath string, readerType string, reader io.Reader) (ApplyDescriptorFn, error) {
	crcsStr := ""
	chunkSize := int64(DefaultFileChunkSize)
	if c, ok := mediaTypeChunkSizeMap[mediaType]; ok {
		chunkSize = int64(c)
	}

	switch readerType {
	case codec.Tar:
		fileCrcs, err := calcCrc32inTar(ctx, reader, chunkSize)
		if err != nil {
			return nil, fmt.Errorf("failed to calculate crc32 in tar: %w", err)
		}
		b, err := json.Marshal(fileCrcs)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal crcs: %w", err)
		}
		return func(desc *ocispec.Descriptor) {
			if desc.Annotations == nil {
				desc.Annotations = make(map[string]string)
			}
			desc.Annotations[CrcsKey] = string(b)
		}, nil
	case codec.Raw:
		crc32Results, err := calcCrc32(ctx, reader, chunkSize)
		if err != nil {
			return nil, fmt.Errorf("failed to calculate crc32: %w", err)
		}
		crcsStr = buildCrc32Str(crc32Results)
		crcInfo := FileCrcInfo{
			FilePath:  filepath,
			ChunkCrcs: crcsStr,
		}
		crcs := FileCrcList{
			Files: []FileCrcInfo{crcInfo},
		}
		b, err := json.Marshal(crcs)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal crcs: %w", err)
		}
		return func(desc *ocispec.Descriptor) {
			if desc.Annotations == nil {
				desc.Annotations = make(map[string]string)
			}
			desc.Annotations[CrcsKey] = string(b)
		}, nil
	default:
		return nil, fmt.Errorf("unsupported reader type: %s", readerType)
	}
}

func calcCrc32inTar(ctx context.Context, r io.Reader, chunkSize int64) (*FileCrcList, error) {
	fileCrcList := FileCrcList{
		Files: make([]FileCrcInfo, 0),
	}
	tarReader := tar.NewReader(r)
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			header, err := tarReader.Next()
			if err == io.EOF {
				return &fileCrcList, nil
			}
			if err != nil {
				return nil, fmt.Errorf("error reading tar: %w", err)
			}
			if header.Typeflag == tar.TypeReg {
				crc32Results, err := calcCrc32(ctx, tarReader, chunkSize)
				if err != nil {
					return nil, fmt.Errorf("failed to calculate crc32: %w", err)
				}
				crcsStr := buildCrc32Str(crc32Results)
				crcInfo := FileCrcInfo{
					FilePath:  header.Name,
					ChunkCrcs: crcsStr,
				}
				fileCrcList.Files = append(fileCrcList.Files, crcInfo)
			}
		}
	}
}

func calcCrc32(ctx context.Context, r io.Reader, chunkSize int64) ([]uint32, error) {
	var crc32Results []uint32
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			limitedReader := io.LimitReader(r, chunkSize)
			hash := crc32.New(table)
			n, err := io.Copy(hash, limitedReader)
			if n == 0 || err == io.EOF {
				// if no data read, return 0 as the crc32 value.
				if len(crc32Results) == 0 {
					return []uint32{0}, nil
				}
				return crc32Results, nil
			}

			if err != nil {
				return nil, fmt.Errorf("failed to read data: %w", err)
			}

			if n > 0 {
				crc32Results = append(crc32Results, hash.Sum32())
			}
		}
	}
}

func buildCrc32Str(crc32Results []uint32) string {
	hexCrcs := make([]string, len(crc32Results))
	for i, crc := range crc32Results {
		hexCrcs[i] = fmt.Sprintf("0x%x", crc)
	}
	return strings.Join(hexCrcs, ",")
}
