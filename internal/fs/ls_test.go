package fs

import (
	"encoding/json"
	"fmt"
	"path"
	"testing"

	_ "github.com/databricks/cli/cmd/fs"
	"github.com/databricks/cli/internal"
	"github.com/databricks/cli/internal/helpers"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func assertObjectListed(t *testing.T, parsedLogs []map[string]string, name string, objectType string) {
	foundFile := false
	for _, v := range parsedLogs {
		if v["Name"] != name {
			continue
		}
		foundFile = true
		assert.Equal(t, objectType, v["Type"])
	}
	assert.True(t, foundFile, fmt.Sprintf("failed to find file %s in output logs", name))
}

func TestAccFsLs(t *testing.T) {
	t.Log(internal.GetEnvOrSkipTest(t, "CLOUD_ENV"))

	// setup some testdata in the workspace
	w := helpers.NewWorkspaceTestdata(t)
	w.AddFile("foo.txt", `hello, world`)
	w.AddFile("python_notebook.py", cmdio.Heredoc(`
	#Databricks notebook source
	print(2)`))
	w.AddFile("python_file.py", `print(1)`)
	w.Mkdir("my_directory")
	w.AddFile("my_directory/.gitkeep", "")

	// run list command
	stdout, stderr := internal.RequireSuccessfulRun(t, "fs", "ls", w.RootPath(), "--output=json")

	// read and parse the output logs
	parsedLogs := make([]map[string]string, 0)
	err := json.Unmarshal(stdout.Bytes(), &parsedLogs)
	require.NoError(t, err)

	// make assertions on the output logs
	assert.Equal(t, stderr.String(), "")
	assertObjectListed(t, parsedLogs, "python_file.py", "FILE")
	assertObjectListed(t, parsedLogs, "foo.txt", "FILE")
	assertObjectListed(t, parsedLogs, "python_notebook", "NOTEBOOK")
	assertObjectListed(t, parsedLogs, "my_directory", "DIRECTORY")
}

func TestAccFsLsWithAbsoluteFlag(t *testing.T) {
	t.Log(internal.GetEnvOrSkipTest(t, "CLOUD_ENV"))

	// setup some testdata in the workspace
	w := helpers.NewWorkspaceTestdata(t)
	w.AddFile("foo.txt", `hello, world`)
	w.AddFile("python_notebook.py", cmdio.Heredoc(`
	#Databricks notebook source
	print(2)`))
	w.AddFile("python_file.py", `print(1)`)
	w.Mkdir("my_directory")
	w.AddFile("my_directory/.gitkeep", "")

	// run list command
	stdout, stderr := internal.RequireSuccessfulRun(t, "fs", "ls", w.RootPath(), "--output=json", "--absolute")

	// read and parse the output logs
	parsedLogs := make([]map[string]string, 0)
	err := json.Unmarshal(stdout.Bytes(), &parsedLogs)
	require.NoError(t, err)

	// make assertions on the output logs
	assert.Equal(t, stderr.String(), "")
	assertObjectListed(t, parsedLogs, path.Join(w.RootPath(), "python_file.py"), "FILE")
	assertObjectListed(t, parsedLogs, path.Join(w.RootPath(), "foo.txt"), "FILE")
	assertObjectListed(t, parsedLogs, path.Join(w.RootPath(), "python_notebook"), "NOTEBOOK")
	assertObjectListed(t, parsedLogs, path.Join(w.RootPath(), "my_directory"), "DIRECTORY")
}
