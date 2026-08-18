package git

import (
	"net/http"
	"testing"

	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Resolving metadata is covered end to end by
// acceptance/bundle/debug/fetch-repository-info, which runs against both a fake
// and a real workspace, and a missing path by TestFetchRepositoryInfoAPI_FromNonRepo.
// Only the failure of the Repos read is left, which neither can provoke.
func TestFetchRepositoryInfoAPI_ReposGetFailure(t *testing.T) {
	const folderPath = "/Workspace/Users/me@example.com/gitfolder"

	server := testserver.New(t)
	// Registered before the default handlers: the first registration for a route
	// wins, so this replaces the Repos API read.
	server.Handle("GET", "/api/2.0/repos/{repo_id}", func(req testserver.Request) any {
		return testserver.Response{
			StatusCode: http.StatusForbidden,
			Body:       map[string]string{"message": "no access"},
		}
	})
	testserver.AddDefaultHandlers(server)

	w, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "testtoken"})
	require.NoError(t, err)

	_, err = w.Repos.Create(t.Context(), workspace.CreateRepoRequest{
		Url:      "https://github.com/databricks/cli",
		Provider: "gitHub",
		Path:     folderPath,
	})
	require.NoError(t, err)

	// The worktree root is still reported; the metadata is left empty rather than
	// failing the call.
	info, err := FetchRepositoryInfoAPI(t.Context(), folderPath, w)
	require.NoError(t, err)

	assert.Equal(t, folderPath, info.WorktreeRoot)
	assert.Empty(t, info.CurrentBranch)
	assert.Empty(t, info.LatestCommit)
	assert.Empty(t, info.OriginURL)
}
