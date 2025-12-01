package resources

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCopyResourceFile(t *testing.T) {
	// Create temp directories for source and destination
	tmpDir := t.TempDir()
	srcDir := filepath.Join(tmpDir, "src", "resources")
	dstDir := filepath.Join(tmpDir, "dst")
	resourcesDir := filepath.Join(dstDir, "resources")

	require.NoError(t, os.MkdirAll(srcDir, 0o755))
	require.NoError(t, os.MkdirAll(resourcesDir, 0o755))

	// Create a test resource file
	testContent := "name: test_job\npackage: test_package"
	testFile := filepath.Join(srcDir, "template.job.yml")
	require.NoError(t, os.WriteFile(testFile, []byte(testContent), 0o644))

	// Test copying with replacements
	replacements := map[string]string{
		"test_job":     "my_job",
		"test_package": "my_package",
	}
	err := CopyResourceFile(srcDir, dstDir, "my_job", ".job.yml", replacements)
	assert.NoError(t, err)

	// Verify the file was copied and modified
	dstFile := filepath.Join(resourcesDir, "my_job.job.yml")
	content, err := os.ReadFile(dstFile)
	require.NoError(t, err)
	assert.Contains(t, string(content), "my_job")
	assert.Contains(t, string(content), "my_package")
	assert.NotContains(t, string(content), "test_job")
	assert.NotContains(t, string(content), "test_package")
}

func TestCopyResourceFile_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	srcDir := filepath.Join(tmpDir, "src", "resources")
	require.NoError(t, os.MkdirAll(srcDir, 0o755))

	// Test with non-existent file type
	err := CopyResourceFile(srcDir, tmpDir, "test", ".nonexistent.yml", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no .nonexistent.yml file found")
}
