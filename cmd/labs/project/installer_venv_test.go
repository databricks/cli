package project

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestShouldRecreateVenv(t *testing.T) {
	dir := t.TempDir()
	venvPath := filepath.Join(dir, "venv")
	pythonBin := filepath.Join(venvPath, "bin", "python3")
	if runtime.GOOS == "windows" {
		pythonBin = filepath.Join(venvPath, "Scripts", "python.exe")
	}

	require.False(t, shouldRecreateVenv(venvPath, pythonBin))

	require.NoError(t, os.MkdirAll(filepath.Dir(pythonBin), 0o755))
	require.True(t, shouldRecreateVenv(venvPath, pythonBin))

	require.NoError(t, os.WriteFile(pythonBin, []byte("python"), 0o755))
	require.False(t, shouldRecreateVenv(venvPath, pythonBin))
}

func TestShouldRecreateVenvDanglingInterpreter(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink dangling interpreter check is unix specific")
	}

	dir := t.TempDir()
	venvPath := filepath.Join(dir, "venv")
	binDir := filepath.Join(venvPath, "bin")
	pythonBin := filepath.Join(binDir, "python3")
	require.NoError(t, os.MkdirAll(binDir, 0o755))
	require.NoError(t, os.Symlink(filepath.Join(dir, "missing-python"), pythonBin))
	require.True(t, shouldRecreateVenv(venvPath, pythonBin))
}
