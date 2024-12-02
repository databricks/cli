package internal

import (
	"os"
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

func assertFullGitInfo(t *testing.T, info git.GitRepositoryInfo, expectedRoot string) {
	assert.Equal(t, "main", info.CurrentBranch)
	assert.NotEmpty(t, info.LatestCommit)
	assert.Equal(t, examplesRepoUrl, info.OriginURL)
	assert.Equal(t, expectedRoot, info.WorktreeRoot)
}

func TestAccFetchRepositoryInfoAPI_FromRepo(t *testing.T) {
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
			assertFullGitInfo(t, info, targetPath)
		})
	}
}

func TestAccFetchRepositoryInfoAPI_FromNonRepo(t *testing.T) {
	ctx, wt := acc.WorkspaceTest(t)
	me, err := wt.W.CurrentUser.Me(ctx)
	require.NoError(t, err)

	rootPath := acc.RandomName("/Workspace/Users/" + me.UserName + "/testing-nonrepo-")
	_, stderr := RequireSuccessfulRun(t, "workspace", "mkdirs", rootPath+"/a/b/c")
	t.Cleanup(func() {
		RequireSuccessfulRun(t, "workspace", "delete", "--recursive", rootPath)
	})

	assert.Empty(t, stderr.String())
	//assert.NotEmpty(t, stdout.String())
	ctx = dbr.MockRuntime(ctx, true)

	tests := []struct {
		input string
		msg   string
	}{
		{
			input: rootPath + "/a/b/c",
			msg:   "",
		},
		{
			input: rootPath,
			msg:   "",
		},
		{
			input: rootPath + "/non-existent",
			msg:   "doesn't exist",
		},
	}

	for _, test := range tests {
		t.Run(test.input+" <==> "+test.msg, func(t *testing.T) {
			info, err := git.FetchRepositoryInfo(ctx, vfs.MustNew(test.input), wt.W)
			if test.msg == "" {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), test.msg)
			}
			assert.Equal(t, "", info.CurrentBranch)
			assert.Equal(t, "", info.LatestCommit)
			assert.Equal(t, "", info.OriginURL)
			assert.Equal(t, "", info.WorktreeRoot)
		})
	}
}

func TestAccFetchRepositoryInfoDotGit_FromGitRepo(t *testing.T) {
	ctx, wt := acc.WorkspaceTest(t)

	repo := cloneRepoLocally(t, examplesRepoUrl)

	for _, inputPath := range []string{
		repo + "/knowledge_base/dashboard_nyc_taxi",
		repo,
	} {
		t.Run(inputPath, func(t *testing.T) {
			info, err := git.FetchRepositoryInfo(ctx, vfs.MustNew(inputPath), wt.W)
			assert.NoError(t, err)
			assertFullGitInfo(t, info, repo)
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

func TestAccFetchRepositoryInfoDotGit_FromNonGitRepo(t *testing.T) {
	ctx, wt := acc.WorkspaceTest(t)

	tempDir := t.TempDir()
	root := filepath.Join(tempDir, "repo")
	require.NoError(t, os.MkdirAll(root+"/a/b/c", 0700))

	tests := []string{
		root + "/a/b/c",
		root,
		root + "/non-existent",
	}

	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			info, err := git.FetchRepositoryInfo(ctx, vfs.MustNew(input), wt.W)
			assert.NoError(t, err)
			assert.Equal(t, "", info.CurrentBranch)
			assert.Equal(t, "", info.LatestCommit)
			assert.Equal(t, "", info.OriginURL)
			assert.Equal(t, "", info.WorktreeRoot)
		})
	}
}

func TestAccFetchRepositoryInfoDotGit_FromBrokenGitRepo(t *testing.T) {
	ctx, wt := acc.WorkspaceTest(t)

	tempDir := t.TempDir()
	root := filepath.Join(tempDir, "repo")
	path := root + "/a/b/c"
	require.NoError(t, os.MkdirAll(path, 0700))
	require.NoError(t, os.WriteFile(root+"/.git", []byte(""), 0000))

	info, err := git.FetchRepositoryInfo(ctx, vfs.MustNew(path), wt.W)
	assert.NoError(t, err)
	assert.Equal(t, root, info.WorktreeRoot)
	assert.Equal(t, "", info.CurrentBranch)
	assert.Equal(t, "", info.LatestCommit)
	assert.Equal(t, "", info.OriginURL)
}
