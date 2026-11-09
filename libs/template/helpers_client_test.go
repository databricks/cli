package template

import (
	"strconv"
	"sync"
	"testing"

	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/databricks-sdk-go"
	sdkconfig "github.com/databricks/databricks-sdk-go/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// recordingClient returns a client for a new test server and a function that returns the
// requests the server received.
func recordingClient(t *testing.T, token string) (*databricks.WorkspaceClient, func() []string) {
	s := testserver.New(t)
	testserver.AddDefaultHandlers(s)
	var mu sync.Mutex
	var requests []string
	s.RequestCallback = func(r *testserver.Request) {
		mu.Lock()
		defer mu.Unlock()
		requests = append(requests, r.Method+" "+r.URL.Path)
	}
	w, err := databricks.NewWorkspaceClient(&databricks.Config{
		Host:        s.URL,
		Token:       testserver.UserNameTokenPrefix + token,
		WorkspaceID: strconv.Itoa(testserver.TestWorkspaceID),
	})
	require.NoError(t, err)
	return w, func() []string {
		mu.Lock()
		defer mu.Unlock()
		return append([]string(nil), requests...)
	}
}

func TestHelpersClientDefaultsToSelectedClient(t *testing.T) {
	selected := &databricks.WorkspaceClient{Config: &sdkconfig.Config{Host: "https://selected.test"}}
	ctx := cmdctx.SetWorkspaceClient(t.Context(), selected)
	assert.Same(t, selected, helpersClient(ctx))
}

func TestHelpersUseHelpersClient(t *testing.T) {
	selected, selectedRequests := recordingClient(t, "selected")
	helpers, helpersRequests := recordingClient(t, "helpers")
	ctx := cmdctx.SetWorkspaceClient(t.Context(), selected)
	ctx = WithHelpersClient(ctx, helpers)

	cachedUser = nil
	t.Cleanup(func() { cachedUser = nil })
	funcs := loadHelpers(ctx)

	host, err := funcs["workspace_host"].(func() (string, error))()
	require.NoError(t, err)
	assert.Equal(t, helpers.Config.Host, host)

	_, err = funcs["user_name"].(func() (string, error))()
	require.NoError(t, err)
	assert.Contains(t, helpersRequests(), "GET /api/2.0/preview/scim/v2/Me")
	assert.NotContains(t, selectedRequests(), "GET /api/2.0/preview/scim/v2/Me")
}
