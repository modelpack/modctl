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

package modelfile

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	configmodelfile "github.com/modelpack/modctl/pkg/config/modelfile"
)

// resetGenerateConfig resets the package-level global to a fresh instance to avoid
// cross-test state pollution.
func resetGenerateConfig() {
	generateConfig = configmodelfile.NewGenerateConfig()
}

// TestIntegration_CLI_Generate_BasicFlags tests that the generate command writes a
// Modelfile to the specified output directory containing expected directives.
func TestIntegration_CLI_Generate_BasicFlags(t *testing.T) {
	// Create temp workspace with model and config files.
	workspaceDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(workspaceDir, "model.bin"), []byte("model data"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(workspaceDir, "config.json"), []byte(`{"model_type":"llama"}`), 0644))

	outputDir := t.TempDir()

	resetGenerateConfig()
	generateConfig.Name = "test-model"
	generateConfig.Arch = "transformer"
	generateConfig.Output = filepath.Join(outputDir, configmodelfile.DefaultModelfileName)
	generateConfig.Workspace = workspaceDir

	err := runGenerate(context.Background())
	require.NoError(t, err)

	modelfilePath := filepath.Join(outputDir, configmodelfile.DefaultModelfileName)
	data, err := os.ReadFile(modelfilePath)
	require.NoError(t, err)

	content := string(data)
	assert.True(t, strings.Contains(content, "NAME"), "expected NAME directive in Modelfile")
	assert.True(t, strings.Contains(content, "ARCH"), "expected ARCH directive in Modelfile")
	assert.True(t, strings.Contains(content, "MODEL"), "expected MODEL directive in Modelfile")
	assert.True(t, strings.Contains(content, "CONFIG"), "expected CONFIG directive in Modelfile")
	assert.True(t, strings.Contains(content, "test-model"), "expected model name in Modelfile")
	assert.True(t, strings.Contains(content, "transformer"), "expected arch in Modelfile")
}

// TestIntegration_CLI_Generate_OutputAndOverwrite tests that generate fails when a
// Modelfile already exists (without --overwrite) and succeeds when --overwrite is set.
func TestIntegration_CLI_Generate_OutputAndOverwrite(t *testing.T) {
	// Create temp workspace with a model file only (no config.json to keep it simple).
	workspaceDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(workspaceDir, "model.bin"), []byte("model data"), 0644))

	outputDir := t.TempDir()
	modelfilePath := filepath.Join(outputDir, configmodelfile.DefaultModelfileName)

	// Pre-create the Modelfile so it already exists.
	require.NoError(t, os.WriteFile(modelfilePath, []byte("# existing"), 0644))

	t.Run("without overwrite flag errors", func(t *testing.T) {
		resetGenerateConfig()
		generateConfig.Name = "test-model"
		generateConfig.Output = modelfilePath
		generateConfig.Workspace = workspaceDir
		generateConfig.Overwrite = false

		err := generateConfig.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "already exists")
	})

	t.Run("with overwrite flag succeeds", func(t *testing.T) {
		resetGenerateConfig()
		generateConfig.Name = "test-model"
		generateConfig.Output = modelfilePath
		generateConfig.Workspace = workspaceDir
		generateConfig.Overwrite = true

		err := generateConfig.Validate()
		require.NoError(t, err)

		err = runGenerate(context.Background())
		require.NoError(t, err)

		data, err := os.ReadFile(modelfilePath)
		require.NoError(t, err)
		assert.NotEqual(t, "# existing", string(data), "Modelfile should have been overwritten")
	})
}

// TestIntegration_CLI_Generate_MutualExclusion tests that providing both a path
// argument and --model-url is rejected as mutually exclusive.
func TestIntegration_CLI_Generate_MutualExclusion(t *testing.T) {
	resetGenerateConfig()

	// Both a positional path arg and --model-url being set is mutually exclusive.
	// Invoke the cobra RunE directly to exercise the validation in generateCmd.
	generateConfig.ModelURL = "https://huggingface.co/some/model"

	err := generateCmd.RunE(generateCmd, []string{"/some/path"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "mutually exclusive")
}
