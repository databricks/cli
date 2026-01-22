package state

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestComputeChecksum(t *testing.T) {
	dir := t.TempDir()

	// create client/ and server/ dirs
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "client"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "server"), 0o755))

	// create source files
	require.NoError(t, os.WriteFile(filepath.Join(dir, "client", "app.ts"), []byte("console.log('hello')"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "server", "main.ts"), []byte("export default {}"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name": "test"}`), 0o644))

	// compute checksum
	checksum1, err := ComputeChecksum(dir)
	require.NoError(t, err)
	assert.Len(t, checksum1, 64) // sha256 hex

	// same content = same checksum
	checksum2, err := ComputeChecksum(dir)
	require.NoError(t, err)
	assert.Equal(t, checksum1, checksum2)

	// modify file = different checksum
	require.NoError(t, os.WriteFile(filepath.Join(dir, "client", "app.ts"), []byte("console.log('changed')"), 0o644))
	checksum3, err := ComputeChecksum(dir)
	require.NoError(t, err)
	assert.NotEqual(t, checksum1, checksum3)
}

func TestComputeChecksumExcludesNodeModules(t *testing.T) {
	dir := t.TempDir()

	require.NoError(t, os.MkdirAll(filepath.Join(dir, "client"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "client", "node_modules"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "client", "app.ts"), []byte("code"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "client", "node_modules", "dep.js"), []byte("dependency"), 0o644))

	checksum1, err := ComputeChecksum(dir)
	require.NoError(t, err)

	// changing node_modules should not affect checksum
	require.NoError(t, os.WriteFile(filepath.Join(dir, "client", "node_modules", "dep.js"), []byte("changed"), 0o644))
	checksum2, err := ComputeChecksum(dir)
	require.NoError(t, err)

	assert.Equal(t, checksum1, checksum2)
}

func TestComputeChecksumEmptyProject(t *testing.T) {
	dir := t.TempDir()

	_, err := ComputeChecksum(dir)
	assert.ErrorContains(t, err, "no source files found")
}

func TestVerifyChecksum(t *testing.T) {
	dir := t.TempDir()

	require.NoError(t, os.MkdirAll(filepath.Join(dir, "client"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "client", "app.ts"), []byte("code"), 0o644))

	checksum, err := ComputeChecksum(dir)
	require.NoError(t, err)

	match, err := VerifyChecksum(dir, checksum)
	require.NoError(t, err)
	assert.True(t, match)

	match, err = VerifyChecksum(dir, "wrong")
	require.NoError(t, err)
	assert.False(t, match)
}
