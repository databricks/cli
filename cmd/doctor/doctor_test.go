package doctor

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/databrickscfg/profile"
	"github.com/databricks/cli/libs/flags"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockProfiler struct {
	profiles profile.Profiles
	path     string
	err      error
}

func (m *mockProfiler) LoadProfiles(_ context.Context, match profile.ProfileMatchFunction) (profile.Profiles, error) {
	if m.err != nil {
		return nil, m.err
	}
	var result profile.Profiles
	for _, p := range m.profiles {
		if match(p) {
			result = append(result, p)
		}
	}
	return result, nil
}

func (m *mockProfiler) GetPath(_ context.Context) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return m.path, nil
}

func newTestCmd(ctx context.Context) *cobra.Command {
	cmd := &cobra.Command{}
	cmd.SetContext(ctx)
	cmd.Flags().String("profile", "", "")
	return cmd
}

func TestCheckCLIVersion(t *testing.T) {
	result := checkCLIVersion()
	assert.Equal(t, "CLI Version", result.Name)
	assert.Equal(t, statusInfo, result.Status)
	assert.NotEmpty(t, result.Message)
}

func TestCheckConfigFilePass(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	ctx = profile.WithProfiler(ctx, &mockProfiler{
		path: "/home/user/.databrickscfg",
		profiles: profile.Profiles{
			{Name: "default", Host: "https://example.com"},
			{Name: "staging", Host: "https://staging.example.com"},
		},
	})
	cmd := newTestCmd(ctx)

	result := checkConfigFile(cmd)
	assert.Equal(t, "Config File", result.Name)
	assert.Equal(t, statusPass, result.Status)
	assert.Contains(t, result.Message, "2 profiles")
	assert.Contains(t, result.Message, "/home/user/.databrickscfg")
}

func TestCheckConfigFileMissing(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	ctx = profile.WithProfiler(ctx, &mockProfiler{
		err: profile.ErrNoConfiguration,
	})
	cmd := newTestCmd(ctx)

	result := checkConfigFile(cmd)
	assert.Equal(t, "Config File", result.Name)
	assert.Equal(t, statusFail, result.Status)
}

func TestCheckCurrentProfileDefault(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	cmd := newTestCmd(ctx)

	result := checkCurrentProfile(cmd)
	assert.Equal(t, "Current Profile", result.Name)
	assert.Equal(t, statusInfo, result.Status)
	assert.Equal(t, "default", result.Message)
}

func TestCheckCurrentProfileFromFlag(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	cmd := newTestCmd(ctx)
	err := cmd.Flag("profile").Value.Set("staging")
	require.NoError(t, err)
	cmd.Flag("profile").Changed = true

	result := checkCurrentProfile(cmd)
	assert.Equal(t, "Current Profile", result.Name)
	assert.Equal(t, statusInfo, result.Status)
	assert.Equal(t, "staging", result.Message)
}

func TestCheckNetworkReachable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	ctx := cmdio.MockDiscard(t.Context())
	cmd := newTestCmd(ctx)

	// Create a fake workspace client config to simulate having a host.
	// We pass nil for the workspace client and let it fall through to resolving from config.
	// Instead, directly test with a real server.
	result := checkNetworkWithHost(cmd, srv.URL)
	assert.Equal(t, "Network", result.Name)
	assert.Equal(t, statusPass, result.Status)
	assert.Contains(t, result.Message, "reachable")
}

func TestCheckNetworkNoHost(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	cmd := newTestCmd(ctx)

	result := checkNetworkWithHost(cmd, "")
	assert.Equal(t, "Network", result.Name)
	assert.Equal(t, statusFail, result.Status)
	assert.Contains(t, result.Message, "No host configured")
}

func TestRenderResultsText(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	results := []CheckResult{
		{Name: "Test", Status: statusPass, Message: "all good"},
		{Name: "Another", Status: statusFail, Message: "broken", Detail: "details here"},
		{Name: "Version", Status: statusInfo, Message: "1.0.0"},
	}

	// Should not panic.
	renderResults(ctx, results)
}

func TestRenderResultsJSON(t *testing.T) {
	results := []CheckResult{
		{Name: "Test", Status: statusPass, Message: "all good"},
		{Name: "Another", Status: statusFail, Message: "broken", Detail: "details here"},
	}

	buf, err := json.MarshalIndent(results, "", "  ")
	require.NoError(t, err)

	var parsed []CheckResult
	err = json.Unmarshal(buf, &parsed)
	require.NoError(t, err)
	assert.Len(t, parsed, 2)
	assert.Equal(t, "Test", parsed[0].Name)
	assert.Equal(t, statusPass, parsed[0].Status)
	assert.Equal(t, "broken", parsed[1].Message)
	assert.Equal(t, "details here", parsed[1].Detail)
}

func TestRenderResultsJSONOmitsEmptyDetail(t *testing.T) {
	results := []CheckResult{
		{Name: "Test", Status: statusPass, Message: "ok"},
	}

	buf, err := json.Marshal(results)
	require.NoError(t, err)
	assert.NotContains(t, string(buf), "detail")
}

func TestAuthFailureBecomesCheckResult(t *testing.T) {
	// Unset env vars that might allow auth to succeed.
	t.Setenv("DATABRICKS_HOST", "")
	t.Setenv("DATABRICKS_TOKEN", "")
	t.Setenv("DATABRICKS_CONFIG_PROFILE", "")
	t.Setenv("HOME", t.TempDir())

	ctx := cmdio.MockDiscard(t.Context())
	cmd := newTestCmd(ctx)

	result, w := checkAuth(cmd)
	// Auth should fail since there is no configuration.
	assert.Equal(t, "Authentication", result.Name)
	assert.Equal(t, statusFail, result.Status)
	assert.Nil(t, w)
}

func TestNewCommandJSON(t *testing.T) {
	// Unset env vars that might cause side effects.
	t.Setenv("DATABRICKS_HOST", "")
	t.Setenv("DATABRICKS_TOKEN", "")
	t.Setenv("DATABRICKS_CONFIG_PROFILE", "")
	t.Setenv("HOME", t.TempDir())

	ctx := cmdio.MockDiscard(t.Context())
	ctx = profile.WithProfiler(ctx, &mockProfiler{
		path: "/tmp/.databrickscfg",
		profiles: profile.Profiles{
			{Name: "default", Host: "https://example.com"},
		},
	})

	cmd := New()
	cmd.SetContext(ctx)

	// Register the output flag that is normally inherited from the root command.
	outputFlag := flags.OutputText
	cmd.PersistentFlags().VarP(&outputFlag, "output", "o", "output type: text or json")

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"--output", "json"})

	err := cmd.Execute()
	require.NoError(t, err)

	var results []CheckResult
	err = json.Unmarshal(buf.Bytes(), &results)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(results), 4)

	// First check should be CLI Version.
	assert.Equal(t, "CLI Version", results[0].Name)
	assert.Equal(t, statusInfo, results[0].Status)
}
