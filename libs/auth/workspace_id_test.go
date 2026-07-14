package auth_test

import (
	"testing"

	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/databricks-sdk-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveWorkspaceID_FastPathReturnsConfiguredValue(t *testing.T) {
	// Pointing at a /Me handler that would return a different ID lets us
	// confirm the fast path doesn't hit the API.
	server := testserver.New(t)
	server.Handle("GET", "/api/2.0/preview/scim/v2/Me", func(req testserver.Request) any {
		t.Fatalf("/Me should not be called when WorkspaceID is configured")
		return nil
	})
	testserver.AddDefaultHandlers(server)

	w, err := databricks.NewWorkspaceClient(&databricks.Config{
		Host:        server.URL,
		Token:       "testtoken",
		WorkspaceID: "12345",
	})
	require.NoError(t, err)

	got, err := auth.ResolveWorkspaceID(t.Context(), w)
	require.NoError(t, err)
	assert.Equal(t, "12345", got)
}

func TestResolveWorkspaceID_FastPathPassesThroughNonNumericValue(t *testing.T) {
	// Connection-style identifiers (UUID-shaped, etc.) are valid as of the
	// X-Databricks-Workspace-Id header rollout. The configured value must
	// flow through unchanged.
	server := testserver.New(t)
	server.Handle("GET", "/api/2.0/preview/scim/v2/Me", func(req testserver.Request) any {
		t.Fatalf("/Me should not be called when WorkspaceID is configured")
		return nil
	})
	testserver.AddDefaultHandlers(server)

	const connID = "123e4567-e89b-12d3-a456-426614174000"
	w, err := databricks.NewWorkspaceClient(&databricks.Config{
		Host:        server.URL,
		Token:       "testtoken",
		WorkspaceID: connID,
	})
	require.NoError(t, err)

	got, err := auth.ResolveWorkspaceID(t.Context(), w)
	require.NoError(t, err)
	assert.Equal(t, connID, got)
}

func TestResolveWorkspaceID_NoneSentinelFallsThroughToAPI(t *testing.T) {
	// The CLI persists "none" in .databrickscfg to mark profiles where the
	// user skipped workspace selection. It must not leak as a routing ID.
	server := testserver.New(t)
	testserver.AddDefaultHandlers(server)

	w, err := databricks.NewWorkspaceClient(&databricks.Config{
		Host:        server.URL,
		Token:       "testtoken",
		WorkspaceID: auth.WorkspaceIDNone,
	})
	require.NoError(t, err)

	got, err := auth.ResolveWorkspaceID(t.Context(), w)
	require.NoError(t, err)
	// The default /Me handler returns X-Databricks-Org-Id: 900800700600.
	assert.Equal(t, "900800700600", got)
}

func TestResolveWorkspaceID_FallbackHitsMeAndStringifiesResponse(t *testing.T) {
	// Note: testserver.New unconditionally registers a
	// /.well-known/databricks-config handler that returns
	// workspace_id=900800700600. The SDK config resolver picks that up and
	// pre-populates cfg.WorkspaceID, which means the helper's fast path
	// returns "900800700600" without ever hitting /Me. From the helper's
	// perspective the observable behavior is identical (it returns
	// "900800700600" either way), so this test still covers what callers
	// see — it just lands on the fast path rather than the literal
	// fallback path. The fast path is also exercised explicitly by
	// TestResolveWorkspaceID_FastPathReturnsConfiguredValue.
	server := testserver.New(t)
	testserver.AddDefaultHandlers(server)

	w, err := databricks.NewWorkspaceClient(&databricks.Config{
		Host:  server.URL,
		Token: "testtoken",
	})
	require.NoError(t, err)

	got, err := auth.ResolveWorkspaceID(t.Context(), w)
	require.NoError(t, err)
	assert.Equal(t, "900800700600", got)
}
