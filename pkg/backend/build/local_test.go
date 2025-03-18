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
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/CloudNativeAI/modctl/pkg/backend/build/hooks"
	storagemock "github.com/CloudNativeAI/modctl/test/mocks/storage"

	modelspec "github.com/CloudNativeAI/model-spec/specs-go/v1"
	godigest "github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type LocalOutputTestSuite struct {
	suite.Suite
	mockStorage *storagemock.Storage
	localOutput *localOutput
	ctx         context.Context
}

func (s *LocalOutputTestSuite) SetupTest() {
	s.mockStorage = new(storagemock.Storage)
	s.localOutput = &localOutput{
		store: s.mockStorage,
		repo:  "test-repo",
		tag:   "test-tag",
	}
	s.ctx = context.Background()
}

func (s *LocalOutputTestSuite) TestNewLocalOutput() {
	output, err := NewLocalOutput(nil, s.mockStorage, "repo", "tag")

	s.NoError(err)
	s.NotNil(output)

	lo, ok := output.(*localOutput)
	s.True(ok)
	s.Equal(s.mockStorage, lo.store)
	s.Equal("repo", lo.repo)
	s.Equal("tag", lo.tag)
}

func (s *LocalOutputTestSuite) TestOutputLayer() {
	s.Run("successful output layer", func() {
		expectedDigest := "sha256:1234567890"
		expectedSize := int64(1024)
		reader := strings.NewReader("test content")

		s.mockStorage.On("PushBlob", s.ctx, "test-repo", mock.Anything, ocispec.Descriptor{}).
			Return(expectedDigest, expectedSize, nil).Once()

		desc, err := s.localOutput.OutputLayer(s.ctx, "test/mediatype", "test-file.txt", expectedDigest, expectedSize, reader, hooks.NewHooks())

		s.NoError(err)
		s.Equal("test/mediatype", desc.MediaType)
		s.Equal(godigest.Digest(expectedDigest), desc.Digest)
		s.Equal(expectedSize, desc.Size)
		s.Equal("test-file.txt", desc.Annotations[modelspec.AnnotationFilepath])
		s.mockStorage.AssertExpectations(s.T())
	})

	s.Run("storage error", func() {
		reader := strings.NewReader("test content")

		s.mockStorage.On("PushBlob", s.ctx, "test-repo", mock.Anything, ocispec.Descriptor{}).
			Return("", int64(0), errors.New("storage error")).Once()

		_, err := s.localOutput.OutputLayer(s.ctx, "test/mediatype", "/work", "test-file.txt", int64(0), reader, hooks.NewHooks())

		s.Error(err)
		s.Contains(err.Error(), "failed to push blob to storage")
		s.mockStorage.AssertExpectations(s.T())
	})
}

func (s *LocalOutputTestSuite) TestOutputConfig() {
	s.Run("successful output config", func() {
		configJSON := []byte(`{"config": "test"}`)
		expectedDigest := "sha256:config1234"
		expectedSize := int64(len(configJSON))

		s.mockStorage.On("PushBlob", s.ctx, "test-repo", mock.Anything, ocispec.Descriptor{}).
			Return(expectedDigest, expectedSize, nil).Once()

		desc, err := s.localOutput.OutputConfig(s.ctx, "test/configtype", expectedDigest, expectedSize, bytes.NewReader(configJSON), hooks.NewHooks())

		s.NoError(err)
		s.Equal("test/configtype", desc.MediaType)
		s.Equal(godigest.Digest(expectedDigest), desc.Digest)
		s.Equal(expectedSize, desc.Size)
		s.mockStorage.AssertExpectations(s.T())
	})

	s.Run("storage error", func() {
		configJSON := []byte(`{"config": "test"}`)

		s.mockStorage.On("PushBlob", s.ctx, "test-repo", mock.Anything, ocispec.Descriptor{}).
			Return("", int64(0), errors.New("config error")).Once()

		_, err := s.localOutput.OutputConfig(s.ctx, "test/configtype", "", int64(0), bytes.NewReader(configJSON), hooks.NewHooks())

		s.Error(err)
		s.Contains(err.Error(), "failed to push config to storage")
		s.mockStorage.AssertExpectations(s.T())
	})
}

func (s *LocalOutputTestSuite) TestOutputManifest() {
	s.Run("successful output manifest", func() {
		manifestJSON := []byte(`{"manifest": "test"}`)
		expectedDigest := "sha256:manifest5678"

		s.mockStorage.On("PushManifest", s.ctx, "test-repo", "test-tag", manifestJSON).
			Return(expectedDigest, nil).Once()

		desc, err := s.localOutput.OutputManifest(s.ctx, "test/manifesttype", expectedDigest, int64(len(manifestJSON)), bytes.NewReader(manifestJSON), hooks.NewHooks())

		s.NoError(err)
		s.Equal("test/manifesttype", desc.MediaType)
		s.Equal(godigest.Digest(expectedDigest), desc.Digest)
		s.Equal(int64(len(manifestJSON)), desc.Size)
		s.mockStorage.AssertExpectations(s.T())
	})

	s.Run("storage error", func() {
		manifestJSON := []byte(`{"manifest": "test"}`)

		s.mockStorage.On("PushManifest", s.ctx, "test-repo", "test-tag", manifestJSON).
			Return("", errors.New("manifest error")).Once()

		_, err := s.localOutput.OutputManifest(s.ctx, "test/manifesttype", "", int64(0), bytes.NewReader(manifestJSON), hooks.NewHooks())

		s.Error(err)
		s.Contains(err.Error(), "failed to push manifest to storage")
		s.mockStorage.AssertExpectations(s.T())
	})
}

func TestLocalOutputSuite(t *testing.T) {
	suite.Run(t, new(LocalOutputTestSuite))
}
