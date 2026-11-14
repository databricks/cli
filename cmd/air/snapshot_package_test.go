package aircmd

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"os"
	"os/exec"
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

func TestCreateGitArchiveSnapshot_GitFailureSurfacesStderr(t *testing.T) {
	ctx := t.Context()
	repo := newTestRepo(t)
	writeRepoFile(t, repo, "a.txt", "1")
	commitAll(t, repo, "init")

	out := filepath.Join(t.TempDir(), "snap.tar.gz")
	err := createGitArchiveSnapshot(ctx, newGitRepo(repo), "0000000000000000000000000000000000000000", out, "repo", nil, "")
	require.ErrorContains(t, err, "failed to create git archive")
	assert.ErrorContains(t, err, "fatal")
}

func TestCreateGitArchiveSnapshotStopsWhenCancelled(t *testing.T) {
	repo := newTestRepo(t)
	writeRepoFile(t, repo, "a.txt", "1")
	sha := commitAll(t, repo, "init")

	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	err := createGitArchiveSnapshot(ctx, newGitRepo(repo), sha, filepath.Join(t.TempDir(), "snap.tar.gz"), "repo", nil, "")
	require.ErrorIs(t, err, context.Canceled)
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

func TestCreatePlainTarballStopsWhenCancelled(t *testing.T) {
	repo := t.TempDir()
	writeRepoFile(t, repo, "a.txt", "1")
	files, err := snapshotFiles(t.Context(), repo, nil, false)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	err = createPlainTarball(ctx, repo, filepath.Join(t.TempDir(), "snap.tar.gz"), files)
	require.ErrorIs(t, err, context.Canceled)
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

// ignoredProject returns a project dir that an enclosing repo's .gitignore excludes.
func ignoredProject(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()
	runGit(t, repo, "init", "-q")
	writeRepoFile(t, repo, ".gitignore", ".scratch/\n")
	writeRepoFile(t, repo, ".scratch/proj/main.py", "print(1)\n")
	return filepath.Join(repo, ".scratch", "proj")
}

func TestSnapshotFiles_RejectsGitIgnoredRoot(t *testing.T) {
	project := ignoredProject(t)

	_, err := snapshotFiles(t.Context(), project, nil, true)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "contains no files")
	assert.Contains(t, err.Error(), "ignored by git")
	// The matching rule and the command to inspect it are both named.
	assert.Contains(t, err.Error(), ".gitignore:1:.scratch/")
	assert.Contains(t, err.Error(), "check-ignore -v")
}

func TestSnapshotFiles_RejectsEmptyDirectory(t *testing.T) {
	_, err := snapshotFiles(t.Context(), t.TempDir(), nil, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "contains no files")
	assert.NotContains(t, err.Error(), "ignored by git")
}

// withoutGit makes the git binary look absent for the rest of the test.
func withoutGit(t *testing.T) {
	t.Helper()
	orig := gitLookPath
	gitLookPath = func(string) (string, error) { return "", exec.ErrNotFound }
	t.Cleanup(func() { gitLookPath = orig })
}

// writeFallbackFixture lays out a directory outside any repository that covers the cases
// the git-free listing must select as git does: root and nested .gitignore files,
// negation, an ignored directory, dotfiles, and symlinks that are listed but not followed.
func writeFallbackFixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	writeRepoFile(t, dir, ".gitignore", "build/\n*.log\n")
	writeRepoFile(t, dir, "keep.txt", "1")
	writeRepoFile(t, dir, "drop.log", "1")
	writeRepoFile(t, dir, "build/out.py", "1")
	writeRepoFile(t, dir, "README.md", "1")
	writeRepoFile(t, dir, ".dotfile", "1")
	writeRepoFile(t, dir, ".hidden/secret.txt", "1")
	writeRepoFile(t, dir, "a/x.py", "1")
	writeRepoFile(t, dir, "src/m.py", "1")
	writeRepoFile(t, dir, "src/deep/n.py", "1")
	writeRepoFile(t, dir, "nested/.gitignore", "*.tmp\n!keep.tmp\n")
	writeRepoFile(t, dir, "nested/drop.tmp", "1")
	writeRepoFile(t, dir, "nested/keep.tmp", "1")
	writeRepoFile(t, dir, "nested/inner/z.py", "1")
	require.NoError(t, os.Symlink("src/m.py", filepath.Join(dir, "lnk.py")))
	require.NoError(t, os.Symlink("src", filepath.Join(dir, "symdir")))
	return dir
}

