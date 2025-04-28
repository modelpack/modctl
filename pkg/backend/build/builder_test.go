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
//nolint:typecheck
package build

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	modelspec "github.com/CloudNativeAI/model-spec/specs-go/v1"
	godigest "github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	buildconfig "github.com/CloudNativeAI/modctl/pkg/backend/build/config"
	"github.com/CloudNativeAI/modctl/pkg/backend/build/hooks"
	buildmock "github.com/CloudNativeAI/modctl/test/mocks/backend/build"
	storagemock "github.com/CloudNativeAI/modctl/test/mocks/storage"
)

type BuilderTestSuite struct {
	suite.Suite
	mockStorage        *storagemock.Storage
	mockOutputStrategy *buildmock.OutputStrategy
	builder            *abstractBuilder
	tempDir            string
	tempFile           string
}

func (s *BuilderTestSuite) SetupTest() {
	s.mockStorage = new(storagemock.Storage)
	s.mockOutputStrategy = new(buildmock.OutputStrategy)

	s.builder = &abstractBuilder{
		store:    s.mockStorage,
		repo:     "test-repo",
		tag:      "test-tag",
		strategy: s.mockOutputStrategy,
	}

	// Create a temporary directory and file for testing.
	var err error
	s.tempDir, err = os.MkdirTemp("", "builder-test")
	s.Require().NoError(err)

	s.tempFile = filepath.Join(s.tempDir, "test-file.txt")
	err = os.WriteFile(s.tempFile, []byte("test content"), 0666)
	s.Require().NoError(err)
}

func (s *BuilderTestSuite) TearDownTest() {
	os.RemoveAll(s.tempDir)
}

func (s *BuilderTestSuite) TestNewBuilder() {
	testCases := []struct {
		name       string
		outputType OutputType
		expectErr  bool
	}{
		{
			name:       "local output",
			outputType: OutputTypeLocal,
			expectErr:  false,
		},
		{
			name:       "remote output",
			outputType: OutputTypeRemote,
			expectErr:  false,
		},
		{
			name:       "unsupported output type",
			outputType: "invalid",
			expectErr:  true,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			// We're not fully testing the output strategies here, just ensuring the right type is selected.
			builder, err := NewBuilder(tc.outputType, s.mockStorage, "localhost/test-repo", "test-tag")

			if tc.expectErr {
				s.Error(err)
				s.Nil(builder)
			} else {
				s.NoError(err)
				s.NotNil(builder)
			}
		})
	}
}

func (s *BuilderTestSuite) TestBuildLayer() {
	s.Run("successful build layer", func() {
		expectedDesc := ocispec.Descriptor{
			MediaType: "test/media-type.tar",
			Digest:    "sha256:test",
			Size:      100,
		}

		s.mockOutputStrategy.On("OutputLayer", mock.Anything, "test/media-type.tar", "test-file.txt", mock.AnythingOfType("string"), mock.AnythingOfType("int64"), mock.AnythingOfType("*io.PipeReader"), mock.Anything).
			Return(expectedDesc, nil)

		desc, err := s.builder.BuildLayer(context.Background(), "test/media-type.tar", s.tempDir, s.tempFile, hooks.NewHooks())
		s.NoError(err)
		s.Equal(expectedDesc.MediaType, desc.MediaType)
		s.Equal(expectedDesc.Digest, desc.Digest)
		s.Equal(expectedDesc.Size, desc.Size)
	})

	s.Run("file not found", func() {
		_, err := s.builder.BuildLayer(context.Background(), "test/media-type.tar", s.tempDir, filepath.Join(s.tempDir, "non-existent.txt"), hooks.NewHooks())
		s.Error(err)
	})

	s.Run("directory not supported", func() {
		_, err := s.builder.BuildLayer(context.Background(), "test/media-type.tar", s.tempDir, s.tempDir, hooks.NewHooks())
		s.Error(err)
		s.True(strings.Contains(err.Error(), "is a directory and not supported yet"))
	})
}

