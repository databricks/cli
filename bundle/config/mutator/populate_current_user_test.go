package mutator

import (
	"net/http"
	"sync/atomic"
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
	var names []string
	for _, g := range b.Config.Workspace.CurrentUser.Groups {
		names = append(names, g.Display)
	}
	// The fake lists the built-in users group for every identity.
	assert.Equal(t, []string{"alpha", "users", "zeta"}, names)
}

// Workspace-local groups are named by the parent-groups response, so admins and users
// resolve like account groups.
func TestPopulateCurrentUserSetsWorkspaceLocalGroups(t *testing.T) {
	b := newPopulateBundle(t, testserver.New(t), "admins")

	diags := bundle.Apply(t.Context(), b, PopulateCurrentUser())
	require.NoError(t, diags.Error())
	var names []string
	for _, g := range b.Config.Workspace.CurrentUser.Groups {
		names = append(names, g.Display)
	}
	assert.Equal(t, []string{"admins", "users"}, names)
}

// A failed parent-groups read leaves the groups empty and only logs, and the current user is still set.
func TestPopulateCurrentUserSkipsGroupsWhenReadFails(t *testing.T) {
	server := testserver.New(t)
	server.Handle("GET", "/api/2.0/identity/me/transitive-parent-groups", func(testserver.Request) any {
		return testserver.Response{StatusCode: http.StatusForbidden, Body: map[string]any{"error_code": "PERMISSION_DENIED", "message": "denied"}}
	})
	b := newPopulateBundle(t, server, "testgroup")

	// The failure is logged, not a diagnostic, so `bundle validate --strict` still passes.
	diags := bundle.Apply(t.Context(), b, PopulateCurrentUser())
	require.NoError(t, diags.Error())
	assert.Empty(t, diags)
	assert.Nil(t, b.Config.Workspace.CurrentUser.Groups)
	assert.True(t, b.CurrentUserGroupsUnknown)
	assert.Equal(t, "tester@databricks.com", b.Config.Workspace.CurrentUser.UserName)
}

