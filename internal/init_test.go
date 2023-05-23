package internal

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/databricks/cli/cmd/bundle"
	"github.com/databricks/cli/cmd/root"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func assertFileContains(t *testing.T, path string, substr string) {
	b, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(b), substr)
}

func TestTemplateInitializationForDevConfig(t *testing.T) {
	// create target directory with the input config
	tmp := t.TempDir()
	f, err := os.Create(filepath.Join(tmp, "config.json"))
	require.NoError(t, err)
	_, err = f.WriteString(`
	{
		"project_name": "development_project",
		"cloud_type": "AWS",
		"ci_type": "github",
		"is_production": false
	}
	`)
	f.Close()
	require.NoError(t, err)

	// materialize the template
	cmd := root.RootCmd
	cmd.SetArgs([]string{"bundle", "init", filepath.FromSlash("testdata/init/templateDefinition"), "--target-dir", tmp, "--config-file", filepath.Join(tmp, "config.json")})
	err = cmd.Execute()
	require.NoError(t, err)

	// assert on materialized template
	assert.FileExists(t, filepath.Join(tmp, "development_project", "aws_file"))
	assert.FileExists(t, filepath.Join(tmp, "development_project", ".github"))
	assert.NoFileExists(t, filepath.Join(tmp, "development_project", "azure_file"))
	assertFileContains(t, filepath.Join(tmp, "development_project", "aws_file"), "This file should only be generated for AWS")
	assertFileContains(t, filepath.Join(tmp, "development_project", ".github"), "This is a development project")
}

func TestTemplateInitializationForProdConfig(t *testing.T) {
	// create target directory with the input config
	tmp := t.TempDir()

	// create target directory to with the input config
	configDir := filepath.Join(tmp, "dir-with-config")
	err := os.Mkdir(configDir, os.ModePerm)
	require.NoError(t, err)
	f, err := os.Create(filepath.Join(configDir, "my_config.json"))
	require.NoError(t, err)
	_, err = f.WriteString(`
	{
		"project_name": "production_project",
		"cloud_type": "Azure",
		"ci_type": "azure_devops",
		"is_production": true
	}
	`)
	f.Close()
	require.NoError(t, err)

	// create directory to initialize the template instance within
	instanceDir := filepath.Join(tmp, "dir-with-instance")
	err = os.Mkdir(instanceDir, os.ModePerm)
	require.NoError(t, err)

	// materialize the template
	cmd := root.RootCmd
	childCommands := cmd.Commands()
	fmt.Println(childCommands)
	cmd.SetArgs([]string{"bundle", "init", filepath.FromSlash("testdata/init/templateDefinition"), "--target-dir", instanceDir, "--config-file", filepath.Join(configDir, "my_config.json")})
	err = cmd.Execute()
	require.NoError(t, err)

	// assert on materialized template
	assert.FileExists(t, filepath.Join(instanceDir, "production_project", "azure_file"))
	assert.FileExists(t, filepath.Join(instanceDir, "production_project", ".azure_devops"))
	assert.NoFileExists(t, filepath.Join(instanceDir, "production_project", "aws_file"))
	assertFileContains(t, filepath.Join(instanceDir, "production_project", "azure_file"), "This file should only be generated for Azure")
	assertFileContains(t, filepath.Join(instanceDir, "production_project", ".azure_devops"), "This is a production project")
}