func (s *BuilderTestSuite) TestBuildConfig() {
	s.Run("successful build config", func() {
		expectedDesc := ocispec.Descriptor{
			MediaType: modelspec.MediaTypeModelConfig,
			Digest:    "sha256:test",
			Size:      100,
		}

		modelConfig := &buildconfig.Model{
			Architecture: "transformer",
			Format:       "safetensors",
			Precision:    "fp16",
			Quantization: "q4_0",
			ParamSize:    "7B",
			Family:       "llama",
			Name:         "llama-2",
		}

		s.mockOutputStrategy.On("OutputConfig", mock.Anything, modelspec.MediaTypeModelConfig, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return(expectedDesc, nil).Once()

		desc, err := s.builder.BuildConfig(context.Background(), []ocispec.Descriptor{}, modelConfig, hooks.NewHooks())
		s.NoError(err)
		s.Equal(expectedDesc, desc)

		s.mockOutputStrategy.AssertExpectations(s.T())
	})

	s.Run("output strategy error", func() {
		modelConfig := &buildconfig.Model{
			Architecture: "transformer",
			Format:       "safetensors",
			Precision:    "fp16",
			Quantization: "q4_0",
			ParamSize:    "7B",
			Family:       "llama",
			Name:         "llama-2",
		}

		s.mockOutputStrategy.On("OutputConfig", mock.Anything, modelspec.MediaTypeModelConfig, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return(ocispec.Descriptor{}, errors.New("output error")).Once()

		_, err := s.builder.BuildConfig(context.Background(), []ocispec.Descriptor{}, modelConfig, hooks.NewHooks())
		s.Error(err)
		s.True(strings.Contains(err.Error(), "output error"))
	})
}

func (s *BuilderTestSuite) TestBuildManifest() {
	s.Run("successful build manifest", func() {
		layers := []ocispec.Descriptor{
			{
				MediaType: "test/layer",
				Digest:    "sha256:layer1",
				Size:      100,
			},
		}
		config := ocispec.Descriptor{
			MediaType: modelspec.MediaTypeModelConfig,
			Digest:    "sha256:config",
			Size:      50,
		}
		annotations := map[string]string{"test": "value"}

		expectedDesc := ocispec.Descriptor{
			MediaType: ocispec.MediaTypeImageManifest,
			Digest:    "sha256:manifest",
			Size:      200,
		}

		s.mockOutputStrategy.On("OutputManifest", mock.Anything, ocispec.MediaTypeImageManifest, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return(expectedDesc, nil).Once()

		desc, err := s.builder.BuildManifest(context.Background(), layers, config, annotations, hooks.NewHooks())
		s.NoError(err)
		s.Equal(expectedDesc, desc)
	})

	s.Run("output strategy error", func() {
		layers := []ocispec.Descriptor{{}}
		config := ocispec.Descriptor{}
		annotations := map[string]string{}

		s.mockOutputStrategy.On("OutputManifest", mock.Anything, ocispec.MediaTypeImageManifest, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return(ocispec.Descriptor{}, errors.New("manifest error")).Once()

		_, err := s.builder.BuildManifest(context.Background(), layers, config, annotations, hooks.NewHooks())
		s.Error(err)
		s.True(strings.Contains(err.Error(), "manifest error"))
	})
}

func (s *BuilderTestSuite) TestBuildModelConfig() {
	modelConfig := &buildconfig.Model{
		Architecture: "transformer",
		Format:       "gguf",
		Precision:    "fp16",
		Quantization: "q4_0",
		ParamSize:    "7B",
		Family:       "llama",
		Name:         "llama-2",
	}

	model, err := buildModelConfig(modelConfig, []ocispec.Descriptor{
		{Digest: godigest.Digest("sha256:layer-1")},
		{Digest: godigest.Digest("sha256:layer-2")},
	})
	s.NoError(err)
	s.NotNil(model)

	s.Equal("transformer", model.Config.Architecture)
	s.Equal("gguf", model.Config.Format)
	s.Equal("fp16", model.Config.Precision)
	s.Equal("q4_0", model.Config.Quantization)
	s.Equal("7B", model.Config.ParamSize)

	s.Equal("llama", model.Descriptor.Family)
	s.Equal("llama-2", model.Descriptor.Name)
	s.NotNil(model.Descriptor.CreatedAt)
	s.WithinDuration(time.Now(), *model.Descriptor.CreatedAt, 5*time.Second)

	s.Equal("layers", model.ModelFS.Type)
	s.Len(model.ModelFS.DiffIDs, 2)
	s.Equal("sha256:layer-1", model.ModelFS.DiffIDs[0].String())
	s.Equal("sha256:layer-2", model.ModelFS.DiffIDs[1].String())
}

func TestBuilderSuite(t *testing.T) {
	suite.Run(t, new(BuilderTestSuite))
}

func TestPipeReader(t *testing.T) {
	r := strings.NewReader("some io.Reader stream to be read\n")
	r1, r2 := splitReader(r)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_, err := io.Copy(os.Stdout, r2)
		assert.NoError(t, err)
	}()
	_, err := io.Copy(os.Stdout, r1)
	assert.NoError(t, err)
	wg.Wait()
}

func createTempFile(t *testing.T, dir, pattern, content string) string {
	t.Helper()
	f, err := os.CreateTemp(dir, pattern)
	assert.NoError(t, err)
	_, err = f.WriteString(content)
	assert.NoError(t, err)
	err = f.Close()
	assert.NoError(t, err)
	return f.Name()
}

func createTempDir(t *testing.T, dir, pattern string) string {
	t.Helper()
	name, err := os.MkdirTemp(dir, pattern)
	assert.NoError(t, err)
	return name
}

func TestGetFileMetadata(t *testing.T) {
	baseTempDir := t.TempDir()

	// --- Test Case 1: Regular File ---
	t.Run("Regular File", func(t *testing.T) {
		content := "hello world"
		filePath := createTempFile(t, baseTempDir, "testfile-*.txt", content)
		fileInfo, err := os.Stat(filePath) // Get ground truth info
		assert.NoError(t, err)

		metadata, err := getFileMetadata(filePath)
		assert.NoError(t, err)

		assert.Equal(t, filepath.Base(filePath), metadata.Name)
		assert.Equal(t, int64(len(content)), metadata.Size)
		assert.Equal(t, uint32(fileInfo.Mode().Perm()), metadata.Mode)
		assert.Equal(t, byte(0), metadata.Typeflag, "Typeflag should be 0 for regular file")
		assert.WithinDuration(t, fileInfo.ModTime(), metadata.ModTime, time.Second)

		// Check UID/GID only on Unix-like systems
		if stat, ok := fileInfo.Sys().(*syscall.Stat_t); ok {
			assert.Equal(t, stat.Uid, metadata.Uid, "UID mismatch")
			assert.Equal(t, stat.Gid, metadata.Gid, "GID mismatch")
		} else if runtime.GOOS != "windows" {
			// If not windows and not syscall.Stat_t, something is unexpected
			t.Logf("Warning: Could not get syscall.Stat_t on non-Windows OS (%s)", runtime.GOOS)
		} else {
			// On Windows, expect 0 as syscall.Stat_t assertion fails
			assert.Equal(t, uint32(0), metadata.Uid, "UID should be 0 on Windows")
			assert.Equal(t, uint32(0), metadata.Gid, "GID should be 0 on Windows")
		}
	})

	// --- Test Case 2: Directory ---
	t.Run("Directory", func(t *testing.T) {
		dirPath := createTempDir(t, baseTempDir, "testdir-*")
		dirInfo, err := os.Stat(dirPath)
		assert.NoError(t, err)

		metadata, err := getFileMetadata(dirPath)
		assert.NoError(t, err)

		assert.Equal(t, filepath.Base(dirPath), metadata.Name)
		assert.Equal(t, dirInfo.Size(), metadata.Size)
		assert.Equal(t, uint32(dirInfo.Mode().Perm()), metadata.Mode)
		assert.Equal(t, byte(5), metadata.Typeflag, "Typeflag should be 5 for directory")
		assert.WithinDuration(t, dirInfo.ModTime(), metadata.ModTime, time.Second)

		// Check UID/GID only on Unix-like systems
		if stat, ok := dirInfo.Sys().(*syscall.Stat_t); ok {
			assert.Equal(t, stat.Uid, metadata.Uid, "UID mismatch")
			assert.Equal(t, stat.Gid, metadata.Gid, "GID mismatch")
		} else if runtime.GOOS != "windows" {
			t.Logf("Warning: Could not get syscall.Stat_t on non-Windows OS (%s)", runtime.GOOS)
		} else {
			assert.Equal(t, uint32(0), metadata.Uid, "UID should be 0 on Windows")
			assert.Equal(t, uint32(0), metadata.Gid, "GID should be 0 on Windows")
		}
	})
}
