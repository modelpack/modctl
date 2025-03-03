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

package processor

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/CloudNativeAI/modctl/test/mocks/storage"
	modelspec "github.com/CloudNativeAI/model-spec/specs-go/v1"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type modelConfigProcessorSuite struct {
	suite.Suite
	mockStore *storage.Storage
	processor Processor
	workDir   string
}

func (s *modelConfigProcessorSuite) SetupTest() {
	s.mockStore = &storage.Storage{}
	s.processor = NewModelConfigProcessor(s.mockStore, modelspec.MediaTypeModelWeightConfig, []string{"config"})
	// generate test files for prorcess.
	s.workDir = s.Suite.T().TempDir()
	if err := os.WriteFile(filepath.Join(s.workDir, "config"), []byte(""), 0644); err != nil {
		s.Suite.T().Fatal(err)
	}
}

func (s *modelConfigProcessorSuite) TestName() {
	assert.Equal(s.Suite.T(), "config", s.processor.Name())
}

func (s *modelConfigProcessorSuite) TestProcess() {
	ctx := context.Background()
	repo := "test-repo"
	s.mockStore.On("PushBlob", ctx, repo, mock.Anything).Return("sha256:1234567890abcdef", int64(1024), nil)

	desc, err := s.processor.Process(ctx, s.workDir, repo)
	assert.NoError(s.Suite.T(), err)
	assert.NotNil(s.Suite.T(), desc)
	assert.Equal(s.Suite.T(), "sha256:1234567890abcdef", desc[0].Digest.String())
	assert.Equal(s.Suite.T(), int64(1024), desc[0].Size)
	assert.Equal(s.Suite.T(), "config", desc[0].Annotations[modelspec.AnnotationFilepath])
}

func TestModelConfigProcessorSuite(t *testing.T) {
	suite.Run(t, new(modelConfigProcessorSuite))
}
