package aicode

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/libs/vfs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// writeTree materializes files (relative slash path -> content) under a fresh
// temp dir and returns a vfs.Path rooted at it.
func writeTree(t *testing.T, files map[string]string) vfs.Path {
	t.Helper()
	dir := t.TempDir()
	for name, content := range files {
		p := filepath.Join(dir, filepath.FromSlash(name))
		require.NoError(t, os.MkdirAll(filepath.Dir(p), 0o755))
		require.NoError(t, os.WriteFile(p, []byte(content), 0o644))
	}
	return vfs.MustNew(dir)
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

func TestBuildCodeSnapshotPrefixesAndExcludes(t *testing.T) {
	root := writeTree(t, map[string]string{
		"train.py":        "print('train')",
		"pkg/util.py":     "x = 1",
		".gitignore":      "secrets.txt\n*.log\n",
		"secrets.txt":     "nope",
		"debug.log":       "nope",
		"pkg/nested.log":  "nope",
		".git/config":     "[core]",
		"._resource_fork": "apple double",
	})

	var buf bytes.Buffer
	sha, err := buildCodeSnapshot(t.Context(), root, "mycode", &buf)
	require.NoError(t, err)
	require.NotEmpty(t, sha)

	entries := tarEntries(t, buf.Bytes())
	// Entries are prefixed with the code dir basename (runtime extracts to
	// /databricks/code_source/<dir>).
	assert.Equal(t, "print('train')", entries["mycode/train.py"])
	assert.Equal(t, "x = 1", entries["mycode/pkg/util.py"])
	assert.Contains(t, entries, "mycode/.gitignore")

	assert.NotContains(t, entries, "mycode/secrets.txt", "gitignored file must be excluded")
	assert.NotContains(t, entries, "mycode/debug.log", "gitignored glob must be excluded")
	assert.NotContains(t, entries, "mycode/pkg/nested.log", "gitignored glob must be excluded in subdirs")
	assert.NotContains(t, entries, "mycode/.git/config", ".git must never be archived")
	assert.NotContains(t, entries, "mycode/._resource_fork", "AppleDouble metadata must be excluded")
}

func TestBuildCodeSnapshotHonorsNestedGitignore(t *testing.T) {
	// A .gitignore in a subdirectory applies within that subtree — this is the
	// case the previous shell-tar implementation missed.
	root := writeTree(t, map[string]string{
		"a.py":           "a",
		"pkg/keep.py":    "keep",
		"pkg/.gitignore": "drop.py\n",
		"pkg/drop.py":    "nope",
	})

	var buf bytes.Buffer
	_, err := buildCodeSnapshot(t.Context(), root, "code", &buf)
	require.NoError(t, err)

	entries := tarEntries(t, buf.Bytes())
	assert.Contains(t, entries, "code/pkg/keep.py")
	assert.NotContains(t, entries, "code/pkg/drop.py", "nested .gitignore must be honored")
}

func TestBuildCodeSnapshotIsReproducible(t *testing.T) {
	files := map[string]string{"a.py": "aaa", "sub/b.py": "bbb"}
	root1 := writeTree(t, files)
	root2 := writeTree(t, files)

	var buf1, buf2 bytes.Buffer
	sha1, err := buildCodeSnapshot(t.Context(), root1, "code", &buf1)
	require.NoError(t, err)
	sha2, err := buildCodeSnapshot(t.Context(), root2, "code", &buf2)
	require.NoError(t, err)

	assert.Equal(t, sha1, sha2, "identical content must produce an identical hash")
	assert.Equal(t, buf1.Bytes(), buf2.Bytes(), "identical content must produce identical bytes")
}

func TestBuildCodeSnapshotHashChangesWithContent(t *testing.T) {
	root1 := writeTree(t, map[string]string{"main.py": "v1"})
	root2 := writeTree(t, map[string]string{"main.py": "v2"})

	var buf1, buf2 bytes.Buffer
	sha1, err := buildCodeSnapshot(t.Context(), root1, "code", &buf1)
	require.NoError(t, err)
	sha2, err := buildCodeSnapshot(t.Context(), root2, "code", &buf2)
	require.NoError(t, err)

	assert.NotEqual(t, sha1, sha2)
}
