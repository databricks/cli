package keys_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/experimental/ssh/internal/keys"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSaveSSHKeyPairCreatesFiles(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "session1")

	err := keys.SaveSSHKeyPair(keyPath, []byte("private"), []byte("public"))
	require.NoError(t, err)

	priv, err := os.ReadFile(keyPath)
	require.NoError(t, err)
	assert.Equal(t, "private", string(priv))

	pub, err := os.ReadFile(keyPath + ".pub")
	require.NoError(t, err)
	assert.Equal(t, "public", string(pub))
}

func TestSaveSSHKeyPairOverwritesExistingFiles(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "session1")

	err := keys.SaveSSHKeyPair(keyPath, []byte("old-private"), []byte("old-public"))
	require.NoError(t, err)

	err = keys.SaveSSHKeyPair(keyPath, []byte("new-private"), []byte("new-public"))
	require.NoError(t, err)

	priv, err := os.ReadFile(keyPath)
	require.NoError(t, err)
	assert.Equal(t, "new-private", string(priv))

	pub, err := os.ReadFile(keyPath + ".pub")
	require.NoError(t, err)
	assert.Equal(t, "new-public", string(pub))
}

func TestSaveSSHKeyPairDoesNotDeleteOtherSessions(t *testing.T) {
	dir := t.TempDir()

	// Save keys for session1.
	err := keys.SaveSSHKeyPair(filepath.Join(dir, "session1"), []byte("priv1"), []byte("pub1"))
	require.NoError(t, err)

	// Save keys for session2 in the same directory.
	err = keys.SaveSSHKeyPair(filepath.Join(dir, "session2"), []byte("priv2"), []byte("pub2"))
	require.NoError(t, err)

	// session1 keys must still exist.
	priv, err := os.ReadFile(filepath.Join(dir, "session1"))
	require.NoError(t, err)
	assert.Equal(t, "priv1", string(priv))

	pub, err := os.ReadFile(filepath.Join(dir, "session1.pub"))
	require.NoError(t, err)
	assert.Equal(t, "pub1", string(pub))
}

func TestSaveSSHKeyPairCreatesParentDirectory(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "nested", "dir", "session1")

	err := keys.SaveSSHKeyPair(keyPath, []byte("private"), []byte("public"))
	require.NoError(t, err)

	_, err = os.ReadFile(keyPath)
	require.NoError(t, err)
}

func TestSaveSSHKeyPairFilePermissions(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "session1")

	err := keys.SaveSSHKeyPair(keyPath, []byte("private"), []byte("public"))
	require.NoError(t, err)

	info, err := os.Stat(keyPath)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())

	info, err = os.Stat(keyPath + ".pub")
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o644), info.Mode().Perm())
}
