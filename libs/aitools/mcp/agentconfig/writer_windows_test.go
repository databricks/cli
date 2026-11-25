//go:build windows

package agentconfig

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sys/windows"
)

// holdUnshared opens the file at path without sharing, as a rename over it does, and closes it after 100 ms.
func holdUnshared(t *testing.T, path string) {
	name, err := windows.UTF16PtrFromString(path)
	require.NoError(t, err)
	h, err := windows.CreateFile(name, windows.GENERIC_READ, 0, nil, windows.OPEN_EXISTING, windows.FILE_ATTRIBUTE_NORMAL, 0)
	require.NoError(t, err)
	time.AfterFunc(100*time.Millisecond, func() { _ = windows.CloseHandle(h) })
}

// TestReadConfigWaitsForFileInUse checks that a read of a config that is in use returns the file once it is closed,
// instead of failing with a sharing violation.
func TestReadConfigWaitsForFileInUse(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	writeFile(t, path, []byte(`{"a": 1}`), 0o600)
	holdUnshared(t, path)

	data, err := readConfig(path)
	require.NoError(t, err)
	assert.JSONEq(t, `{"a": 1}`, string(data))
}

// TestCheckWritableWaitsForFileInUse checks that the write check of a config that is in use passes once the file is
// closed, instead of failing with a sharing violation.
func TestCheckWritableWaitsForFileInUse(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	writeFile(t, path, []byte(`{"a": 1}`), 0o600)
	holdUnshared(t, path)

	require.NoError(t, checkWritable(path))
}
