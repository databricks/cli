package workspace

import (
	"testing"

	"github.com/databricks/cli/libs/iamutil"
	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/databricks-sdk-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetWorkspaceHomeDirReturnsUserFolder(t *testing.T) {
	server := testserver.New(t)
	testserver.AddDefaultHandlers(server)
	w, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "test-token"})
	require.NoError(t, err)

	home, err := GetWorkspaceHomeDir(t.Context(), w)
	require.NoError(t, err)
	assert.Equal(t, "/Workspace/Users/tester@databricks.com", home)
}

// An assumed group has no /Workspace/Users folder, so the home lookup fails with the
// same error as the bundle commands.
func TestGetWorkspaceHomeDirRejectsAssumedGroup(t *testing.T) {
	server := testserver.New(t)
	server.Handle("GET", "/api/2.0/identity/me", func(testserver.Request) any {
		return map[string]any{"group": map[string]any{"group_id": "3000033333", "group_name": "data-engineers"}}
	})
	testserver.AddDefaultHandlers(server)
	w, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "test-token"})
	require.NoError(t, err)

	home, err := GetWorkspaceHomeDir(t.Context(), w)
	require.ErrorIs(t, err, iamutil.ErrGroupPrincipal)
	assert.Empty(t, home)
}
