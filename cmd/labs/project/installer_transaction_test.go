package project

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVirtualEnvScriptsDir(t *testing.T) {
	for _, tt := range []struct {
		goos string
		want string
	}{
		{goos: "linux", want: filepath.Join("venv", "bin")},
		{goos: "darwin", want: filepath.Join("venv", "bin")},
		{goos: "windows", want: filepath.Join("venv", "Scripts")},
	} {
		t.Run(tt.goos, func(t *testing.T) {
			assert.Equal(t, tt.want, virtualEnvScriptsDir("venv", tt.goos))
		})
	}
}

func TestLibraryTransactionRestoresPreviousLibrary(t *testing.T) {
	root := t.TempDir()
	lib := filepath.Join(root, "lib")
	staging := filepath.Join(root, "staging")
	require.NoError(t, os.Mkdir(lib, 0o755))
	require.NoError(t, os.Mkdir(staging, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(lib, "sentinel"), []byte("previous"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(staging, "sentinel"), []byte("new"), 0o600))

	txn := newLibraryTransaction(lib, staging)
	require.NoError(t, txn.swap())
	installationErr := errors.New("installation failed")
	err := errors.Join(installationErr, txn.rollback())
	assert.ErrorIs(t, err, installationErr)
	contents, readErr := os.ReadFile(filepath.Join(lib, "sentinel"))
	require.NoError(t, readErr)
	assert.Equal(t, "previous", string(contents))
}

func TestLibraryTransactionRemovesNewLibraryOnRollback(t *testing.T) {
	root := t.TempDir()
	lib := filepath.Join(root, "lib")
	staging := filepath.Join(root, "staging")
	require.NoError(t, os.Mkdir(staging, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(staging, "new"), []byte("new"), 0o600))

	txn := newLibraryTransaction(lib, staging)
	require.NoError(t, txn.swap())
	require.NoError(t, txn.rollback())
	_, err := os.Stat(lib)
	assert.ErrorIs(t, err, os.ErrNotExist)
}

func TestLibraryTransactionRestoresLibraryWhenSwapFails(t *testing.T) {
	root := t.TempDir()
	lib := filepath.Join(root, "lib")
	staging := filepath.Join(root, "staging")
	require.NoError(t, os.Mkdir(lib, 0o755))
	require.NoError(t, os.Mkdir(staging, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(lib, "sentinel"), []byte("previous"), 0o600))

	txn := newLibraryTransaction(lib, staging)
	swapErr := errors.New("swap failed")
	txn.rename = func(oldPath, newPath string) error {
		if oldPath == staging {
			return swapErr
		}
		return os.Rename(oldPath, newPath)
	}
	err := txn.swap()
	assert.ErrorIs(t, err, swapErr)
	contents, readErr := os.ReadFile(filepath.Join(lib, "sentinel"))
	require.NoError(t, readErr)
	assert.Equal(t, "previous", string(contents))
}
