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

// writeCodeDir materializes files (relative path -> content) under a fresh temp
// directory and returns a vfs.Path rooted at it.
func writeCodeDir(t *testing.T, files map[string]string) vfs.Path {
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

func TestBuildTarballPrefixesEntriesAndOmitsGitAndGitignored(t *testing.T) {
	root := writeCodeDir(t, map[string]string{
		"train.py":        "print('train')",
		"pkg/util.py":     "x = 1",
		".gitignore":      "ignored.txt\n*.log\n",
		"ignored.txt":     "should not be archived",
		"debug.log":       "should not be archived",
		".git/config":     "[core]",
		"._resource_fork": "apple double metadata",
	})

	var buf bytes.Buffer
	sha, err := buildTarball(t.Context(), root, "mycode", &buf)
	require.NoError(t, err)
	require.NotEmpty(t, sha)

	entries := tarEntries(t, buf.Bytes())
	assert.Equal(t, "print('train')", entries["mycode/train.py"])
	assert.Equal(t, "x = 1", entries["mycode/pkg/util.py"])
	// .gitignore itself is a tracked file and is archived.
	assert.Contains(t, entries, "mycode/.gitignore")

	assert.NotContains(t, entries, "mycode/ignored.txt", "gitignored file must be excluded")
	assert.NotContains(t, entries, "mycode/debug.log", "gitignored glob must be excluded")
	assert.NotContains(t, entries, "mycode/.git/config", ".git must never be archived")
	assert.NotContains(t, entries, "mycode/._resource_fork", "AppleDouble metadata must be excluded")
}

func TestBuildTarballIsDeterministic(t *testing.T) {
	files := map[string]string{
		"a.py":      "aaa",
		"sub/b.py":  "bbb",
		"sub/c.txt": "ccc",
	}
	root1 := writeCodeDir(t, files)
	root2 := writeCodeDir(t, files)

	var buf1, buf2 bytes.Buffer
	sha1, err := buildTarball(t.Context(), root1, "code", &buf1)
	require.NoError(t, err)
	sha2, err := buildTarball(t.Context(), root2, "code", &buf2)
	require.NoError(t, err)

	assert.Equal(t, sha1, sha2, "identical content must produce an identical hash")
	assert.Equal(t, buf1.Bytes(), buf2.Bytes(), "identical content must produce identical bytes")
}

func TestBuildTarballHashChangesWithContent(t *testing.T) {
	root1 := writeCodeDir(t, map[string]string{"main.py": "v1"})
	root2 := writeCodeDir(t, map[string]string{"main.py": "v2"})

	var buf1, buf2 bytes.Buffer
	sha1, err := buildTarball(t.Context(), root1, "code", &buf1)
	require.NoError(t, err)
	sha2, err := buildTarball(t.Context(), root2, "code", &buf2)
	require.NoError(t, err)

	assert.NotEqual(t, sha1, sha2)
}