// The groups are cached with the principal, so a second command with the same
// fingerprint reads neither identity/me nor the parent groups.
func TestPopulateCurrentUserCachesGroupsWithPrincipal(t *testing.T) {
	server := testserver.New(t)
	var meCalls, parentCalls atomic.Int32
	server.Handle("GET", "/api/2.0/identity/me", func(req testserver.Request) any {
		meCalls.Add(1)
		return req.Workspace.IdentityMe(req.Token)
	})
	server.Handle("GET", "/api/2.0/identity/me/transitive-parent-groups", func(req testserver.Request) any {
		parentCalls.Add(1)
		return req.Workspace.IdentityMeTransitiveParentGroups()
	})
	first := newPopulateBundle(t, server, "alpha")

	diags := bundle.Apply(t.Context(), first, PopulateCurrentUser())
	require.NoError(t, diags.Error())
	assert.Equal(t, int32(1), meCalls.Load())
	assert.Equal(t, int32(1), parentCalls.Load())

	second := &bundle.Bundle{Config: config.Root{}}
	second.Cache = first.Cache
	second.SetWorkpaceClient(first.WorkspaceClient(t.Context()))
	diags = bundle.Apply(t.Context(), second, PopulateCurrentUser())
	require.NoError(t, diags.Error())
	assert.Equal(t, int32(1), meCalls.Load())
	assert.Equal(t, int32(1), parentCalls.Load())
	require.Len(t, second.Config.Workspace.CurrentUser.Groups, 2)
	assert.Equal(t, "alpha", second.Config.Workspace.CurrentUser.Groups[0].Display)
	assert.Equal(t, "users", second.Config.Workspace.CurrentUser.Groups[1].Display)
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

// A workspace without identity/me reads the current user from SCIM /Me, and both source
// metrics say so.
func TestPopulateCurrentUserFallsBackToSCIMWhenIdentityMeNotEnabled(t *testing.T) {
	server := testserver.New(t)
	notEnabled := func(testserver.Request) any {
		return testserver.Response{StatusCode: http.StatusNotFound, Body: map[string]any{"error_code": "ENDPOINT_NOT_FOUND", "message": "Me API is not enabled."}}
	}
	server.Handle("GET", "/api/2.0/identity/me", notEnabled)
	server.Handle("GET", "/api/2.0/identity/me/transitive-parent-groups", notEnabled)
	b := newPopulateBundle(t, server, "testgroup")

	diags := bundle.Apply(t.Context(), b, PopulateCurrentUser())
	require.NoError(t, diags.Error())
	assert.Empty(t, diags)
	assert.Equal(t, "tester@databricks.com", b.Config.Workspace.CurrentUser.UserName)
	assert.False(t, b.CurrentUserGroupsUnknown)
	require.NotNil(t, b.Telemetry.CurrentUserFromSCIMFallback)
	assert.True(t, *b.Telemetry.CurrentUserFromSCIMFallback)
	require.NotNil(t, b.Telemetry.CurrentUserGroupsFromSCIMFallback)
	assert.True(t, *b.Telemetry.CurrentUserGroupsFromSCIMFallback)
}

// A workspace that serves identity/me records that the user did not come from the fallback.
func TestPopulateCurrentUserRecordsIdentityMeAsSource(t *testing.T) {
	b := newPopulateBundle(t, testserver.New(t), "alpha")

	diags := bundle.Apply(t.Context(), b, PopulateCurrentUser())
	require.NoError(t, diags.Error())
	require.NotNil(t, b.Telemetry.CurrentUserFromSCIMFallback)
	assert.False(t, *b.Telemetry.CurrentUserFromSCIMFallback)
	require.NotNil(t, b.Telemetry.CurrentUserGroupsFromSCIMFallback)
	assert.False(t, *b.Telemetry.CurrentUserGroupsFromSCIMFallback)
}

// A cache hit reports the source of the read that filled the cache, so the metrics do not
// depend on whether the command read the current user itself.
func TestPopulateCurrentUserCacheKeepsSource(t *testing.T) {
	server := testserver.New(t)
	var scimCalls atomic.Int32
	notEnabled := func(testserver.Request) any {
		return testserver.Response{StatusCode: http.StatusNotFound, Body: map[string]any{"error_code": "ENDPOINT_NOT_FOUND", "message": "Me API is not enabled."}}
	}
	server.Handle("GET", "/api/2.0/identity/me", notEnabled)
	server.Handle("GET", "/api/2.0/identity/me/transitive-parent-groups", notEnabled)
	server.Handle("GET", "/api/2.0/preview/scim/v2/Me", func(req testserver.Request) any {
		scimCalls.Add(1)
		return req.Workspace.MeUser(req.Token)
	})
	first := newPopulateBundle(t, server, "testgroup")
	require.NoError(t, bundle.Apply(t.Context(), first, PopulateCurrentUser()).Error())

	second := &bundle.Bundle{Config: config.Root{}}
	second.Cache = first.Cache
	second.SetWorkpaceClient(first.WorkspaceClient(t.Context()))
	require.NoError(t, bundle.Apply(t.Context(), second, PopulateCurrentUser()).Error())

	assert.Equal(t, int32(1), scimCalls.Load())
	require.NotNil(t, second.Telemetry.CurrentUserFromSCIMFallback)
	assert.True(t, *second.Telemetry.CurrentUserFromSCIMFallback)
}

// Only a not-enabled identity/me falls back. A forbidden read fails the command, and SCIM is
// not asked for the user in its place.
func TestPopulateCurrentUserDoesNotFallBackOnForbidden(t *testing.T) {
	server := testserver.New(t)
	var scimCalls atomic.Int32
	server.Handle("GET", "/api/2.0/identity/me", func(testserver.Request) any {
		return testserver.Response{StatusCode: http.StatusForbidden, Body: map[string]any{"error_code": "PERMISSION_DENIED", "message": "denied"}}
	})
	server.Handle("GET", "/api/2.0/preview/scim/v2/Me", func(req testserver.Request) any {
		scimCalls.Add(1)
		return req.Workspace.MeUser(req.Token)
	})
	b := newPopulateBundle(t, server, "testgroup")

	diags := bundle.Apply(t.Context(), b, PopulateCurrentUser())
	require.Error(t, diags.Error())
	assert.Nil(t, b.Config.Workspace.CurrentUser)
	assert.Equal(t, int32(0), scimCalls.Load())
}

// A failed groups read does not fail the fallback: the groups come from the SCIM response, and
// the not-enabled groups error is not shown.
func TestPopulateCurrentUserFallbackIgnoresGroupsError(t *testing.T) {
	server := testserver.New(t)
	server.Handle("GET", "/api/2.0/identity/me", func(testserver.Request) any {
		return testserver.Response{StatusCode: http.StatusNotFound, Body: map[string]any{"error_code": "ENDPOINT_NOT_FOUND", "message": "Me API is not enabled."}}
	})
	server.Handle("GET", "/api/2.0/identity/me/transitive-parent-groups", func(testserver.Request) any {
		return testserver.Response{StatusCode: http.StatusNotFound, Body: map[string]any{"error_code": "ENDPOINT_NOT_FOUND", "message": "groups not enabled"}}
	})
	b := newPopulateBundle(t, server)

	diags := bundle.Apply(t.Context(), b, PopulateCurrentUser())
	require.NoError(t, diags.Error())
	assert.False(t, b.CurrentUserGroupsUnknown)
	assert.Equal(t, "tester@databricks.com", b.Config.Workspace.CurrentUser.UserName)
}

// An assumed group is rejected with the same error whether or not its groups read fails,
// and the failed groups read does not turn into the unknown-groups state.
func TestPopulateCurrentUserRejectsGroupPrincipalWhenGroupsReadFails(t *testing.T) {
	server := testserver.New(t)
	server.Handle("GET", "/api/2.0/identity/me", func(testserver.Request) any {
		return map[string]any{"group": map[string]any{"group_id": "3000033333", "group_name": "data-engineers"}}
	})
	server.Handle("GET", "/api/2.0/identity/me/transitive-parent-groups", func(testserver.Request) any {
		return testserver.Response{StatusCode: http.StatusForbidden, Body: map[string]any{"error_code": "PERMISSION_DENIED", "message": "groups denied"}}
	})
	b := newPopulateBundle(t, server)

	diags := bundle.Apply(t.Context(), b, PopulateCurrentUser())
	require.Len(t, diags, 1)
	assert.Equal(t, diag.Error, diags[0].Severity)
	assert.Equal(t, iamutil.ErrGroupPrincipal.Error(), diags[0].Summary)
	assert.False(t, b.CurrentUserGroupsUnknown)
	assert.Nil(t, b.Config.Workspace.CurrentUser)
}
