package aircmd

import (
	"archive/tar"
	"compress/gzip"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// tarballEntries returns the sorted list of entry names in a .tar.gz.
func tarballEntries(t *testing.T, path string) []string {
	t.Helper()
	f, err := os.Open(path)
	require.NoError(t, err)
	defer f.Close()

	gz, err := gzip.NewReader(f)
	require.NoError(t, err)
	defer gz.Close()

	var names []string
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err != nil {
			break
		}
		names = append(names, hdr.Name)
	}
	slices.Sort(names)
	return names
}

// writeRepoFile writes a file at a dir-relative path, creating parent dirs.
func writeRepoFile(t *testing.T, dir, rel, content string) {
	t.Helper()
	full := filepath.Join(dir, rel)
	require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o755))
	require.NoError(t, os.WriteFile(full, []byte(content), 0o600))
}

func TestCreatePlainTarball(t *testing.T) {
	ctx := t.Context()
	repo := t.TempDir()
	writeRepoFile(t, repo, "a.txt", "1")
	writeRepoFile(t, repo, "src/model.py", "print()")
	writeRepoFile(t, repo, "dirty.txt", "wip")
	// A .git dir must never be shipped.
	writeRepoFile(t, repo, ".git/config", "x")

	out := filepath.Join(t.TempDir(), "snap.tar.gz")
	require.NoError(t, createPlainTarball(ctx, repo, out, nil))

	dirName := filepath.Base(repo)
	entries := tarballEntries(t, out)
	assert.Contains(t, entries, dirName+"/a.txt")
	assert.Contains(t, entries, dirName+"/dirty.txt")
	// .git is never shipped.
	for _, e := range entries {
		assert.NotContains(t, e, "/.git/")
	}
}

func TestCreatePlainTarball_HonorsGitignore(t *testing.T) {
	ctx := t.Context()
	repo := t.TempDir()
	writeRepoFile(t, repo, "keep.txt", "1")
	writeRepoFile(t, repo, "junk.log", "noise")
	writeRepoFile(t, repo, ".gitignore", "*.log\n")

	out := filepath.Join(t.TempDir(), "snap.tar.gz")
	require.NoError(t, createPlainTarball(ctx, repo, out, nil))

	dirName := filepath.Base(repo)
	entries := tarballEntries(t, out)
	assert.Contains(t, entries, dirName+"/keep.txt")
	assert.NotContains(t, entries, dirName+"/junk.log")
}

func TestCreatePlainTarball_IncludePaths(t *testing.T) {
	ctx := t.Context()
	repo := t.TempDir()
	writeRepoFile(t, repo, "a.txt", "1")
	writeRepoFile(t, repo, "src/model.py", "print()")

	out := filepath.Join(t.TempDir(), "snap.tar.gz")
	require.NoError(t, createPlainTarball(ctx, repo, out, []string{"src"}))

	dirName := filepath.Base(repo)
	entries := tarballEntries(t, out)
	assert.Contains(t, entries, dirName+"/src/model.py")
	assert.NotContains(t, entries, dirName+"/a.txt")
}

func TestParseGitignore(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".gitignore")
	content := "# comment\n" +
		"\n" +
		"*.log\n" +
		"!keep.log\n" + // negation: skipped
		"build/\n" + // trailing slash stripped
		"**/node_modules\n" + // **/foo -> foo
		"dist/**\n" + // foo/** -> foo
		"a/**/b\n" + // mid ** : skipped
		"src/config\n" // path-relative kept as-is
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))

	patterns, err := parseGitignore(path)
	require.NoError(t, err)
	assert.Equal(t, []string{
		"*.log",
		"build",
		"node_modules",
		"dist",
		"src/config",
	}, patterns)
}

func TestParseGitignore_Missing(t *testing.T) {
	_, err := parseGitignore(filepath.Join(t.TempDir(), "nope"))
	require.Error(t, err)
}
