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

// tarEntryNames returns the set of entry names in a gzipped tarball file.
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

func TestCreatePlainTarballPrefixesExcludesGitAndGitignored(t *testing.T) {
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
	require.NoError(t, createPlainTarball(t.Context(), code, out))

	names := tarEntryNames(t, out)
	// Entries are prefixed with the code dir basename (the runtime extracts to
	// /databricks/code_source/<dir>).
	assert.True(t, names["mycode/train.py"])
	assert.True(t, names["mycode/pkg/util.py"])
	assert.True(t, names["mycode/.gitignore"])
	assert.False(t, names["mycode/ignored.txt"], "gitignored file must be excluded")
	assert.False(t, names["mycode/debug.log"], "gitignored glob must be excluded")
	assert.False(t, names["mycode/.git/config"], ".git must never be archived")
	assert.False(t, names["mycode/._resource_fork"], "AppleDouble metadata must be excluded")
}
