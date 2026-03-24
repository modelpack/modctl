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

package backend

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/modelpack/modctl/pkg/config"
	"github.com/modelpack/modctl/test/mocks/modelfile"
)

func TestGetProcessors(t *testing.T) {
	modelfile := &modelfile.Modelfile{}
	modelfile.On("GetConfigs").Return([]string{"config1", "config2"})
	modelfile.On("GetModels").Return([]string{"model1", "model2"})
	modelfile.On("GetCodes").Return([]string{"1.py", "2.py"})
	modelfile.On("GetDocs").Return([]string{"doc1", "doc2"})

	b := &backend{}
	processors := b.getProcessors(modelfile, &config.Build{})

	assert.Len(t, processors, 4)
	assert.Equal(t, "config", processors[0].Name())
	assert.Equal(t, "model", processors[1].Name())
	assert.Equal(t, "code", processors[2].Name())
	assert.Equal(t, "doc", processors[3].Name())
}

func TestEstimateBuildSize(t *testing.T) {
	t.Run("single files", func(t *testing.T) {
		workDir := t.TempDir()

		// Create test files with known sizes.
		assert.NoError(t, os.WriteFile(filepath.Join(workDir, "model.bin"), make([]byte, 1024), 0644))
		assert.NoError(t, os.WriteFile(filepath.Join(workDir, "config.json"), make([]byte, 256), 0644))

		mf := &modelfile.Modelfile{}
		mf.On("GetConfigs").Return([]string{"config.json"})
		mf.On("GetModels").Return([]string{"model.bin"})
		mf.On("GetCodes").Return([]string{})
		mf.On("GetDocs").Return([]string{})

		size := estimateBuildSize(workDir, mf)
		assert.Equal(t, int64(1280), size)
	})

	t.Run("directory entry", func(t *testing.T) {
		workDir := t.TempDir()

		// Create a subdirectory with files.
		subDir := filepath.Join(workDir, "models")
		assert.NoError(t, os.MkdirAll(subDir, 0755))
		assert.NoError(t, os.WriteFile(filepath.Join(subDir, "a.bin"), make([]byte, 512), 0644))
		assert.NoError(t, os.WriteFile(filepath.Join(subDir, "b.bin"), make([]byte, 512), 0644))

		mf := &modelfile.Modelfile{}
		mf.On("GetConfigs").Return([]string{})
		mf.On("GetModels").Return([]string{"models"})
		mf.On("GetCodes").Return([]string{})
		mf.On("GetDocs").Return([]string{})

		size := estimateBuildSize(workDir, mf)
		assert.Equal(t, int64(1024), size)
	})

	t.Run("nonexistent file is skipped", func(t *testing.T) {
		workDir := t.TempDir()

		assert.NoError(t, os.WriteFile(filepath.Join(workDir, "real.bin"), make([]byte, 100), 0644))

		mf := &modelfile.Modelfile{}
		mf.On("GetConfigs").Return([]string{})
		mf.On("GetModels").Return([]string{"real.bin", "missing.bin"})
		mf.On("GetCodes").Return([]string{})
		mf.On("GetDocs").Return([]string{})

		size := estimateBuildSize(workDir, mf)
		assert.Equal(t, int64(100), size)
	})

	t.Run("empty modelfile", func(t *testing.T) {
		workDir := t.TempDir()

		mf := &modelfile.Modelfile{}
		mf.On("GetConfigs").Return([]string{})
		mf.On("GetModels").Return([]string{})
		mf.On("GetCodes").Return([]string{})
		mf.On("GetDocs").Return([]string{})

		size := estimateBuildSize(workDir, mf)
		assert.Equal(t, int64(0), size)
	})
}
