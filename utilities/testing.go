package utilities

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func CreateTestProject(t *testing.T, root string, projectName string) string {
	// Create temp project dir
	projectDir := filepath.Join(root, "tmp", projectName)
	err := os.MkdirAll(projectDir, 0o755)
	assert.NoError(t, err)

	// Initialize the temp project
	cmd := exec.Command("git", "init")
	cmd.Dir = projectDir
	_, err = cmd.Output()
	assert.NoError(t, err)

	// Initialize the databrick.yml config
	content := []byte(fmt.Sprintf("name: %s\nprofile: DEFAULT", projectName))
	f, err := os.Create(filepath.Join(projectDir, "databricks.yml"))
	assert.NoError(t, err)
	defer f.Close()
	_, err = f.Write(content)
	assert.NoError(t, err)

	return projectDir
}

func DeleteTestProject(t *testing.T, root string, projectName string) {

	projectDir := filepath.Join(root, "tmp", projectName)
	err := os.RemoveAll(projectDir)
	assert.NoError(t, err)
}
