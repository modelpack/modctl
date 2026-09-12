package modelfile

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	configmodelfile "github.com/modelpack/modctl/pkg/config/modelfile"
)

func TestIntegration_ExcludePatterns_SinglePattern(t *testing.T) {
	tempDir := t.TempDir()

	// Create workspace with model files and log files.
	files := map[string]string{
		"model.bin":   "model data",
		"config.json": `{"model_type": "test"}`,
		"train.log":   "training log",
		"eval.log":    "eval log",
		"run.py":      "code",
	}
	for name, content := range files {
		require.NoError(t, os.WriteFile(filepath.Join(tempDir, name), []byte(content), 0644))
	}

	config := &configmodelfile.GenerateConfig{
		Workspace:       tempDir,
		Name:            "exclude-test",
		ExcludePatterns: []string{"*.log"},
	}

	mf, err := NewModelfileByWorkspace(tempDir, config)
	require.NoError(t, err)

	// Collect all files in the modelfile.
	allFiles := append(append(append(mf.GetConfigs(), mf.GetModels()...), mf.GetCodes()...), mf.GetDocs()...)

	// .log files should be excluded.
	for _, f := range allFiles {
		assert.NotContains(t, f, ".log", "excluded file %s should not appear", f)
	}

	// model.bin, config.json, run.py should still be present.
	assert.Contains(t, mf.GetModels(), "model.bin")
	assert.Contains(t, mf.GetConfigs(), "config.json")
	assert.Contains(t, mf.GetCodes(), "run.py")
}

func TestIntegration_ExcludePatterns_MultiplePatterns(t *testing.T) {
	tempDir := t.TempDir()

	dirs := []string{"checkpoints", "src"}
	for _, d := range dirs {
		require.NoError(t, os.MkdirAll(filepath.Join(tempDir, d), 0755))
	}

	files := map[string]string{
		"model.bin":                "model",
		"config.json":             `{"model_type": "test"}`,
		"debug.log":               "log",
		"checkpoints/step100.bin": "ckpt",
		"src/train.py":            "code",
	}
	for name, content := range files {
		require.NoError(t, os.WriteFile(filepath.Join(tempDir, name), []byte(content), 0644))
	}

	config := &configmodelfile.GenerateConfig{
		Workspace:       tempDir,
		Name:            "multi-exclude-test",
		ExcludePatterns: []string{"*.log", "checkpoints/*"},
	}

	mf, err := NewModelfileByWorkspace(tempDir, config)
	require.NoError(t, err)

	allFiles := append(append(append(mf.GetConfigs(), mf.GetModels()...), mf.GetCodes()...), mf.GetDocs()...)

	// .log files and checkpoints/* should be excluded.
	for _, f := range allFiles {
		assert.NotContains(t, f, ".log")
		assert.NotContains(t, f, "checkpoints/")
	}

	// Remaining files should be present.
	assert.Contains(t, mf.GetModels(), "model.bin")
	assert.Contains(t, mf.GetCodes(), "src/train.py")
}
