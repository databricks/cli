package internal

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAccBundleInitErrorOnUnknownFields(t *testing.T) {
	t.Log(GetEnvOrSkipTest(t, "CLOUD_ENV"))

	tmpDir := t.TempDir()
	_, _, err := RequireErrorRun(t, "bundle", "init", "./testdata/init/field-does-not-exist", "--output-dir", tmpDir)
	assert.EqualError(t, err, "failed to compute file content for bar.tmpl. variable \"does_not_exist\" not defined")
}

func TestAccBundleInitOnMlopsStacks(t *testing.T) {
	t.Log(GetEnvOrSkipTest(t, "CLOUD_ENV"))
	tmpDir1 := t.TempDir()
	tmpDir2 := t.TempDir()

	// Create a config file with the project name and root dir
	config := map[string]string{
		"input_project_name": "foobar",
		"input_root_dir":     "foobar",
	}
	b, err := json.Marshal(config)
	require.NoError(t, err)
	os.WriteFile(filepath.Join(tmpDir1, "config.json"), b, 0644)

	// Run bundle init
	assert.NoFileExists(t, filepath.Join(tmpDir2, "foobar", "README.md"))
	RequireSuccessfulRun(t, "bundle", "init", "mlops-stacks", "--output-dir", tmpDir2, "--config-file", filepath.Join(tmpDir1, "config.json"))

	// Assert that the README.md file was created
	assert.FileExists(t, filepath.Join(tmpDir2, "foobar", "README.md"))
}