// selectedNames returns the repo-relative paths snapshotFiles selects, sorted.
func selectedNames(t *testing.T, dir string, includePaths []string, isGitRepo bool) []string {
	t.Helper()
	files, err := snapshotFiles(t.Context(), dir, includePaths, isGitRepo)
	require.NoError(t, err)
	names := make([]string, 0, len(files))
	for _, f := range files {
		names = append(names, filepath.ToSlash(f.rel))
	}
	slices.Sort(names)
	return names
}

// The git listing and the git-free fallback select the same files for the common cases,
// and the git listing is pinned here so changing the fallback cannot change it.
func TestSnapshotFilesWithoutGit_MatchesGitListing(t *testing.T) {
	cases := []struct {
		name         string
		includePaths []string
		want         []string
	}{
		{
			name: "whole tree",
			want: []string{
				".dotfile", ".gitignore", ".hidden/secret.txt", "README.md", "a/x.py",
				"keep.txt", "lnk.py", "nested/.gitignore", "nested/inner/z.py",
				"nested/keep.tmp", "src/deep/n.py", "src/m.py", "symdir",
			},
		},
		{
			name:         "directory include_path",
			includePaths: []string{"src"},
			want:         []string{"src/deep/n.py", "src/m.py"},
		},
		{
			name:         "file include_path",
			includePaths: []string{"a/x.py"},
			want:         []string{"a/x.py"},
		},
		{
			name:         "nested include_path with nested rules",
			includePaths: []string{"nested"},
			want:         []string{"nested/.gitignore", "nested/inner/z.py", "nested/keep.tmp"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := writeFallbackFixture(t)
			assert.Equal(t, tc.want, selectedNames(t, dir, tc.includePaths, false), "git listing changed")

			withoutGit(t)
			assert.Equal(t, tc.want, selectedNames(t, dir, tc.includePaths, false), "git-free fallback")
		})
	}
}

// A nested repository is one "dir/" entry in both listings, and its contents are not walked.
func TestSnapshotFilesWithoutGit_NestedRepoIsOneEntry(t *testing.T) {
	dir := t.TempDir()
	writeRepoFile(t, dir, "top.py", "1")
	nested := filepath.Join(dir, "nested")
	require.NoError(t, os.MkdirAll(nested, 0o755))
	runGit(t, nested, "init", "-q")
	writeRepoFile(t, nested, "inner.py", "1")

	want := []string{"nested/", "top.py"}
	assert.Equal(t, want, selectedNames(t, dir, nil, false))
	withoutGit(t)
	assert.Equal(t, want, selectedNames(t, dir, nil, false))
}

// Known difference: a root *.log excludes nested/keep.log even though the nested file
// re-includes it, because libs/git returns the first matching rule from the root down.
func TestSnapshotFilesWithoutGit_NestedNegationDiffersFromGit(t *testing.T) {
	dir := t.TempDir()
	writeRepoFile(t, dir, ".gitignore", "*.log\n")
	writeRepoFile(t, dir, "nested/.gitignore", "!keep.log\n")
	writeRepoFile(t, dir, "nested/keep.log", "1")

	assert.Contains(t, selectedNames(t, dir, nil, false), "nested/keep.log")
	withoutGit(t)
	assert.NotContains(t, selectedNames(t, dir, nil, false), "nested/keep.log")
}

// Known difference: include_paths globs use path.Match, where "*" does not cross "/",
// so "*.py" selects only the root file (git's pathspec selects a/x.py too).
func TestSnapshotFilesWithoutGit_IncludeGlobDiffersFromGit(t *testing.T) {
	dir := t.TempDir()
	writeRepoFile(t, dir, "top.py", "1")
	writeRepoFile(t, dir, "a/x.py", "1")

	assert.Equal(t, []string{"a/x.py", "top.py"}, selectedNames(t, dir, []string{"*.py"}, false))
	withoutGit(t)
	assert.Equal(t, []string{"top.py"}, selectedNames(t, dir, []string{"*.py"}, false))
}

func TestSnapshotFiles_RejectsIncludePathsMatchingNothing(t *testing.T) {
	repo := t.TempDir()
	writeRepoFile(t, repo, "a.txt", "1")

	_, err := snapshotFiles(t.Context(), repo, []string{"missing"}, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "contains no files")
	assert.Contains(t, err.Error(), "include_paths")
}
