package aircmd

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"os"
	"path/filepath"
	"runtime"
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

func TestSnapshotFiles_NonGitRootFromSubdirectory(t *testing.T) {
	// The CLI often runs from the workload's directory, inside root_path. sub/main.py
	// shadows main.py, so a listing relative to the working directory picks the wrong
	// files instead of failing.
	tests := []struct {
		name         string
		includePaths []string
		want         []string
	}{
		{name: "include_paths", includePaths: []string{"main.py", "data"}, want: []string{"data/x.csv", "main.py"}},
		{name: "whole root", want: []string{"data/x.csv", "main.py", "sub/main.py"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			writeRepoFile(t, root, "main.py", "print('root')")
			writeRepoFile(t, root, "sub/main.py", "print('sub')")
			writeRepoFile(t, root, "data/x.csv", "a,b")
			t.Chdir(filepath.Join(root, "sub"))

			files, err := snapshotFiles(t.Context(), root, tt.includePaths, false)
			require.NoError(t, err)
			var got []string
			for _, f := range files {
				got = append(got, filepath.ToSlash(f.rel))
			}
			slices.Sort(got)
			assert.Equal(t, tt.want, got)
		})
	}
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

func TestSnapshotFiles_RejectsIncludePathsMatchingNothing(t *testing.T) {
	repo := t.TempDir()
	writeRepoFile(t, repo, "a.txt", "1")

	_, err := snapshotFiles(t.Context(), repo, []string{"missing"}, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "contains no files")
	assert.Contains(t, err.Error(), "include_paths")
}

func TestSnapshotFiles_RejectsIncludePathMatchingNothingAlongsideOthers(t *testing.T) {
	repo := t.TempDir()
	writeRepoFile(t, repo, "ray_bootstrap.sh", "#!/bin/bash\n")
	writeRepoFile(t, repo, "my-job/main.py", "print(1)\n")

	_, err := snapshotFiles(t.Context(), repo, []string{"ray_bootstrap.sh", "my_job"}, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "include_paths select no files")
	assert.Contains(t, err.Error(), ": my_job.")
	assert.NotContains(t, err.Error(), "ray_bootstrap.sh")
}

func TestSnapshotFiles_RejectsIncludePathWhoseFilesAreIgnored(t *testing.T) {
	repo := newTestRepo(t)
	writeRepoFile(t, repo, ".gitignore", "build/\n")
	writeRepoFile(t, repo, "main.py", "print(1)\n")
	writeRepoFile(t, repo, "build/out.bin", "x")

	_, err := snapshotFiles(t.Context(), repo, []string{"main.py", "build"}, true)
	require.Error(t, err)
	assert.Contains(t, err.Error(), ": build.")
	assert.Contains(t, err.Error(), "check-ignore -v")
}

func TestSnapshotFiles_AcceptsIncludePathForms(t *testing.T) {
	repo := t.TempDir()
	writeRepoFile(t, repo, "src/model.py", "print()")
	writeRepoFile(t, repo, "README.md", "readme")
	writeRepoFile(t, repo, "notes.txt", "notes")

	files, err := snapshotFiles(t.Context(), repo, []string{"./src/", "README.md", "*.txt"}, false)
	require.NoError(t, err)
	var rels []string
	for _, file := range files {
		rels = append(rels, filepath.ToSlash(file.rel))
	}
	assert.ElementsMatch(t, []string{"src/model.py", "README.md", "notes.txt"}, rels)
}

func TestSnapshotFiles_IncludePathMatching(t *testing.T) {
	tests := []struct {
		name         string
		includePaths []string
		want         []string // selected files; nil when an error is expected
		wantErr      string
	}{
		{name: "root", includePaths: []string{"."}, want: []string{"README.md", "b.txt", "data_v2/a.csv", "notes.txt", "src/model.py", "src/tests/test_model.py"}},
		{name: "root with slash", includePaths: []string{"./"}, want: []string{"README.md", "b.txt", "data_v2/a.csv", "notes.txt", "src/model.py", "src/tests/test_model.py"}},
		{name: "root and a file", includePaths: []string{".", "README.md"}, want: []string{"README.md", "b.txt", "data_v2/a.csv", "notes.txt", "src/model.py", "src/tests/test_model.py"}},
		{name: "question mark glob", includePaths: []string{"note?.txt"}, want: []string{"notes.txt"}},
		{name: "bracket glob", includePaths: []string{"[ab].txt"}, want: []string{"b.txt"}},
		{name: "exclude magic", includePaths: []string{"src", ":!src/tests"}, want: []string{"src/model.py"}},
		{name: "long exclude magic", includePaths: []string{"src", ":(exclude)src/tests"}, want: []string{"src/model.py"}},
		{name: "prefix of another directory", includePaths: []string{"data_v2", "data"}, wantErr: ": data."},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := t.TempDir()
			for _, f := range []string{"README.md", "b.txt", "data_v2/a.csv", "notes.txt", "src/model.py", "src/tests/test_model.py"} {
				writeRepoFile(t, repo, f, "x")
			}

			files, err := snapshotFiles(t.Context(), repo, tt.includePaths, false)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "include_paths select no files")
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)
			var rels []string
			for _, file := range files {
				rels = append(rels, filepath.ToSlash(file.rel))
			}
			assert.ElementsMatch(t, tt.want, rels)
		})
	}
}

// git lists a nested repository as a single directory entry, which the tarball
// writer skips, so it must not count as a match or as a file.
func TestSnapshotFiles_NestedRepoSelectsNoFiles(t *testing.T) {
	repo := t.TempDir()
	writeRepoFile(t, repo, "train.py", "print(1)\n")
	nested := filepath.Join(repo, "my_lib")
	writeRepoFile(t, nested, "lib.py", "x = 1\n")
	runGit(t, nested, "init", "-q")

	_, err := snapshotFiles(t.Context(), repo, []string{"train.py", "my_lib"}, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), ": my_lib.")

	_, err = snapshotFiles(t.Context(), repo, []string{"my_lib"}, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "contains no files")
}

func TestSnapshotFiles_RejectsPathspecOutsideRoot(t *testing.T) {
	repo := newTestRepo(t)
	writeRepoFile(t, repo, "secret/key.txt", "s")
	writeRepoFile(t, repo, "proj/main.py", "print(1)\n")
	commitAll(t, repo, "init")

	_, err := snapshotFiles(t.Context(), filepath.Join(repo, "proj"), []string{"main.py", ":(top)secret"}, true)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "outside root_path")
}

// git reads a backslash in a pathspec as an escape, so the entry is a pattern and is
// left to git. On Windows a backslash is a path separator instead.
func TestSnapshotFiles_AcceptsBackslashEscapedIncludePath(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("a backslash is a path separator on Windows")
	}
	repo := t.TempDir()
	writeRepoFile(t, repo, "my file.txt", "x")
	writeRepoFile(t, repo, "other.txt", "x")

	files, err := snapshotFiles(t.Context(), repo, []string{`my\ file.txt`}, false)
	require.NoError(t, err)
	require.Len(t, files, 1)
	assert.Equal(t, "my file.txt", filepath.ToSlash(files[0].rel))
}

func TestSnapshotFiles_OmitsIgnoreRuleHintOutsideGit(t *testing.T) {
	repo := t.TempDir()
	writeRepoFile(t, repo, "main.py", "print(1)\n")

	_, err := snapshotFiles(t.Context(), repo, []string{"main.py", "missing"}, false)
	require.Error(t, err)
	assert.NotContains(t, err.Error(), "check-ignore")
}
