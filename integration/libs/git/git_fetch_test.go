package git_test

import (
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/databricks/cli/integration/internal/acc"
	"github.com/databricks/cli/libs/dbr"
	"github.com/databricks/cli/libs/git"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	examplesRepoUrl      = "https://github.com/databricks/bundle-examples"
	examplesRepoProvider = "gitHub"
)

func assertFullGitInfo(t *testing.T, expectedRoot string, info git.RepositoryInfo) {
	assert.Equal(t, "main", info.CurrentBranch)
	assert.NotEmpty(t, info.LatestCommit)
	assert.Equal(t, examplesRepoUrl, info.OriginURL)
	assert.Equal(t, expectedRoot, info.WorktreeRoot)
}

func assertEmptyGitInfo(t *testing.T, info git.RepositoryInfo) {
	assertSparseGitInfo(t, "", info)
}

func assertSparseGitInfo(t *testing.T, expectedRoot string, info git.RepositoryInfo) {
	assert.Equal(t, "", info.CurrentBranch)
	assert.Equal(t, "", info.LatestCommit)
	assert.Equal(t, "", info.OriginURL)
	assert.Equal(t, expectedRoot, info.WorktreeRoot)
}

func ensureWorkspacePrefix(root string) string {
	// The fixture helper doesn't include /Workspace, so include it here.
	if !strings.HasPrefix(root, "/Workspace/") {
		return path.Join("/Workspace", root)
	}
	return root
}

func TestFetchRepositoryInfoAPI_FromRepo(t *testing.T) {
	ctx, wt := acc.WorkspaceTest(t)
	targetPath := ensureWorkspacePrefix(acc.TemporaryRepo(wt, examplesRepoUrl))

	ctx = dbr.MockRuntime(ctx, dbr.Environment{IsDbr: true, Version: "15.4"})

	for _, inputPath := range []string{
		path.Join(targetPath, "knowledge_base/dashboard_nyc_taxi"),
		targetPath,
	} {
		t.Run(inputPath, func(t *testing.T) {
			info, err := git.FetchRepositoryInfo(ctx, inputPath, wt.W)
			assert.NoError(t, err)
			assertFullGitInfo(t, targetPath, info)
		})
	}
}

func TestFetchRepositoryInfoAPI_FromNonRepo(t *testing.T) {
	ctx, wt := acc.WorkspaceTest(t)
	rootPath := ensureWorkspacePrefix(acc.TemporaryWorkspaceDir(wt, "testing-nonrepo-"))

	// Create directory inside this root path (this is cleaned up as part of the root path).
	err := wt.W.Workspace.MkdirsByPath(ctx, path.Join(rootPath, "a/b/c"))
	require.NoError(t, err)

	ctx = dbr.MockRuntime(ctx, dbr.Environment{IsDbr: true, Version: "15.4"})

	tests := []struct {
		input string
		msg   string
	}{
		{
			input: path.Join(rootPath, "a/b/c"),
			msg:   "",
		},
		{
			input: rootPath,
			msg:   "",
		},
		{
			input: path.Join(rootPath, "/non-existent"),
			msg:   "doesn't exist",
		},
	}

	for _, test := range tests {
		t.Run(test.input+" <==> "+test.msg, func(t *testing.T) {
			info, err := git.FetchRepositoryInfo(ctx, test.input, wt.W)
			if test.msg == "" {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.ErrorContains(t, err, test.msg)
			}
			assertEmptyGitInfo(t, info)
		})
	}
}

func TestFetchRepositoryInfoDotGit_FromGitRepo(t *testing.T) {
	ctx, wt := acc.WorkspaceTest(t)

	repo := cloneRepoLocally(t, examplesRepoUrl)

	for _, inputPath := range []string{
		filepath.Join(repo, "knowledge_base/dashboard_nyc_taxi"),
		repo,
	} {
		t.Run(inputPath, func(t *testing.T) {
			info, err := git.FetchRepositoryInfo(ctx, inputPath, wt.W)
			assert.NoError(t, err)
			assertFullGitInfo(t, repo, info)
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

func TestFetchRepositoryInfoDotGit_FromNonGitRepo(t *testing.T) {
	ctx, wt := acc.WorkspaceTest(t)

	tempDir := t.TempDir()
	root := filepath.Join(tempDir, "repo")
	require.NoError(t, os.MkdirAll(filepath.Join(root, "a/b/c"), 0o700))

	tests := []string{
		filepath.Join(root, "a/b/c"),
		root,
		filepath.Join(root, "/non-existent"),
	}

	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			info, err := git.FetchRepositoryInfo(ctx, input, wt.W)
			assert.ErrorIs(t, err, os.ErrNotExist)
			assertEmptyGitInfo(t, info)
		})
	}
}

func TestFetchRepositoryInfoDotGit_FromBrokenGitRepo(t *testing.T) {
	ctx, wt := acc.WorkspaceTest(t)

	tempDir := t.TempDir()
	root := filepath.Join(tempDir, "repo")
	path := filepath.Join(root, "a/b/c")
	require.NoError(t, os.MkdirAll(path, 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(root, ".git"), []byte(""), 0o000))

	info, err := git.FetchRepositoryInfo(ctx, path, wt.W)
	assert.NoError(t, err)
	assertSparseGitInfo(t, root, info)
}
