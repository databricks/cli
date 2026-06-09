package main

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/cmd"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/mod/module"
)

func TestCommandsDontUseUnderscoreInName(t *testing.T) {
	// We use underscore as separator between commands in logs
	// so need to enforce that no command uses it in its name.
	//
	// This test lives in the main package because this is where
	// all commands are imported.
	//
	queue := []*cobra.Command{cmd.New(t.Context())}
	for len(queue) > 0 {
		cmd := queue[0]
		assert.NotContains(t, cmd.Name(), "_")
		queue = append(queue[1:], cmd.Commands()...)
	}
}

func TestImportDoesNotSetCliPathEnv(t *testing.T) {
	// Exporting DATABRICKS_CLI_PATH is done in main, not in a package init,
	// so that importing CLI packages (e.g. from test binaries or generators)
	// does not mutate the process environment.
	//
	// This test lives in the main package because this is where
	// all commands are imported.
	//
	// Test binaries run by their absolute path, which is exactly the condition
	// under which main exports the variable; an import-time export would
	// therefore have set it to this test binary's path by now.
	require.NotEqual(t, filepath.Base(os.Args[0]), os.Args[0])
	assert.NotEqual(t, os.Args[0], os.Getenv("DATABRICKS_CLI_PATH"))
}

func TestFilePath(t *testing.T) {
	// To import this repository as a library, all files must match the
	// file path constraints made by Go. This test ensures that all files
	// in the repository have a valid file path.
	//
	// See https://github.com/databricks/cli/issues/1629
	//
	err := filepath.WalkDir(".", func(path string, _ fs.DirEntry, err error) error {
		switch path {
		case ".":
			return nil
		case ".git":
			return filepath.SkipDir
		}
		if assert.NoError(t, err) {
			assert.NoError(t, module.CheckFilePath(filepath.ToSlash(path)))
		}
		return nil
	})
	assert.NoError(t, err)
}
