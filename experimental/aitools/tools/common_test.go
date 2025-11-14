package tools

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateDatabricksProject(t *testing.T) {
	err := ValidateDatabricksProject("")
	assert.Error(t, err)

	tmpDir := t.TempDir()
	err = ValidateDatabricksProject(tmpDir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "databricks.yml or app.yml not found")

	// Test with databricks.yml
	databricksYml := filepath.Join(tmpDir, "databricks.yml")
	require.NoError(t, os.WriteFile(databricksYml, []byte("# test"), 0o644))
	err = ValidateDatabricksProject(tmpDir)
	assert.NoError(t, err)

	// Test with app.yml (standalone app project)
	tmpDir2 := t.TempDir()
	appYml := filepath.Join(tmpDir2, "app.yml")
	require.NoError(t, os.WriteFile(appYml, []byte("# test"), 0o644))
	err = ValidateDatabricksProject(tmpDir2)
	assert.NoError(t, err)

	// Test with *.app.yml (named app project)
	tmpDir3 := t.TempDir()
	namedAppYml := filepath.Join(tmpDir3, "my_app.app.yml")
	require.NoError(t, os.WriteFile(namedAppYml, []byte("# test"), 0o644))
	err = ValidateDatabricksProject(tmpDir3)
	assert.NoError(t, err)
}
