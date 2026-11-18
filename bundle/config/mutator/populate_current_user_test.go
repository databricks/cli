package mutator

import (
	"net/http"
	"testing"
	"time"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/cache"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/iamutil"
	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/databricks-sdk-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newPopulateBundle returns a bundle whose workspace serves the default identity and
// membership in the groups named by groups. The handlers registered on server before
// this call take precedence, so a test can override one endpoint.
func newPopulateBundle(t *testing.T, server *testserver.Server, groups ...string) *bundle.Bundle {
	testserver.AddDefaultHandlers(server)
	server.AddGroups("test-token", groups, nil)
	w, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "test-token"})
	require.NoError(t, err)

	b := &bundle.Bundle{Config: config.Root{}}
	b.Cache = cache.NewCache(t.Context(), "user", time.Minute, &b.Metrics)
	b.SetWorkpaceClient(w)
	return b
}

func TestPopulateCurrentUserSetsGroups(t *testing.T) {
	b := newPopulateBundle(t, testserver.New(t), "zeta", "alpha")

	diags := bundle.Apply(t.Context(), b, PopulateCurrentUser())
	require.NoError(t, diags.Error())
	require.Len(t, b.Config.Workspace.CurrentUser.Groups, 2)
	assert.Equal(t, "alpha", b.Config.Workspace.CurrentUser.Groups[0].Display)
	assert.Equal(t, "zeta", b.Config.Workspace.CurrentUser.Groups[1].Display)
}

// Workspace-local groups are named by the parent-groups response, so admins and users
// resolve like account groups.
func TestPopulateCurrentUserSetsWorkspaceLocalGroups(t *testing.T) {
	b := newPopulateBundle(t, testserver.New(t), "admins", "users")

	diags := bundle.Apply(t.Context(), b, PopulateCurrentUser())
	require.NoError(t, diags.Error())
	var names []string
	for _, g := range b.Config.Workspace.CurrentUser.Groups {
		names = append(names, g.Display)
	}
	assert.Equal(t, []string{"admins", "users"}, names)
}

// A failed parent-groups read leaves the groups empty and warns, and the current user is still set.
func TestPopulateCurrentUserWarnsWhenGroupsFail(t *testing.T) {
	server := testserver.New(t)
	server.Handle("GET", "/api/2.0/identity/me/transitive-parent-groups", func(testserver.Request) any {
		return testserver.Response{StatusCode: http.StatusForbidden, Body: map[string]any{"error_code": "PERMISSION_DENIED", "message": "denied"}}
	})
	b := newPopulateBundle(t, server, "testgroup")

	diags := bundle.Apply(t.Context(), b, PopulateCurrentUser())
	require.Len(t, diags, 1)
	assert.Equal(t, diag.Warning, diags[0].Severity)
	assert.Contains(t, diags[0].Summary, "Could not read the groups of the current user")
	assert.Nil(t, b.Config.Workspace.CurrentUser.Groups)
	assert.Equal(t, "tester@databricks.com", b.Config.Workspace.CurrentUser.UserName)
}

// An assumed group is not a user: bundle commands that deploy as the current user fail
// early with a clear error rather than using the group name as a user name.
func TestPopulateCurrentUserRejectsGroupPrincipal(t *testing.T) {
	server := testserver.New(t)
	server.Handle("GET", "/api/2.0/identity/me", func(testserver.Request) any {
		return map[string]any{"group": map[string]any{"group_id": "3000033333", "group_name": "data-engineers"}}
	})
	b := newPopulateBundle(t, server)

	diags := bundle.Apply(t.Context(), b, PopulateCurrentUser())
	require.Len(t, diags, 1)
	assert.Equal(t, diag.Error, diags[0].Severity)
	assert.Equal(t, iamutil.ErrGroupPrincipal.Error(), diags[0].Summary)
	assert.Nil(t, b.Config.Workspace.CurrentUser)
}
