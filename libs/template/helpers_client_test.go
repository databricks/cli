package template

import (
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"testing"

	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/databricks-sdk-go"
	sdkconfig "github.com/databricks/databricks-sdk-go/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// recordingServer returns a test server and a function that returns the requests it received.
func recordingServer(t *testing.T) (*testserver.Server, func() []string) {
	s := testserver.New(t)
	testserver.AddDefaultHandlers(s)
	var mu sync.Mutex
	var requests []string
	s.RequestCallback = func(r *testserver.Request) {
		mu.Lock()
		defer mu.Unlock()
		requests = append(requests, r.Method+" "+r.URL.Path)
	}
	return s, func() []string {
		mu.Lock()
		defer mu.Unlock()
		return append([]string(nil), requests...)
	}
}

func writeConfigFile(t *testing.T, content string) {
	path := filepath.Join(t.TempDir(), ".databrickscfg")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	t.Setenv("DATABRICKS_CONFIG_FILE", path)
}

func TestHelpersClientWithoutProfileUsesSelectedClient(t *testing.T) {
	testutil.CleanupEnvironment(t)
	selected := &databricks.WorkspaceClient{Config: &sdkconfig.Config{Host: "https://selected.test"}}
	ctx := cmdctx.SetWorkspaceClient(t.Context(), selected)

	ctx, err := ConfigureHelpersClient(ctx)
	require.NoError(t, err)
	assert.Same(t, selected, helpersClient(ctx))
}

func TestHelpersClientUsesHelpersProfile(t *testing.T) {
	testutil.CleanupEnvironment(t)
	workspaceID := strconv.Itoa(testserver.TestWorkspaceID)

	helpersServer, helpersRequests := recordingServer(t)
	selectedServer, selectedRequests := recordingServer(t)
	writeConfigFile(t, "[classic]\nhost = "+helpersServer.URL+"\nworkspace_id = "+workspaceID+"\ntoken = "+testserver.UserNameTokenPrefix+"classic\n")

	selected, err := databricks.NewWorkspaceClient(&databricks.Config{
		Host:        selectedServer.URL,
		Token:       testserver.UserNameTokenPrefix + "selected",
		WorkspaceID: workspaceID,
	})
	require.NoError(t, err)
	ctx := cmdctx.SetWorkspaceClient(t.Context(), selected)
	ctx = env.Set(ctx, helpersProfileEnvVar, "classic")

	ctx, err = ConfigureHelpersClient(ctx)
	require.NoError(t, err)

	cachedUser = nil
	t.Cleanup(func() { cachedUser = nil })
	helpers := loadHelpers(ctx)

	host, err := helpers["workspace_host"].(func() (string, error))()
	require.NoError(t, err)
	assert.Equal(t, helpersServer.URL, host)

	_, err = helpers["user_name"].(func() (string, error))()
	require.NoError(t, err)
	assert.Contains(t, helpersRequests(), "GET /api/2.0/preview/scim/v2/Me")
	assert.NotContains(t, selectedRequests(), "GET /api/2.0/preview/scim/v2/Me")
}

func TestConfigureHelpersClientErrors(t *testing.T) {
	tests := []struct {
		name    string
		profile string
		wantErr string
	}{
		{
			name:    "other workspace",
			profile: "other",
			wantErr: `profile "other" from DATABRICKS_BUNDLE_INIT_HELPERS_PROFILE has workspace ID 222, but the selected profile has workspace ID 111`,
		},
		{
			name:    "missing profile",
			profile: "missing",
			wantErr: `failed to load profile "missing" from DATABRICKS_BUNDLE_INIT_HELPERS_PROFILE: `,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testutil.CleanupEnvironment(t)
			writeConfigFile(t, "[other]\nhost = https://other.test\nworkspace_id = 222\ntoken = x\n")
			selected := &databricks.WorkspaceClient{Config: &sdkconfig.Config{Host: "https://selected.test", WorkspaceID: "111"}}
			ctx := cmdctx.SetWorkspaceClient(t.Context(), selected)
			ctx = env.Set(ctx, helpersProfileEnvVar, tt.profile)

			_, err := ConfigureHelpersClient(ctx)
			assert.ErrorContains(t, err, tt.wantErr)
		})
	}
}
