package aicode

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// tarEntryNames returns the sorted set of entry names in a gzipped tarball file.
func tarEntryNames(t *testing.T, tarballPath string) map[string]bool {
	t.Helper()
	f, err := os.Open(tarballPath)
	require.NoError(t, err)
	defer f.Close()
	gzr, err := gzip.NewReader(f)
	require.NoError(t, err)
	tr := tar.NewReader(gzr)
	names := map[string]bool{}
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		names[hdr.Name] = true
	}
	return names
}

func TestCreatePlainTarballExcludesGitAndGitignored(t *testing.T) {
	dir := t.TempDir()
	code := filepath.Join(dir, "mycode")
	require.NoError(t, os.MkdirAll(filepath.Join(code, "pkg"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(code, ".git"), 0o755))
	write := func(rel, content string) {
		require.NoError(t, os.WriteFile(filepath.Join(code, rel), []byte(content), 0o644))
	}
	write("train.py", "print('train')")
	write("pkg/util.py", "x = 1")
	write(".gitignore", "ignored.txt\n*.log\n")
	write("ignored.txt", "nope")
	write("debug.log", "nope")
	write(".git/config", "[core]")
	write("._resource_fork", "apple double")

	out := filepath.Join(dir, "out.tar.gz")
	require.NoError(t, createPlainTarball(t.Context(), code, out, nil))

	names := tarEntryNames(t, out)
	assert.True(t, names["mycode/train.py"])
	assert.True(t, names["mycode/pkg/util.py"])
	assert.True(t, names["mycode/.gitignore"])
	assert.False(t, names["mycode/ignored.txt"], "gitignored file must be excluded")
	assert.False(t, names["mycode/debug.log"], "gitignored glob must be excluded")
	assert.False(t, names["mycode/.git/config"], ".git must never be archived")
	assert.False(t, names["mycode/._resource_fork"], "AppleDouble metadata must be excluded")
}

func TestCreateGitArchiveSnapshotPrefixesAndScopesToSubdir(t *testing.T) {
	// Repo with two top-level dirs; archiving the "src" subdir must include only
	// src, prefixed by its basename (the runtime extracts to code_source/<dir>).
	repo := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(repo, "src"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(repo, "other"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(repo, "src", "a.py"), []byte("a"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(repo, "other", "b.py"), []byte("b"), 0o644))
	commit := initGitRepo(t, repo)

	src := filepath.Join(repo, "src")
	out := filepath.Join(t.TempDir(), "out.tar.gz")
	require.NoError(t, createGitArchiveSnapshot(t.Context(), newGitRepo(src), commit, out, "src", nil))

	names := tarEntryNames(t, out)
	assert.True(t, names["src/a.py"])
	assert.False(t, names["other/b.py"], "sibling dir must not be in a src-scoped archive")
	assert.False(t, names["src/../other/b.py"])
}
