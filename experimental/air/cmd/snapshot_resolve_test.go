package aircmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveSnapshotPlan_Commit(t *testing.T) {
	ctx := t.Context()
	repo := newTestRepo(t)
	writeRepoFile(t, repo, "a.txt", "1")
	sha := commitAll(t, repo, "init")

	plan, err := resolveSnapshotPlan(ctx, newGitRepo(repo), &gitRef{Commit: new(sha)}, nil)
	require.NoError(t, err)
	assert.Equal(t, modeGitArchive, plan.mode)
	assert.Equal(t, sha, plan.commitSHA)
	assert.True(t, plan.isGitRepo)

	// A commit pin is valid even with a dirty tree: local changes are irrelevant.
	writeRepoFile(t, repo, "a.txt", "2")
	plan, err = resolveSnapshotPlan(ctx, newGitRepo(repo), &gitRef{Commit: new(sha)}, nil)
	require.NoError(t, err)
	assert.Equal(t, modeGitArchive, plan.mode)
	assert.True(t, plan.hasUncommit)
}

func TestResolveSnapshotPlan_CommitNotLocal(t *testing.T) {
	ctx := t.Context()
	repo := newTestRepo(t)
	writeRepoFile(t, repo, "a.txt", "1")
	commitAll(t, repo, "init")

	absent := "deadbeefdeadbeefdeadbeefdeadbeefdeadbeef"
	_, err := resolveSnapshotPlan(ctx, newGitRepo(repo), &gitRef{Commit: new(absent)}, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not exist locally")
}

func TestResolveSnapshotPlan_BranchLocalHead(t *testing.T) {
	ctx := t.Context()
	repo := newTestRepo(t)
	writeRepoFile(t, repo, "a.txt", "1")
	mainSHA := commitAll(t, repo, "init")

	plan, err := resolveSnapshotPlan(ctx, newGitRepo(repo), &gitRef{Branch: new("main")}, nil)
	require.NoError(t, err)
	assert.Equal(t, modeGitArchive, plan.mode)
	assert.Equal(t, mainSHA, plan.commitSHA)
}

func TestResolveSnapshotPlan_BranchDirtyIsError(t *testing.T) {
	ctx := t.Context()
	repo := newTestRepo(t)
	writeRepoFile(t, repo, "a.txt", "1")
	commitAll(t, repo, "init")
	writeRepoFile(t, repo, "a.txt", "2") // uncommitted

	_, err := resolveSnapshotPlan(ctx, newGitRepo(repo), &gitRef{Branch: new("main")}, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "uncommitted changes")
	assert.Contains(t, err.Error(), "git.commit")
}

func TestResolveSnapshotPlan_NoRefPlainTar(t *testing.T) {
	ctx := t.Context()
	repo := newTestRepo(t)
	writeRepoFile(t, repo, "a.txt", "1")
	commitAll(t, repo, "init")
	writeRepoFile(t, repo, "a.txt", "2") // dirty tree is fine for plain tar

	plan, err := resolveSnapshotPlan(ctx, newGitRepo(repo), nil, nil)
	require.NoError(t, err)
	assert.Equal(t, modePlainTar, plan.mode)
	assert.Empty(t, plan.commitSHA)
	assert.True(t, plan.isGitRepo)
	assert.True(t, plan.hasUncommit)
}

func TestResolveSnapshotPlan_NonGitDir(t *testing.T) {
	ctx := t.Context()
	dir := t.TempDir()

	plan, err := resolveSnapshotPlan(ctx, newGitRepo(dir), nil, nil)
	require.NoError(t, err)
	assert.Equal(t, modePlainTar, plan.mode)
	assert.False(t, plan.isGitRepo)

	// A git ref on a non-git directory is an error.
	_, err = resolveSnapshotPlan(ctx, newGitRepo(dir), &gitRef{Branch: new("main")}, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not a git repository")
}

func TestResolveSnapshotPlan_IncludePaths(t *testing.T) {
	ctx := t.Context()
	repo := newTestRepo(t)
	writeRepoFile(t, repo, "src/model.py", "1")
	writeRepoFile(t, repo, "configs/train.yaml", "x")
	sha := commitAll(t, repo, "init")

	// All include paths exist at the commit.
	plan, err := resolveSnapshotPlan(ctx, newGitRepo(repo), &gitRef{Commit: new(sha)}, []string{"src", "configs"})
	require.NoError(t, err)
	assert.Equal(t, modeGitArchive, plan.mode)
	assert.Equal(t, []string{"src", "configs"}, plan.includePaths)

	// A missing include path fails fast.
	_, err = resolveSnapshotPlan(ctx, newGitRepo(repo), &gitRef{Commit: new(sha)}, []string{"src", "missing"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing")
}
