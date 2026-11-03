package aircmd

import (
	"archive/tar"
	"compress/gzip"
	"os"
	"path/filepath"
	"slices"
	"strings"
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

func TestCreateGitArchiveSnapshot(t *testing.T) {
	ctx := t.Context()
	repo := newTestRepo(t)
	writeRepoFile(t, repo, "a.txt", "1")
	writeRepoFile(t, repo, "src/model.py", "print()")
	sha := commitAll(t, repo, "init")

	out := filepath.Join(t.TempDir(), "snap.tar.gz")
	dirName := filepath.Base(repo)
	require.NoError(t, createGitArchiveSnapshot(ctx, newGitRepo(repo), sha, out, dirName, nil, ""))

	entries := tarballEntries(t, out)
	// Every real entry is prefixed with the directory name. git archive also emits a
	// `pax_global_header` pseudo-entry (no prefix) that tar ignores on extraction.
	assert.Contains(t, entries, dirName+"/a.txt")
	assert.Contains(t, entries, dirName+"/src/model.py")
	for _, e := range entries {
		if e == "pax_global_header" {
			continue
		}
		assert.True(t, strings.HasPrefix(e, dirName+"/"), "entry %q lacks prefix", e)
	}
}

func TestCreateGitArchiveSnapshot_IncludePaths(t *testing.T) {
	ctx := t.Context()
	repo := newTestRepo(t)
	writeRepoFile(t, repo, "a.txt", "1")
	writeRepoFile(t, repo, "src/model.py", "print()")
	sha := commitAll(t, repo, "init")

	out := filepath.Join(t.TempDir(), "snap.tar.gz")
	dirName := filepath.Base(repo)
	require.NoError(t, createGitArchiveSnapshot(ctx, newGitRepo(repo), sha, out, dirName, []string{"src"}, ""))

	entries := tarballEntries(t, out)
	assert.Contains(t, entries, dirName+"/src/model.py")
	assert.NotContains(t, entries, dirName+"/a.txt")
}

func TestCreateGitArchiveSnapshot_SubdirectoryRootPath(t *testing.T) {
	ctx := t.Context()
	repo := newTestRepo(t)
	writeRepoFile(t, repo, "README.md", "repo root")
	writeRepoFile(t, repo, "subpkg/train.py", "print()")
	writeRepoFile(t, repo, "subpkg/nested/util.py", "pass")
	sha := commitAll(t, repo, "init")

	rootPath := filepath.Join(repo, "subpkg")
	prefix, err := newGitRepo(rootPath).repoRelativePrefix(ctx)
	require.NoError(t, err)

	out := filepath.Join(t.TempDir(), "snap.tar.gz")
	require.NoError(t, createGitArchiveSnapshot(ctx, newGitRepo(rootPath), sha, out, "subpkg", nil, prefix))

	entries := tarballEntries(t, out)
	assert.Contains(t, entries, "subpkg/train.py")
	assert.Contains(t, entries, "subpkg/nested/util.py")
	assert.NotContains(t, entries, "subpkg/README.md")
	for _, entry := range entries {
		assert.False(t, strings.HasPrefix(entry, "subpkg/subpkg/"), "entry %q is double nested", entry)
	}
}

func TestCreateGitArchiveSnapshot_SubdirectoryRootPathWithIncludePaths(t *testing.T) {
	ctx := t.Context()
	repo := newTestRepo(t)
	writeRepoFile(t, repo, "subpkg/train.py", "print()")
	writeRepoFile(t, repo, "subpkg/src/model.py", "pass")
	writeRepoFile(t, repo, "subpkg/configs/train.yaml", "x")
	sha := commitAll(t, repo, "init")

	rootPath := filepath.Join(repo, "subpkg")
	prefix, err := newGitRepo(rootPath).repoRelativePrefix(ctx)
	require.NoError(t, err)

	out := filepath.Join(t.TempDir(), "snap.tar.gz")
	require.NoError(t, createGitArchiveSnapshot(ctx, newGitRepo(rootPath), sha, out, "subpkg", []string{"src"}, prefix))

	entries := tarballEntries(t, out)
	assert.Contains(t, entries, "subpkg/src/model.py")
	assert.NotContains(t, entries, "subpkg/train.py")
	assert.NotContains(t, entries, "subpkg/configs/train.yaml")
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
	files, err := snapshotFiles(ctx, repo, nil, false)
	require.NoError(t, err)
	require.NoError(t, createPlainTarball(ctx, repo, out, files))

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
	files, err := snapshotFiles(ctx, repo, nil, false)
	require.NoError(t, err)
	require.NoError(t, createPlainTarball(ctx, repo, out, files))

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
	files, err := snapshotFiles(ctx, repo, []string{"src"}, false)
	require.NoError(t, err)
	require.NoError(t, createPlainTarball(ctx, repo, out, files))

	dirName := filepath.Base(repo)
	entries := tarballEntries(t, out)
	assert.Contains(t, entries, dirName+"/src/model.py")
	assert.NotContains(t, entries, dirName+"/a.txt")
}

func TestCreatePlainTarball_HonorsNestedGitignoreAndNegation(t *testing.T) {
	repo := t.TempDir()
	writeRepoFile(t, repo, ".gitignore", "*.log\n!keep.log\n/root-only.txt\n")
	writeRepoFile(t, repo, "drop.log", "drop")
	writeRepoFile(t, repo, "keep.log", "keep")
	writeRepoFile(t, repo, "root-only.txt", "drop")
	writeRepoFile(t, repo, "nested/root-only.txt", "keep")
	writeRepoFile(t, repo, "nested/.gitignore", "*.tmp\n!keep.tmp\n")
	writeRepoFile(t, repo, "nested/drop.tmp", "drop")
	writeRepoFile(t, repo, "nested/keep.tmp", "keep")

	out := filepath.Join(t.TempDir(), "snap.tar.gz")
	files, err := snapshotFiles(t.Context(), repo, nil, false)
	require.NoError(t, err)
	require.NoError(t, createPlainTarball(t.Context(), repo, out, files))

	dirName := filepath.Base(repo)
	entries := tarballEntries(t, out)
	assert.NotContains(t, entries, dirName+"/drop.log")
	assert.Contains(t, entries, dirName+"/keep.log")
	assert.NotContains(t, entries, dirName+"/root-only.txt")
	assert.Contains(t, entries, dirName+"/nested/root-only.txt")
	assert.NotContains(t, entries, dirName+"/nested/drop.tmp")
	assert.Contains(t, entries, dirName+"/nested/keep.tmp")
}

func TestCreatePlainTarball_SkipsDeletedTrackedFiles(t *testing.T) {
	repo := newTestRepo(t)
	writeRepoFile(t, repo, "keep.txt", "keep")
	writeRepoFile(t, repo, "deleted.txt", "deleted")
	commitAll(t, repo, "init")
	require.NoError(t, os.Remove(filepath.Join(repo, "deleted.txt")))

	out := filepath.Join(t.TempDir(), "snap.tar.gz")
	files, err := snapshotFiles(t.Context(), repo, nil, true)
	require.NoError(t, err)
	require.NoError(t, createPlainTarball(t.Context(), repo, out, files))

	dirName := filepath.Base(repo)
	entries := tarballEntries(t, out)
	assert.Contains(t, entries, dirName+"/keep.txt")
	assert.NotContains(t, entries, dirName+"/deleted.txt")
}

func TestSnapshotFilesCapturesModeTypeAndSymlinkTarget(t *testing.T) {
	repo := t.TempDir()
	writeRepoFile(t, repo, "run.sh", "#!/bin/sh\n")
	require.NoError(t, os.Chmod(filepath.Join(repo, "run.sh"), 0o755))
	require.NoError(t, os.Symlink("run.sh", filepath.Join(repo, "current")))

	files, err := snapshotFiles(t.Context(), repo, nil, false)
	require.NoError(t, err)
	byName := make(map[string]snapshotFile, len(files))
	for _, file := range files {
		byName[filepath.ToSlash(file.rel)] = file
	}

	runInfo, err := os.Lstat(filepath.Join(repo, "run.sh"))
	require.NoError(t, err)
	assert.Equal(t, uint32(runInfo.Mode()), byName["run.sh"].mode)
	assert.Empty(t, byName["run.sh"].linkTarget)

	linkInfo, err := os.Lstat(filepath.Join(repo, "current"))
	require.NoError(t, err)
	assert.Equal(t, uint32(linkInfo.Mode()), byName["current"].mode)
	assert.NotZero(t, os.FileMode(byName["current"].mode)&os.ModeSymlink)
	assert.Equal(t, "run.sh", byName["current"].linkTarget)
}
