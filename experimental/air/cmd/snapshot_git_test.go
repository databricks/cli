package aircmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestRepo initializes a git repo in a temp dir with a deterministic identity
// and returns its path. Tests build up real commits/branches/dirty states on top,
// mirroring the Python git_state tests (which drive real repos, not a fake).
func newTestRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	runGit(t, dir, "init", "-q", "-b", "main")
	// Deterministic identity so commits succeed in a bare CI environment.
	runGit(t, dir, "config", "user.email", "test@example.test")
	runGit(t, dir, "config", "user.name", "Test")
	return dir
}

// runGit runs a git command in dir and fails the test on error.
func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "git %v: %s", args, out)
}

// writeRepoFile writes a file at a repo-relative path, creating parent dirs.
func writeRepoFile(t *testing.T, repo, rel, content string) {
	t.Helper()
	full := filepath.Join(repo, rel)
	require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o755))
	require.NoError(t, os.WriteFile(full, []byte(content), 0o600))
}

// commitAll stages everything and commits, returning the new HEAD SHA.
func commitAll(t *testing.T, repo, msg string) string {
	t.Helper()
	runGit(t, repo, "add", "-A")
	runGit(t, repo, "commit", "-q", "-m", msg)
	sha, err := newGitRepo(repo).headSHA(t.Context())
	require.NoError(t, err)
	return sha
}

func TestGitRepo_IsRepository(t *testing.T) {
	ctx := t.Context()

	repo := newTestRepo(t)
	assert.True(t, newGitRepo(repo).isRepository(ctx))

	// A subdirectory of a repo is still inside the work tree.
	writeRepoFile(t, repo, "sub/x.txt", "hi")
	assert.True(t, newGitRepo(filepath.Join(repo, "sub")).isRepository(ctx))

	// A plain temp dir with no repo is not.
	assert.False(t, newGitRepo(t.TempDir()).isRepository(ctx))
}

func TestGitRepo_HeadSHA(t *testing.T) {
	ctx := t.Context()
	repo := newTestRepo(t)
	writeRepoFile(t, repo, "a.txt", "1")
	sha := commitAll(t, repo, "init")

	got, err := newGitRepo(repo).headSHA(ctx)
	require.NoError(t, err)
	assert.Equal(t, sha, got)
	assert.Len(t, got, 40)
}

func TestGitRepo_HasUncommittedChanges(t *testing.T) {
	ctx := t.Context()
	repo := newTestRepo(t)
	writeRepoFile(t, repo, "a.txt", "1")
	commitAll(t, repo, "init")

	// Clean tree.
	dirty, err := newGitRepo(repo).hasUncommittedChanges(ctx)
	require.NoError(t, err)
	assert.False(t, dirty)

	// Unstaged modification.
	writeRepoFile(t, repo, "a.txt", "2")
	dirty, err = newGitRepo(repo).hasUncommittedChanges(ctx)
	require.NoError(t, err)
	assert.True(t, dirty)
}

func TestGitRepo_HasUncommittedChangesInPaths(t *testing.T) {
	ctx := t.Context()

	repo := newTestRepo(t)
	writeRepoFile(t, repo, "src/model.py", "1")
	writeRepoFile(t, repo, "other/x.py", "1")
	commitAll(t, repo, "init")
	g := newGitRepo(repo)

	// No paths: no changes, and git is never consulted.
	dirty, err := g.hasUncommittedChangesInPaths(ctx, nil)
	require.NoError(t, err)
	assert.False(t, dirty)

	// A change outside the included paths is ignored.
	writeRepoFile(t, repo, "other/x.py", "2")
	dirty, err = g.hasUncommittedChangesInPaths(ctx, []string{"src", "configs"})
	require.NoError(t, err)
	assert.False(t, dirty)

	// A change inside an included path is reported.
	writeRepoFile(t, repo, "src/model.py", "2")
	dirty, err = g.hasUncommittedChangesInPaths(ctx, []string{"src", "configs"})
	require.NoError(t, err)
	assert.True(t, dirty)

	// Trailing slashes on include paths are trimmed for the pathspec.
	dirty, err = g.hasUncommittedChangesInPaths(ctx, []string{"other/"})
	require.NoError(t, err)
	assert.True(t, dirty)
}

func TestGitRepo_HasUncommittedChangesInPaths_Rename(t *testing.T) {
	ctx := t.Context()
	repo := newTestRepo(t)
	writeRepoFile(t, repo, "src/old.py", "content")
	commitAll(t, repo, "init")

	// A rename within an included path counts as a change, however git classifies
	// it (rename vs delete+add); we only assert the boolean.
	runGit(t, repo, "mv", "src/old.py", "src/new.py")
	dirty, err := newGitRepo(repo).hasUncommittedChangesInPaths(ctx, []string{"src"})
	require.NoError(t, err)
	assert.True(t, dirty)
}

func TestGitRepo_ResolveLocalBranchSHA(t *testing.T) {
	ctx := t.Context()
	repo := newTestRepo(t)
	writeRepoFile(t, repo, "a.txt", "1")
	mainSHA := commitAll(t, repo, "init")

	// A second branch at its own commit.
	runGit(t, repo, "checkout", "-q", "-b", "feature")
	writeRepoFile(t, repo, "b.txt", "2")
	featSHA := commitAll(t, repo, "feature work")
	g := newGitRepo(repo)

	got, err := g.resolveLocalBranchSHA(ctx, "main")
	require.NoError(t, err)
	assert.Equal(t, mainSHA, got)

	got, err = g.resolveLocalBranchSHA(ctx, "feature")
	require.NoError(t, err)
	assert.Equal(t, featSHA, got)

	// A branch that does not exist locally errors (no remote is contacted).
	_, err = g.resolveLocalBranchSHA(ctx, "nope")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "resolve local branch")
}

func TestGitRepo_CommitExistsLocally(t *testing.T) {
	ctx := t.Context()
	repo := newTestRepo(t)
	writeRepoFile(t, repo, "a.txt", "1")
	sha := commitAll(t, repo, "init")
	g := newGitRepo(repo)

	assert.True(t, g.commitExistsLocally(ctx, sha))
	assert.False(t, g.commitExistsLocally(ctx, "deadbeefdeadbeefdeadbeefdeadbeefdeadbeef"))
}

func TestGitRepo_ValidateIncludePathsExist(t *testing.T) {
	ctx := t.Context()
	repo := newTestRepo(t)
	writeRepoFile(t, repo, "src/model.py", "1")
	writeRepoFile(t, repo, "configs/train.yaml", "x")
	writeRepoFile(t, repo, "train.py", "print()")
	sha := commitAll(t, repo, "init")
	g := newGitRepo(repo)

	// Both directory and file include_paths are accepted (ls-tree without -d).
	require.NoError(t, g.validateIncludePathsExist(ctx, sha, []string{"src", "configs", "train.py"}))

	err := g.validateIncludePathsExist(ctx, sha, []string{"src", "missing"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing")
	assert.Contains(t, err.Error(), sha[:8])
}
