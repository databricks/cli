package dlt

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Add your imports here

// Add your tests here

type dltTestEnv struct {
	tempDir        string
	binName        string
	symlinkName    string
	databricksPath string
}

func setupDLTTestEnv(t *testing.T) *dltTestEnv {
	tempDir := t.TempDir()
	binName := "databricks"
	symlinkName := "dlt"
	if runtime.GOOS == "windows" {
		binName += ".bat"
		symlinkName += ".bat"
	}
	databricksPath := filepath.Join(tempDir, binName)
	f, err := os.Create(databricksPath)
	assert.NoError(t, err)
	f.Close()
	err = os.Chmod(databricksPath, 0o755)
	assert.NoError(t, err)

	oldPath := os.Getenv("PATH")
	os.Setenv("PATH", tempDir+string(os.PathListSeparator)+oldPath)
	t.Cleanup(func() { os.Setenv("PATH", oldPath) })

	return &dltTestEnv{
		tempDir:        tempDir,
		binName:        binName,
		symlinkName:    symlinkName,
		databricksPath: databricksPath,
	}
}

func TestInstallDLTSymlink(t *testing.T) {
	env := setupDLTTestEnv(t)

	err := InstallDLTSymlink()
	assert.NoError(t, err)
	dltSymlink := filepath.Join(env.tempDir, env.symlinkName)
	_, err = os.Lstat(dltSymlink)
	assert.NoError(t, err, "symlink was not created")
}

func TestDLTSymlinkRunsInit(t *testing.T) {
	env := setupDLTTestEnv(t)

	var err error
	var script string
	if os.PathSeparator == '\\' {
		script = "@echo off\necho databricks called with: %*\n"
	} else {
		script = "#!/bin/sh\necho databricks called with: $@\n"
	}

	// Overwrite the dummy 'databricks' binary as a shell/batch script that prints args
	err = os.WriteFile(env.databricksPath, []byte(script), 0o755)
	assert.NoError(t, err)

	// Create the symlink
	err = InstallDLTSymlink()
	assert.NoError(t, err)

	// Run the symlinked binary with 'init' and check it executes without error
	dltSymlink := filepath.Join(env.tempDir, env.symlinkName)
	cmd := exec.Command(dltSymlink, "init")
	cmd.Dir = env.tempDir
	output, err := cmd.CombinedOutput()
	assert.NoErrorf(t, err, "failed to run dlt init: output: %s", string(output))

	// Check the output is as expected
	assert.Contains(t, string(output), "databricks", "unexpected output: got %q, want substring containing %q", string(output), "databricks")
	assert.Contains(t, string(output), "init", "unexpected output: got %q, want substring containing %q", string(output), "databricks")
}

func TestInstallDLTSymlink_AlreadyExists(t *testing.T) {
	env := setupDLTTestEnv(t)
	dltPath := filepath.Join(env.tempDir, env.symlinkName)
	err := os.WriteFile(dltPath, []byte("not a symlink"), 0o644)
	assert.NoError(t, err)
	err = InstallDLTSymlink()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists", "expected error about symlink already existing, got: %v", err)
}

func TestInstallDLTSymlink_AlreadyInstalled(t *testing.T) {
	env := setupDLTTestEnv(t)
	realPath, err := filepath.EvalSymlinks(env.databricksPath)
	assert.NoError(t, err)
	dltPath := filepath.Join(env.tempDir, env.symlinkName)
	err = os.Symlink(realPath, dltPath)
	assert.NoError(t, err)

	err = InstallDLTSymlink()
	assert.Error(t, err)
	assert.EqualError(t, err, "dlt already installed")
}
