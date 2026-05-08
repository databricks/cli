package keys_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/experimental/ssh/internal/keys"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSaveSSHKeyPairNewFiles(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "session1")
	privateKey := []byte("private-key-content")
	publicKey := []byte("public-key-content")

	err := keys.SaveSSHKeyPair(keyPath, privateKey, publicKey)
	require.NoError(t, err)

	gotPrivate, err := os.ReadFile(keyPath)
	require.NoError(t, err)
	assert.Equal(t, privateKey, gotPrivate)

	gotPublic, err := os.ReadFile(keyPath + ".pub")
	require.NoError(t, err)
	assert.Equal(t, publicKey, gotPublic)
}

func TestSaveSSHKeyPairOverwritesExistingFiles(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "session1")

	// Write initial keys.
	require.NoError(t, keys.SaveSSHKeyPair(keyPath, []byte("old-private"), []byte("old-public")))

	// Overwrite with new keys.
	newPrivate := []byte("new-private-key-content")
	newPublic := []byte("new-public-key-content")
	err := keys.SaveSSHKeyPair(keyPath, newPrivate, newPublic)
	require.NoError(t, err)

	gotPrivate, err := os.ReadFile(keyPath)
	require.NoError(t, err)
	assert.Equal(t, newPrivate, gotPrivate)

	gotPublic, err := os.ReadFile(keyPath + ".pub")
	require.NoError(t, err)
	assert.Equal(t, newPublic, gotPublic)
}

func TestSaveSSHKeyPairCreatesDirectory(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "nonexistent-subdir", "session1")
	privateKey := []byte("private-key-content")
	publicKey := []byte("public-key-content")

	err := keys.SaveSSHKeyPair(keyPath, privateKey, publicKey)
	require.NoError(t, err)

	gotPrivate, err := os.ReadFile(keyPath)
	require.NoError(t, err)
	assert.Equal(t, privateKey, gotPrivate)

	gotPublic, err := os.ReadFile(keyPath + ".pub")
	require.NoError(t, err)
	assert.Equal(t, publicKey, gotPublic)
}

func TestSaveSSHKeyPairDoesNotAffectOtherSessions(t *testing.T) {
	dir := t.TempDir()
	keyPath1 := filepath.Join(dir, "session1")
	keyPath2 := filepath.Join(dir, "session2")

	require.NoError(t, keys.SaveSSHKeyPair(keyPath1, []byte("private-1"), []byte("public-1")))
	require.NoError(t, keys.SaveSSHKeyPair(keyPath2, []byte("private-2"), []byte("public-2")))

	// Overwrite session1 — session2 must be untouched.
	require.NoError(t, keys.SaveSSHKeyPair(keyPath1, []byte("private-1-new"), []byte("public-1-new")))

	gotPrivate2, err := os.ReadFile(keyPath2)
	require.NoError(t, err)
	assert.Equal(t, []byte("private-2"), gotPrivate2)

	gotPublic2, err := os.ReadFile(keyPath2 + ".pub")
	require.NoError(t, err)
	assert.Equal(t, []byte("public-2"), gotPublic2)
}
