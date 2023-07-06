package internal

import (
	"context"
	"fmt"
	"strconv"
	"testing"

	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func synthesizeTemporaryRepoPath(t *testing.T, w *databricks.WorkspaceClient, ctx context.Context) string {
	me, err := w.CurrentUser.Me(ctx)
	require.NoError(t, err)
	repoPath := fmt.Sprintf("/Repos/%s/%s", me.UserName, RandomName("empty-repo-integration-"))

	// Cleanup if repo was created at specified path.
	t.Cleanup(func() {
		oi, err := w.Workspace.GetStatusByPath(ctx, repoPath)
		if apierr.IsMissing(err) {
			return
		}
		require.NoError(t, err)
		err = w.Repos.DeleteByRepoId(ctx, oi.ObjectId)
		require.NoError(t, err)
	})

	return repoPath
}

func createTemporaryRepo(t *testing.T, w *databricks.WorkspaceClient, ctx context.Context) (int64, string) {
	repoPath := synthesizeTemporaryRepoPath(t, w, ctx)
	repoInfo, err := w.Repos.Create(ctx, workspace.CreateRepo{
		Path:     repoPath,
		Url:      repoUrl,
		Provider: "gitHub",
	})
	require.NoError(t, err)
	return repoInfo.Id, repoPath
}

func TestReposCreateWithProvider(t *testing.T) {
	t.Log(GetEnvOrSkipTest(t, "CLOUD_ENV"))

	ctx := context.Background()
	w, err := databricks.NewWorkspaceClient()
	require.NoError(t, err)
	repoPath := synthesizeTemporaryRepoPath(t, w, ctx)

	_, stderr := RequireSuccessfulRun(t, "repos", "create", repoUrl, "gitHub", "--path", repoPath)
	assert.Equal(t, "", stderr.String())

	// Confirm the repo was created.
	oi, err := w.Workspace.GetStatusByPath(ctx, repoPath)
	assert.NoError(t, err)
	assert.Equal(t, workspace.ObjectTypeRepo, oi.ObjectType)
}

func TestReposCreateWithoutProvider(t *testing.T) {
	t.Log(GetEnvOrSkipTest(t, "CLOUD_ENV"))

	ctx := context.Background()
	w, err := databricks.NewWorkspaceClient()
	require.NoError(t, err)
	repoPath := synthesizeTemporaryRepoPath(t, w, ctx)

	_, stderr := RequireSuccessfulRun(t, "repos", "create", repoUrl, "--path", repoPath)
	assert.Equal(t, "", stderr.String())

	// Confirm the repo was created.
	oi, err := w.Workspace.GetStatusByPath(ctx, repoPath)
	assert.NoError(t, err)
	assert.Equal(t, workspace.ObjectTypeRepo, oi.ObjectType)
}

func TestReposGet(t *testing.T) {
	t.Log(GetEnvOrSkipTest(t, "CLOUD_ENV"))

	ctx := context.Background()
	w, err := databricks.NewWorkspaceClient()
	require.NoError(t, err)

	repoId, repoPath := createTemporaryRepo(t, w, ctx)

	// Get by ID
	byIdOutput, stderr := RequireSuccessfulRun(t, "repos", "get", strconv.FormatInt(repoId, 10), "--output=json")
	assert.Equal(t, "", stderr.String())

	// Get by path
	byPathOutput, stderr := RequireSuccessfulRun(t, "repos", "get", repoPath, "--output=json")
	assert.Equal(t, "", stderr.String())

	// Output should be the same
	assert.Equal(t, byIdOutput.String(), byPathOutput.String())

	// Get by path fails
	_, stderr, err = RequireErrorRun(t, "repos", "get", repoPath+"-doesntexist", "--output=json")
	assert.ErrorContains(t, err, "failed to look up repo")

	// Get by path resolves to something other than a repo
	_, stderr, err = RequireErrorRun(t, "repos", "get", "/Repos", "--output=json")
	assert.ErrorContains(t, err, "is not a repo")
}

func TestReposUpdate(t *testing.T) {
	t.Log(GetEnvOrSkipTest(t, "CLOUD_ENV"))

	ctx := context.Background()
	w, err := databricks.NewWorkspaceClient()
	require.NoError(t, err)

	repoId, repoPath := createTemporaryRepo(t, w, ctx)

	// Update by ID
	byIdOutput, stderr := RequireSuccessfulRun(t, "repos", "update", strconv.FormatInt(repoId, 10), "--branch", "ide")
	assert.Equal(t, "", stderr.String())

	// Update by path
	byPathOutput, stderr := RequireSuccessfulRun(t, "repos", "update", repoPath, "--branch", "ide")
	assert.Equal(t, "", stderr.String())

	// Output should be the same
	assert.Equal(t, byIdOutput.String(), byPathOutput.String())
}

func TestReposDeleteByID(t *testing.T) {
	t.Log(GetEnvOrSkipTest(t, "CLOUD_ENV"))

	ctx := context.Background()
	w, err := databricks.NewWorkspaceClient()
	require.NoError(t, err)

	repoId, _ := createTemporaryRepo(t, w, ctx)

	// Delete by ID
	stdout, stderr := RequireSuccessfulRun(t, "repos", "delete", strconv.FormatInt(repoId, 10))
	assert.Equal(t, "", stdout.String())
	assert.Equal(t, "", stderr.String())

	// Check it was actually deleted
	_, err = w.Repos.GetByRepoId(ctx, repoId)
	assert.True(t, apierr.IsMissing(err), err)
}

func TestReposDeleteByPath(t *testing.T) {
	t.Log(GetEnvOrSkipTest(t, "CLOUD_ENV"))

	ctx := context.Background()
	w, err := databricks.NewWorkspaceClient()
	require.NoError(t, err)

	repoId, repoPath := createTemporaryRepo(t, w, ctx)

	// Delete by path
	stdout, stderr := RequireSuccessfulRun(t, "repos", "delete", repoPath)
	assert.Equal(t, "", stdout.String())
	assert.Equal(t, "", stderr.String())

	// Check it was actually deleted
	_, err = w.Repos.GetByRepoId(ctx, repoId)
	assert.True(t, apierr.IsMissing(err), err)
}
