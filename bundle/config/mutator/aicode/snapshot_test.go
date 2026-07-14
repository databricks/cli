package aicode

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// initGitRepo initializes a deterministic git repo at dir, commits everything in
// it, and returns the HEAD commit SHA. Author/committer dates are pinned so the
// resulting SHA (and thus any git-archive cache key) is stable across runs.
func initGitRepo(t *testing.T, dir string) string {
	t.Helper()
	run := func(env []string, args ...string) string {
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		cmd.Env = append(os.Environ(), env...)
		out, err := cmd.CombinedOutput()
		require.NoErrorf(t, err, "git %v: %s", args, out)
		return string(out)
	}
	dates := []string{
		"GIT_AUTHOR_DATE=2020-01-01T00:00:00Z",
		"GIT_COMMITTER_DATE=2020-01-01T00:00:00Z",
		"GIT_AUTHOR_NAME=Tester", "GIT_AUTHOR_EMAIL=tester@databricks.com",
		"GIT_COMMITTER_NAME=Tester", "GIT_COMMITTER_EMAIL=tester@databricks.com",
	}
	run(nil, "init", "-qb", "main")
	run(nil, "config", "core.hooksPath", "no-hooks")
	run(nil, "add", "-A")
	run(dates, "commit", "-qm", "init")

	cmd := exec.Command("git", "-C", dir, "rev-parse", "HEAD")
	out, err := cmd.Output()
	require.NoError(t, err)
	return string(out[:40])
}

func TestResolveSnapshotPlanCleanGitRepoUsesGitArchive(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "a.py"), []byte("x"), 0o644))
	commit := initGitRepo(t, dir)

	plan, err := resolveSnapshotPlan(t.Context(), newGitRepo(dir))
	require.NoError(t, err)
	require.Equal(t, modeGitArchive, plan.mode)
	require.Equal(t, commit, plan.commitSHA)
}

func TestResolveSnapshotPlanDirtyGitRepoUsesPlainTar(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "a.py"), []byte("x"), 0o644))
	initGitRepo(t, dir)
	// Introduce an uncommitted change.
	require.NoError(t, os.WriteFile(filepath.Join(dir, "a.py"), []byte("changed"), 0o644))

	plan, err := resolveSnapshotPlan(t.Context(), newGitRepo(dir))
	require.NoError(t, err)
	require.Equal(t, modePlainTar, plan.mode)
	require.Empty(t, plan.commitSHA)
}

func TestResolveSnapshotPlanNonGitUsesPlainTar(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "a.py"), []byte("x"), 0o644))

	plan, err := resolveSnapshotPlan(t.Context(), newGitRepo(dir))
	require.NoError(t, err)
	require.Equal(t, modePlainTar, plan.mode)
}
