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
	// Test binaries run by their absolute path, which is exactly the condition
	// under which main exports DATABRICKS_CLI_PATH; an import-time export would
	// therefore have set it to this test binary's path by now.
	require.NotEqual(t, filepath.Base(os.Args[0]), os.Args[0])
	assert.NotEqual(t, os.Args[0], os.Getenv("DATABRICKS_CLI_PATH"))
}

func TestCommandArgsForDockerCredentialHelper(t *testing.T) {
	tests := []struct {
		name       string
		executable string
		args       []string
		want       []string
		wantHelper bool
		wantError  string
	}{
		{name: "databricks", executable: "databricks", args: []string{"auth", "token"}, want: []string{"auth", "token"}},
		{name: "helper", executable: "/usr/local/bin/docker-credential-databricks", args: []string{"get"}, want: []string{"auth", "token", "--format=docker"}, wantHelper: true},
		{name: "Windows helper", executable: `C:\Program Files\Databricks\docker-credential-databricks.exe`, args: []string{"get"}, want: []string{"auth", "token", "--format=docker"}, wantHelper: true},
		{name: "unsupported operation", executable: "docker-credential-databricks", args: []string{"store"}, wantHelper: true, wantError: "only supports get"},
		{name: "uppercase operation", executable: "docker-credential-databricks.exe", args: []string{"GET"}, wantHelper: true, wantError: "only supports get"},
		{name: "missing operation", executable: "docker-credential-databricks", wantHelper: true, wantError: "only supports get"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, helper, err := commandArgs(tt.executable, tt.args)
			if tt.wantError != "" {
				require.ErrorContains(t, err, tt.wantError)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
			require.Equal(t, tt.wantHelper, helper)
		})
	}
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
