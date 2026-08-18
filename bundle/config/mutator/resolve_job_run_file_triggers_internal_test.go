package mutator

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFingerprintFileContentStableAcrossTouch(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "seed.txt")
	require.NoError(t, os.WriteFile(path, []byte("v1"), 0o644))

	info, err := os.Stat(path)
	require.NoError(t, err)
	first, err := fingerprintFile(path, info, resources.JobRunFileFingerprint{})
	require.NoError(t, err)
	require.NotEmpty(t, first.Hash)

	// Advance mtime without changing contents (touch).
	require.NoError(t, os.Chtimes(path, time.Now().Add(time.Minute), time.Now().Add(time.Minute)))
	info, err = os.Stat(path)
	require.NoError(t, err)
	assert.NotEqual(t, first.MtimeNs, info.ModTime().UnixNano())

	second, err := fingerprintFile(path, info, first)
	require.NoError(t, err)
	assert.Equal(t, first, second, "unchanged content must reuse the previous fingerprint")
}

func TestFingerprintFileFastPathSkipsWhenMtimeAndSizeMatch(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "seed.txt")
	require.NoError(t, os.WriteFile(path, []byte("v1"), 0o644))

	info, err := os.Stat(path)
	require.NoError(t, err)
	prev := resources.JobRunFileFingerprint{
		Hash:    "not-the-real-hash",
		Size:    info.Size(),
		MtimeNs: info.ModTime().UnixNano(),
	}

	got, err := fingerprintFile(path, info, prev)
	require.NoError(t, err)
	assert.Equal(t, prev, got, "matching size+mtime must reuse prev without re-hashing")
}

func TestFingerprintFileContentChange(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "seed.txt")
	require.NoError(t, os.WriteFile(path, []byte("v1"), 0o644))

	info, err := os.Stat(path)
	require.NoError(t, err)
	first, err := fingerprintFile(path, info, resources.JobRunFileFingerprint{})
	require.NoError(t, err)

	// Different size so the mtime+size fast path cannot reuse prev.
	require.NoError(t, os.WriteFile(path, []byte("v2-changed"), 0o644))
	info, err = os.Stat(path)
	require.NoError(t, err)
	second, err := fingerprintFile(path, info, first)
	require.NoError(t, err)
	assert.NotEqual(t, first.Hash, second.Hash)
	assert.Equal(t, info.Size(), second.Size)
	assert.Equal(t, info.ModTime().UnixNano(), second.MtimeNs)
}
