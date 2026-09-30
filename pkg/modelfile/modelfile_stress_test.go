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

//go:build stress

package modelfile

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	configmodelfile "github.com/modelpack/modctl/pkg/config/modelfile"
)

// TestStress_NearMaxFileCount creates a workspace with 2040 files (near
// MaxWorkspaceFileCount=2048) and asserts that NewModelfileByWorkspace
// handles it without error.
func TestStress_NearMaxFileCount(t *testing.T) {
	const fileCount = 2040

	tempDir := t.TempDir()

	// Create one model file to satisfy the "no model/code/dataset" requirement.
	require.NoError(t, os.WriteFile(filepath.Join(tempDir, "model.bin"), []byte("model data"), 0644))

	// Fill the rest with .py files (code type).
	for i := 1; i < fileCount; i++ {
		name := fmt.Sprintf("script_%04d.py", i)
		content := fmt.Sprintf("# script %d\nprint('hello')\n", i)
		require.NoError(t, os.WriteFile(filepath.Join(tempDir, name), []byte(content), 0644))
	}

	config := &configmodelfile.GenerateConfig{
		Workspace: tempDir,
		Name:      "stress-near-max",
	}

	_, err := NewModelfileByWorkspace(tempDir, config)
	require.NoError(t, err, "workspace with %d files (near limit %d) should be accepted", fileCount, MaxWorkspaceFileCount)
}

// TestStress_DeeplyNestedDirs creates a 100-level deep directory structure
// with a model.bin at the deepest level and asserts that
// NewModelfileByWorkspace traverses it without error or stack overflow, and
// that the deep model file is discoverable via GetModels().
func TestStress_DeeplyNestedDirs(t *testing.T) {
	const depth = 100

	tempDir := t.TempDir()

	// Build the nested path: tempDir/d0/d1/.../d99
	parts := make([]string, depth+1)
	parts[0] = tempDir
	for i := 0; i < depth; i++ {
		parts[i+1] = fmt.Sprintf("d%d", i)
	}
	deepDir := filepath.Join(parts...)
	require.NoError(t, os.MkdirAll(deepDir, 0755))

	// Place the model at the deepest level.
	deepModel := filepath.Join(deepDir, "model.bin")
	require.NoError(t, os.WriteFile(deepModel, []byte("deep model data"), 0644))

	config := &configmodelfile.GenerateConfig{
		Workspace: tempDir,
		Name:      "stress-deep-nesting",
	}

	mf, err := NewModelfileByWorkspace(tempDir, config)
	require.NoError(t, err, "deeply nested workspace should not cause error or stack overflow")

	// Build the expected relative path for the deep model.
	relParts := make([]string, depth+1)
	for i := 0; i < depth; i++ {
		relParts[i] = fmt.Sprintf("d%d", i)
	}
	relParts[depth] = "model.bin"
	expectedRelPath := strings.Join(relParts, string(filepath.Separator))

	models := mf.GetModels()
	require.Contains(t, models, expectedRelPath, "deep model file should appear in GetModels()")
}
