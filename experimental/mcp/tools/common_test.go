package tools

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateDatabricksProject(t *testing.T) {
	// Test with empty path
	err := ValidateDatabricksProject("")
	assert.Error(t, err)

	// Test with directory without databricks.yml
	tmpDir := t.TempDir()
	err = ValidateDatabricksProject(tmpDir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "databricks.yml not found")

	// Test with valid Databricks project
	databricksYml := filepath.Join(tmpDir, "databricks.yml")
	require.NoError(t, os.WriteFile(databricksYml, []byte("# test"), 0o644))
	err = ValidateDatabricksProject(tmpDir)
	assert.NoError(t, err)
}
