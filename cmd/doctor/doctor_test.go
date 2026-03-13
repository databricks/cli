package doctor

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/databrickscfg/profile"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/config"
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

// noConfigProfiler returns a path but ErrNoConfiguration from LoadProfiles.
type noConfigProfiler struct {
	path string
}

func (m *noConfigProfiler) LoadProfiles(_ context.Context, _ profile.ProfileMatchFunction) (profile.Profiles, error) {
	return nil, profile.ErrNoConfiguration
}

func (m *noConfigProfiler) GetPath(_ context.Context) (string, error) {
	return m.path, nil
}

func newTestCmd(ctx context.Context) *cobra.Command {
	cmd := &cobra.Command{}
	cmd.SetContext(ctx)
	cmd.Flags().String("profile", "", "")
	return cmd
}

func clearConfigEnv(t *testing.T) {
	t.Helper()

	for _, attr := range config.ConfigAttributes {
		for _, key := range attr.EnvVars {
			t.Setenv(key, "")
		}
	}

	t.Setenv(env.HomeEnvVar(), t.TempDir())
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

func TestCheckConfigFileMissingWarn(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	ctx = profile.WithProfiler(ctx, &mockProfiler{
		path: "/home/user/.databrickscfg",
		err:  profile.ErrNoConfiguration,
	})
	cmd := newTestCmd(ctx)

	result := checkConfigFile(cmd)
	assert.Equal(t, "Config File", result.Name)
	// GetPath returns err first, so this hits the first failure branch.
	// To test the warn path, we need GetPath to succeed but LoadProfiles to fail.
	assert.Equal(t, statusFail, result.Status)
}

func TestCheckConfigFileAbsentIsWarn(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	// Profiler that returns a path but fails on LoadProfiles with ErrNoConfiguration.
	ctx = profile.WithProfiler(ctx, &noConfigProfiler{path: "/home/user/.databrickscfg"})
	cmd := newTestCmd(ctx)

	result := checkConfigFile(cmd)
	assert.Equal(t, "Config File", result.Name)
	assert.Equal(t, statusWarn, result.Status)
	assert.Contains(t, result.Message, "environment variables")
}

func TestCheckCurrentProfileDefault(t *testing.T) {
	clearConfigEnv(t)

	ctx := cmdio.MockDiscard(t.Context())
	cmd := newTestCmd(ctx)

	result := checkCurrentProfile(cmd)
	assert.Equal(t, "Current Profile", result.Name)
	assert.Equal(t, statusInfo, result.Status)
	assert.Equal(t, "none (using environment or defaults)", result.Message)
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

func TestCheckCurrentProfileFromEnv(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	ctx = env.Set(ctx, "DATABRICKS_CONFIG_PROFILE", "from-env")
	cmd := newTestCmd(ctx)

	result := checkCurrentProfile(cmd)
	assert.Equal(t, statusInfo, result.Status)
	assert.Equal(t, "from-env (from DATABRICKS_CONFIG_PROFILE)", result.Message)
}

func TestCheckCurrentProfileFlagOverridesEnv(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	ctx = env.Set(ctx, "DATABRICKS_CONFIG_PROFILE", "from-env")
	cmd := newTestCmd(ctx)
	err := cmd.Flag("profile").Value.Set("from-flag")
	require.NoError(t, err)
	cmd.Flag("profile").Changed = true

	result := checkCurrentProfile(cmd)
	assert.Equal(t, "from-flag", result.Message)
}

func TestCheckAuthSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	clearConfigEnv(t)
	t.Setenv("DATABRICKS_HOST", srv.URL)
	t.Setenv("DATABRICKS_TOKEN", "test-token")

	ctx := cmdio.MockDiscard(t.Context())
	cmd := newTestCmd(ctx)

	cfg, err := resolveConfig(cmd)
	require.NoError(t, err)

	result, w := checkAuth(cmd, cfg, err)
	assert.Equal(t, "Authentication", result.Name)
	assert.Equal(t, statusPass, result.Status)
	assert.Contains(t, result.Message, "OK")
	assert.NotNil(t, w)
}

func TestCheckAuthFailure(t *testing.T) {
	clearConfigEnv(t)

	ctx := cmdio.MockDiscard(t.Context())
	cmd := newTestCmd(ctx)

	cfg, err := resolveConfig(cmd)
	result, w := checkAuth(cmd, cfg, err)
	assert.Equal(t, "Authentication", result.Name)
	assert.Equal(t, statusFail, result.Status)
	assert.Nil(t, w)
}

func TestResolveConfigUsesCommandContextEnv(t *testing.T) {
	clearConfigEnv(t)
	t.Setenv("DATABRICKS_HOST", "https://real.example.com")
	t.Setenv("DATABRICKS_TOKEN", "real-token")

	ctx := cmdio.MockDiscard(t.Context())
	ctx = env.Set(ctx, "DATABRICKS_HOST", "https://context.example.com")
	ctx = env.Set(ctx, "DATABRICKS_TOKEN", "context-token")
	cmd := newTestCmd(ctx)

	cfg, err := resolveConfig(cmd)
	require.NoError(t, err)
	assert.Equal(t, "https://context.example.com", cfg.Host)
	assert.Equal(t, "context-token", cfg.Token)
}

func TestCheckIdentitySuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/2.0/preview/scim/v2/Me" {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"userName": "test@example.com"}`))
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	w, err := databricks.NewWorkspaceClient((*databricks.Config)(&config.Config{
		Host:  srv.URL,
		Token: "test-token",
	}))
	require.NoError(t, err)

	ctx := cmdio.MockDiscard(t.Context())
	cmd := newTestCmd(ctx)

	result := checkIdentity(cmd, w)
	assert.Equal(t, "Identity", result.Name)
	assert.Equal(t, statusPass, result.Status)
	assert.Equal(t, "test@example.com", result.Message)
}

func TestCheckIdentityFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	w, err := databricks.NewWorkspaceClient((*databricks.Config)(&config.Config{
		Host:  srv.URL,
		Token: "bad-token",
	}))
	require.NoError(t, err)

	ctx := cmdio.MockDiscard(t.Context())
	cmd := newTestCmd(ctx)

	result := checkIdentity(cmd, w)
	assert.Equal(t, "Identity", result.Name)
	assert.Equal(t, statusFail, result.Status)
}

func TestCheckNetworkReachable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	ctx := cmdio.MockDiscard(t.Context())
	cmd := newTestCmd(ctx)

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

func TestCheckNetworkWithClient(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	w, err := databricks.NewWorkspaceClient((*databricks.Config)(&config.Config{
		Host:  srv.URL,
		Token: "test-token",
	}))
	require.NoError(t, err)

	ctx := cmdio.MockDiscard(t.Context())
	cmd := newTestCmd(ctx)

	result := checkNetwork(cmd, w.Config, nil, w)
	assert.Equal(t, "Network", result.Name)
	assert.Equal(t, statusPass, result.Status)
	assert.Contains(t, result.Message, "reachable")
}

func TestCheckNetworkConfigResolutionFailure(t *testing.T) {
	clearConfigEnv(t)

	configFile := filepath.Join(t.TempDir(), ".databrickscfg")
	err := os.WriteFile(configFile, []byte("[DEFAULT]\nhost = https://example.com\n"), 0o600)
	require.NoError(t, err)

	ctx := cmdio.MockDiscard(t.Context())
	ctx = env.Set(ctx, "DATABRICKS_CONFIG_FILE", configFile)
	ctx = env.Set(ctx, "DATABRICKS_CONFIG_PROFILE", "missing")
	cmd := newTestCmd(ctx)

	cfg, err := resolveConfig(cmd)
	require.Error(t, err)

	result := checkNetwork(cmd, cfg, err, nil)
	assert.Equal(t, "Network", result.Name)
	assert.Equal(t, statusFail, result.Status)
	assert.Equal(t, "Cannot resolve config", result.Message)
	assert.Contains(t, result.Detail, "missing profile")
}

func TestRenderResultsText(t *testing.T) {
	results := []CheckResult{
		{Name: "Test", Status: statusPass, Message: "all good"},
		{Name: "Another", Status: statusFail, Message: "broken", Detail: "details here"},
		{Name: "Version", Status: statusInfo, Message: "1.0.0"},
		{Name: "Config", Status: statusWarn, Message: "not found"},
	}

	var buf bytes.Buffer
	renderResults(&buf, results)
	output := buf.String()
	assert.Contains(t, output, "Test")
	assert.Contains(t, output, "all good")
	assert.Contains(t, output, "broken")
	assert.Contains(t, output, "details here")
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

func TestNewCommandJSON(t *testing.T) {
	clearConfigEnv(t)

	ctx := cmdio.MockDiscard(t.Context())
	ctx = profile.WithProfiler(ctx, &mockProfiler{
		path: "/tmp/.databrickscfg",
		profiles: profile.Profiles{
			{Name: "default", Host: "https://example.com"},
		},
	})

	cmd := New()
	cmd.SetContext(ctx)

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

	assert.Equal(t, "CLI Version", results[0].Name)
	assert.Equal(t, statusInfo, results[0].Status)
}

func TestNewCommandJSONTrailingNewline(t *testing.T) {
	clearConfigEnv(t)

	ctx := cmdio.MockDiscard(t.Context())
	ctx = profile.WithProfiler(ctx, &mockProfiler{
		path: "/tmp/.databrickscfg",
		profiles: profile.Profiles{
			{Name: "default", Host: "https://example.com"},
		},
	})

	cmd := New()
	cmd.SetContext(ctx)

	outputFlag := flags.OutputText
	cmd.PersistentFlags().VarP(&outputFlag, "output", "o", "output type: text or json")

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"--output", "json"})

	err := cmd.Execute()
	require.NoError(t, err)
	assert.True(t, buf.Len() > 0)
	assert.Equal(t, byte('\n'), buf.Bytes()[buf.Len()-1])
}
