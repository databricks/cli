package aircmd

import (
	"context"
	"io"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/databricks-sdk-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveRootPath(t *testing.T) {
	ctx := t.Context()
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "proj"), 0o755))

	// root_path "." resolves against configDir to an absolute path whose basename is
	// the real directory name — not "." (which would name the tarball ._<key>.tar.gz,
	// colliding with the AppleDouble exclude pattern the remote strips).
	got, err := resolveRootPath(ctx, ".", filepath.Join(dir, "proj"))
	require.NoError(t, err)
	assert.True(t, filepath.IsAbs(got))
	assert.Equal(t, "proj", filepath.Base(got))

	// A relative subpath resolves against configDir and keeps its own basename.
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "proj", "sub"), 0o755))
	got, err = resolveRootPath(ctx, "sub", filepath.Join(dir, "proj"))
	require.NoError(t, err)
	assert.Equal(t, "sub", filepath.Base(got))

	// A non-existent path errors.
	_, err = resolveRootPath(ctx, "missing", dir)
	require.Error(t, err)
}

// newSnapshotTestClient returns a workspace client backed by the in-process fake,
// which models workspace get-status / import-file with real state.
func newSnapshotTestClient(t *testing.T) *databricks.WorkspaceClient {
	t.Helper()
	server := testserver.New(t)
	t.Cleanup(server.Close)
	testserver.AddDefaultHandlers(server)
	w, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "token"})
	require.NoError(t, err)
	return w
}

// testUploader builds a snapshotUploader whose tar store and sidecar store both live
// under distinct workspace roots on the fake server.
func testUploader(t *testing.T, w *databricks.WorkspaceClient, tarBase, sidecarBase string) snapshotUploader {
	t.Helper()
	tarStore, err := filer.NewWorkspaceFilesClient(w, tarBase)
	require.NoError(t, err)
	sidecarStore, err := filer.NewWorkspaceFilesClient(w, sidecarBase)
	require.NoError(t, err)
	return snapshotUploader{tarStore: tarStore, sidecarStore: sidecarStore, tarBase: tarBase, sidecarBase: sidecarBase}
}

func TestRunSnapshot_GitArchive(t *testing.T) {
	ctx := t.Context()
	w := newSnapshotTestClient(t)
	repo := newTestRepo(t)
	writeRepoFile(t, repo, "train.py", "print()")
	sha := commitAll(t, repo, "init")

	up := testUploader(t, w, "/Workspace/Users/me/.air/repo_snapshots/repo", "/Workspace/Users/me/.air/cli_launch/exp/run")
	res, err := runSnapshot(ctx, up, repo, &snapshotSourceConfig{RootPath: repo, Git: &gitRef{Commit: &sha}})
	require.NoError(t, err)

	// Tarball is cache-key-named under the tar base, prefixed with the repo dir name
	// (the temp dir's basename); a clean git repo yields a git_state sidecar, no diff.
	cacheKey := computeSnapshotCacheKey(sha, nil)
	wantName := filepath.Base(repo) + "_" + cacheKey[:16] + ".tar.gz"
	assert.Equal(t, path.Join(up.tarBase, wantName), res.CodeSourcePath)
	assert.Equal(t, path.Join(up.sidecarBase, gitStateName), res.GitStatePath)
	assert.Empty(t, res.GitDiffPath)
}

func TestRunSnapshot_CacheHitSkipsUpload(t *testing.T) {
	ctx := t.Context()
	w := newSnapshotTestClient(t)
	repo := newTestRepo(t)
	writeRepoFile(t, repo, "train.py", "print()")
	sha := commitAll(t, repo, "init")

	up := testUploader(t, w, "/Workspace/Users/me/.air/repo_snapshots/repo", "/Workspace/Users/me/.air/cli_launch/exp/run")
	snap := &snapshotSourceConfig{RootPath: repo, Git: &gitRef{Commit: &sha}}

	// First submission uploads the tarball.
	res1, err := runSnapshot(ctx, up, repo, snap)
	require.NoError(t, err)

	// Count uploads to the tarball path on a fresh uploader: the second run should
	// see the cached tarball via Stat and not re-upload it.
	writes := &countingFiler{Filer: up.tarStore}
	up2 := up
	up2.tarStore = writes
	res2, err := runSnapshot(ctx, up2, repo, snap)
	require.NoError(t, err)

	assert.Equal(t, res1.CodeSourcePath, res2.CodeSourcePath)
	assert.Zero(t, writes.writes, "cache hit must not re-upload the tarball")
}

func TestRunSnapshot_PlainTarDirty(t *testing.T) {
	ctx := t.Context()
	w := newSnapshotTestClient(t)
	repo := newTestRepo(t)
	writeRepoFile(t, repo, "train.py", "print()")
	commitAll(t, repo, "init")
	writeRepoFile(t, repo, "train.py", "print('wip')") // dirty, no git ref

	up := testUploader(t, w, "/Workspace/Users/me/.air/repo_snapshots/repo", "/Workspace/Users/me/.air/cli_launch/exp/run")
	res, err := runSnapshot(ctx, up, repo, &snapshotSourceConfig{RootPath: repo})
	require.NoError(t, err)

	// Plain tar is timestamp-named (not cache-key-named); a dirty tree captures both
	// the state and the diff sidecar.
	assert.Contains(t, res.CodeSourcePath, path.Join(up.tarBase, filepath.Base(repo)+"_"))
	assert.Equal(t, path.Join(up.sidecarBase, gitStateName), res.GitStatePath)
	assert.Equal(t, path.Join(up.sidecarBase, gitDiffName), res.GitDiffPath)
}

func TestRunSnapshot_NonGitDir(t *testing.T) {
	ctx := t.Context()
	w := newSnapshotTestClient(t)
	dir := t.TempDir()
	writeRepoFile(t, dir, "train.py", "print()")

	up := testUploader(t, w, "/Workspace/Users/me/.air/repo_snapshots/proj", "/Workspace/Users/me/.air/cli_launch/exp/run")
	res, err := runSnapshot(ctx, up, dir, &snapshotSourceConfig{RootPath: dir})
	require.NoError(t, err)

	// Non-git dir: plain tar, and no provenance sidecars.
	assert.NotEmpty(t, res.CodeSourcePath)
	assert.Empty(t, res.GitStatePath)
	assert.Empty(t, res.GitDiffPath)
}

// countingFiler wraps a Filer to count Write calls, for asserting cache-hit skips.
type countingFiler struct {
	filer.Filer
	writes int
}

func (c *countingFiler) Write(ctx context.Context, name string, reader io.Reader, mode ...filer.WriteMode) error {
	c.writes++
	return c.Filer.Write(ctx, name, reader, mode...)
}
