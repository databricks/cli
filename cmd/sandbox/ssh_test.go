package sandbox

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/databricks/cli/libs/env"
	"github.com/databricks/databricks-sdk-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveSSHName(t *testing.T) {
	ctx := env.WithUserHomeDir(t.Context(), t.TempDir())
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/.well-known/databricks-config" {
			_, _ = io.WriteString(w, `{}`)
			return
		}
		calls++
		if strings.HasSuffix(r.URL.Path, "missing") {
			w.WriteHeader(http.StatusNotFound)
			_, _ = io.WriteString(w, `{"error_code":"NOT_FOUND","message":"missing"}`)
			return
		}
		_, _ = io.WriteString(w, `{"name":"sandboxes/test","status":{"state":"SANDBOX_STATE_RUNNING"}}`)
	}))
	defer server.Close()
	w, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "test-token"})
	require.NoError(t, err)
	require.NoError(t, setDefault(ctx, server.URL, "missing"))
	_, err = ResolveSSHName(ctx, w, "")
	require.ErrorContains(t, err, "unavailable through the Sandbox API")
	assert.Equal(t, "missing", getDefault(ctx, server.URL))
	for _, id := range []string{".", "..", "sandboxes/.", "sandboxes/..", "a/b", `a\b`, "a?b", "a#b", "a b", "a\tb"} {
		_, err = ResolveSSHName(ctx, w, id)
		require.ErrorContains(t, err, "invalid sandbox ID", id)
	}
	name, err := ResolveSSHName(ctx, w, "sandboxes/test")
	require.NoError(t, err)
	assert.Equal(t, "sandboxes/test", name)
	assert.Equal(t, 2, calls)
}
