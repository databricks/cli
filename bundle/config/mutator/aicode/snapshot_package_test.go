package aicode

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/libs/fileset"
	"github.com/databricks/cli/libs/vfs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// writeTree materializes files (relative slash path -> content) under a fresh
// temp dir and returns a vfs.Path rooted at it plus the fileset for its contents.
func writeTree(t *testing.T, files map[string]string) (vfs.Path, []fileset.File) {
	t.Helper()
	dir := t.TempDir()
	for name, content := range files {
		p := filepath.Join(dir, filepath.FromSlash(name))
		require.NoError(t, os.MkdirAll(filepath.Dir(p), 0o755))
		require.NoError(t, os.WriteFile(p, []byte(content), 0o644))
	}
	root := vfs.MustNew(dir)
	fs, err := fileset.New(root).Files()
	require.NoError(t, err)
	return root, fs
}

// tarEntries reads a gzipped tarball and returns entry name -> content.
func tarEntries(t *testing.T, b []byte) map[string]string {
	t.Helper()
	gzr, err := gzip.NewReader(bytes.NewReader(b))
	require.NoError(t, err)
	tr := tar.NewReader(gzr)
	out := map[string]string{}
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		content, err := io.ReadAll(tr)
		require.NoError(t, err)
		out[hdr.Name] = string(content)
	}
	return out
}

// tarModes reads a gzipped tarball and returns entry name -> permission bits.
func tarModes(t *testing.T, b []byte) map[string]int64 {
	t.Helper()
	gzr, err := gzip.NewReader(bytes.NewReader(b))
	require.NoError(t, err)
	tr := tar.NewReader(gzr)
	out := map[string]int64{}
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		out[hdr.Name] = hdr.Mode & 0o777
	}
	return out
}

// An executable file keeps the execute bit (0o755) so a bundled helper the user
// invokes still runs; a non-executable file is normalized to 0o644.
func TestBuildCodeSnapshotPreservesExecuteBit(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "run.sh"), []byte("#!/bin/sh\n"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "train.py"), []byte("x"), 0o644))
	root := vfs.MustNew(dir)
	files, err := fileset.New(root).Files()
	require.NoError(t, err)

	var buf bytes.Buffer
	_, err = buildCodeSnapshot(root, ".", files, "code", &buf)
	require.NoError(t, err)

	modes := tarModes(t, buf.Bytes())
	assert.Equal(t, int64(0o755), modes["code/run.sh"], "executable helper must keep its execute bit")
	assert.Equal(t, int64(0o644), modes["code/train.py"], "non-executable file is normalized to 0o644")
}

func TestBuildCodeSnapshotPrefixesEntries(t *testing.T) {
	root, files := writeTree(t, map[string]string{
		"train.py":        "print('train')",
		"pkg/util.py":     "x = 1",
		"._resource_fork": "apple double",
	})

	var buf bytes.Buffer
	sha, err := buildCodeSnapshot(root, ".", files, "mycode", &buf)
	require.NoError(t, err)
	require.NotEmpty(t, sha)

	entries := tarEntries(t, buf.Bytes())
	// Entries are prefixed with the code dir basename (runtime extracts to
	// /databricks/code_source/<dir>).
	assert.Equal(t, "print('train')", entries["mycode/train.py"])
	assert.Equal(t, "x = 1", entries["mycode/pkg/util.py"])
	assert.NotContains(t, entries, "mycode/._resource_fork", "AppleDouble metadata must be excluded")
}

func TestBuildCodeSnapshotRebasesUnderRelBase(t *testing.T) {
	// Files listed relative to a sync root; only the "src" subtree is packaged and
	// re-based so entries nest under the prefix (not the intermediate "src/").
	root, files := writeTree(t, map[string]string{
		"src/train.py":    "t",
		"src/pkg/util.py": "u",
		"other/ignore.py": "o",
	})

	var buf bytes.Buffer
	_, err := buildCodeSnapshot(root, "src", files, "src", &buf)
	require.NoError(t, err)

	entries := tarEntries(t, buf.Bytes())
	assert.Contains(t, entries, "src/train.py")
	assert.Contains(t, entries, "src/pkg/util.py")
	// A file outside relBase is not under "src/", so it is skipped.
	assert.NotContains(t, entries, "src/other/ignore.py")
	assert.NotContains(t, entries, "other/ignore.py")
}

func TestBuildCodeSnapshotIsReproducible(t *testing.T) {
	files := map[string]string{"a.py": "aaa", "sub/b.py": "bbb"}
	root1, fs1 := writeTree(t, files)
	root2, fs2 := writeTree(t, files)

	var buf1, buf2 bytes.Buffer
	sha1, err := buildCodeSnapshot(root1, ".", fs1, "code", &buf1)
	require.NoError(t, err)
	sha2, err := buildCodeSnapshot(root2, ".", fs2, "code", &buf2)
	require.NoError(t, err)

	assert.Equal(t, sha1, sha2, "identical content must produce an identical hash")
	assert.Equal(t, buf1.Bytes(), buf2.Bytes(), "identical content must produce identical bytes")
}

func TestBuildCodeSnapshotHashChangesWithContent(t *testing.T) {
	root1, fs1 := writeTree(t, map[string]string{"main.py": "v1"})
	root2, fs2 := writeTree(t, map[string]string{"main.py": "v2"})

	var buf1, buf2 bytes.Buffer
	sha1, err := buildCodeSnapshot(root1, ".", fs1, "code", &buf1)
	require.NoError(t, err)
	sha2, err := buildCodeSnapshot(root2, ".", fs2, "code", &buf2)
	require.NoError(t, err)

	assert.NotEqual(t, sha1, sha2)
}
