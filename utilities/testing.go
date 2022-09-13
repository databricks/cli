package utilities

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func GetTestProject(t *testing.T, root string) string {
	// Create temp project dir
	projectDir := t.TempDir()

	// Initialize the temp project
	cmd := exec.Command("git", "init")
	cmd.Dir = projectDir
	_, err := cmd.Output()
	assert.NoError(t, err)

	// Initialize the databrick.yml config
	content := []byte("name: test-project\nprofile: DEFAULT")
	f, err := os.Create(filepath.Join(projectDir, "databricks.yml"))
	assert.NoError(t, err)
	defer f.Close()
	_, err = f.Write(content)
	assert.NoError(t, err)

	return projectDir
}
