package dbconnect

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUvArgs(t *testing.T) {
	m := &uvManager{bin: "uv"}
	assert.Equal(t, []string{"sync"}, m.syncArgs())
	assert.Equal(t, []string{"python", "install", "3.12"}, m.pythonInstallArgs("3.12"))
	assert.Equal(t, []string{"pip", "install", "pip", "--python", "/p/.venv/bin/python"}, m.pipSeedArgs("/p/.venv/bin/python"))
}

func TestDiscoverUvFindsBinOnPath(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "uv")
	require.NoError(t, os.WriteFile(bin, []byte("#!/bin/sh\n"), 0o755))
	t.Setenv("PATH", dir)
	got, err := discoverUv(t.Context())
	require.NoError(t, err)
	assert.Equal(t, bin, got)
}
