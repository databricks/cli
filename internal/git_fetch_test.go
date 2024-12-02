package internal

import (
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/internal/acc"
	"github.com/databricks/cli/libs/dbr"
	"github.com/databricks/cli/libs/git"
	"github.com/databricks/cli/libs/vfs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const examplesRepoUrl = "https://github.com/databricks/bundle-examples"
const examplesRepoProvider = "gitHub"

func assertGitInfo(t *testing.T, info git.GitRepositoryInfo, expectedRoot string) {
	assert.Equal(t, "main", info.CurrentBranch)
	assert.NotEmpty(t, info.LatestCommit)
	assert.Equal(t, examplesRepoUrl, info.OriginURL)
	assert.Equal(t, expectedRoot, info.WorktreeRoot.Native())
}

func TestFetchRepositoryInfoAPI(t *testing.T) {
	ctx, wt := acc.WorkspaceTest(t)
	me, err := wt.W.CurrentUser.Me(ctx)
	require.NoError(t, err)

	targetPath := acc.RandomName("/Workspace/Users/" + me.UserName + "/testing-clone-bundle-examples-")
	stdout, stderr := RequireSuccessfulRun(t, "repos", "create", examplesRepoUrl, examplesRepoProvider, "--path", targetPath)
	t.Cleanup(func() {
		RequireSuccessfulRun(t, "repos", "delete", targetPath)
	})

	assert.Empty(t, stderr.String())
	assert.NotEmpty(t, stdout.String())
	ctx = dbr.MockRuntime(ctx, true)

	for _, inputPath := range []string{
		targetPath + "/knowledge_base/dashboard_nyc_taxi",
		targetPath,
	} {
		t.Run(inputPath, func(t *testing.T) {
			info, err := git.FetchRepositoryInfo(ctx, vfs.MustNew(inputPath), wt.W)
			assert.NoError(t, err)
			assertGitInfo(t, info, targetPath)
		})
	}
}

func TestFetchRepositoryInfoDotGit(t *testing.T) {
	ctx, wt := acc.WorkspaceTest(t)

	repo := cloneRepoLocally(t, examplesRepoUrl)

	for _, inputPath := range []string{
		repo + "/knowledge_base/dashboard_nyc_taxi",
		repo,
	} {
		t.Run(inputPath, func(t *testing.T) {
			info, err := git.FetchRepositoryInfo(ctx, vfs.MustNew(inputPath), wt.W)
			assert.NoError(t, err)
			assertGitInfo(t, info, repo)
		})
	}
}

func cloneRepoLocally(t *testing.T, repoUrl string) string {
	tempDir := t.TempDir()
	localRoot := filepath.Join(tempDir, "repo")

	cmd := exec.Command("git", "clone", "--depth=1", examplesRepoUrl, localRoot)
	err := cmd.Run()
	require.NoError(t, err)
	return localRoot
}
