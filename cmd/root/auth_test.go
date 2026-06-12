package root

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

// noNetworkTransport prevents real HTTP calls in auth tests.
// Returns 404 for all requests so host metadata resolution falls back gracefully.
var noNetworkTransport = roundTripperFunc(func(r *http.Request) (*http.Response, error) {
	return &http.Response{StatusCode: http.StatusNotFound, Body: http.NoBody}, nil
})

func TestEmptyHttpRequest(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	req := emptyHttpRequest(ctx)
	assert.Equal(t, req.Context(), ctx)
}

type promptFn func(ctx context.Context, cfg *config.Config, retry bool) (any, error)

var accountPromptFn = func(ctx context.Context, cfg *config.Config, retry bool) (any, error) {
	return accountClientOrPrompt(ctx, cfg, retry)
}

var workspacePromptFn = func(ctx context.Context, cfg *config.Config, retry bool) (any, error) {
	return workspaceClientOrPrompt(ctx, cfg, retry)
}

func expectPrompts(t *testing.T, fn promptFn, cfg *config.Config) {
	// Prevent real HTTP calls during auth resolution.
	cfg.HTTPTransport = noNetworkTransport

	ctx, cancel := context.WithTimeout(t.Context(), 1*time.Second)
	defer cancel()

	// Channel to pass errors from the prompting function back to the test.
	errch := make(chan error, 1)

	ctx, io := cmdio.SetupTest(ctx, cmdio.TestOptions{PromptSupported: true})
	go func() {
		defer close(errch)
		defer cancel()
		_, err := fn(ctx, cfg, true)
		errch <- err
	}()

	// Expect a prompt
	line, _, err := io.Stderr.ReadLine()
	if assert.NoError(t, err, "Expected to read a line from stderr") {
		assert.Contains(t, string(line), "Search:")
	} else {
		// If there was an error reading from stderr, the prompting function must have terminated early.
		assert.NoError(t, <-errch)
	}
}

func expectReturns(t *testing.T, fn promptFn, cfg *config.Config) {
	// Prevent real HTTP calls during auth resolution.
	cfg.HTTPTransport = noNetworkTransport

	ctx, cancel := context.WithTimeout(t.Context(), 1*time.Second)
	defer cancel()

	ctx, _ = cmdio.SetupTest(ctx, cmdio.TestOptions{PromptSupported: true})
	client, err := fn(ctx, cfg, true)
	require.NoError(t, err)
	require.NotNil(t, client)
}

func TestAccountClientOrPrompt(t *testing.T) {
	testutil.CleanupEnvironment(t)

	dir := t.TempDir()
	configFile := filepath.Join(dir, ".databrickscfg")
	err := os.WriteFile(
		configFile,
		[]byte(`
			[account-1111]
			host = https://accounts.azuredatabricks.net/
			account_id = 1111
			token = foobar

			[account-1112]
			host = https://accounts.azuredatabricks.net/
			account_id = 1112
			token = foobar
			`),
		0o755)
	require.NoError(t, err)
	t.Setenv("DATABRICKS_CONFIG_FILE", configFile)
	// Clear PATH to prevent the SDK from invoking external tools (e.g. az) during auth resolution.
	t.Setenv("PATH", "")

	t.Run("Prompt if nothing is specified", func(t *testing.T) {
		expectPrompts(t, accountPromptFn, &config.Config{})
	})

	t.Run("Returns if a workspace host is specified with valid auth and account ID", func(t *testing.T) {
		// If auth succeeds and an account ID is present, trust the SDK's resolution.
		// This supports unified hosts where HostType() returns WorkspaceHost but
		// account APIs are available.
		expectReturns(t, accountPromptFn, &config.Config{
			Host:      "https://adb-1234567.89.azuredatabricks.net/",
			AccountID: "1234",
			Token:     "foobar",
		})
	})

	t.Run("Prompt if account ID is not specified", func(t *testing.T) {
		expectPrompts(t, accountPromptFn, &config.Config{
			Host:  "https://accounts.azuredatabricks.net/",
			Token: "foobar",
		})
	})

	t.Run("Prompt if no credential provider can be configured", func(t *testing.T) {
		// The SDK probes all auth types when not specified and this fails for the u2m probe on Windows.
		t.Skip("Skipping as of #2920")

		expectPrompts(t, accountPromptFn, &config.Config{
			Host:      "https://accounts.azuredatabricks.net/",
			AccountID: "1234",

			// Force SDK to not try and lookup the tenant ID from the host.
			// The host above is invalid and will not be reachable.
			AzureTenantID: "nonempty",
		})
	})

	t.Run("Returns if configuration is valid", func(t *testing.T) {
		expectReturns(t, accountPromptFn, &config.Config{
			Host:      "https://accounts.azuredatabricks.net/",
			AccountID: "1234",
			Token:     "foobar",
		})
	})

	t.Run("Returns if a valid profile is specified", func(t *testing.T) {
		expectReturns(t, accountPromptFn, &config.Config{
			Profile: "account-1111",
		})
	})
}

func TestWorkspaceClientOrPrompt(t *testing.T) {
	testutil.CleanupEnvironment(t)

	dir := t.TempDir()
	configFile := filepath.Join(dir, ".databrickscfg")
	err := os.WriteFile(
		configFile,
		[]byte(`
			[workspace-1111]
			host = https://adb-1111.11.azuredatabricks.net/
			token = foobar

			[workspace-1112]
			host = https://adb-1112.12.azuredatabricks.net/
			token = foobar
			`),
		0o755)
	require.NoError(t, err)
	t.Setenv("DATABRICKS_CONFIG_FILE", configFile)
	// Clear PATH to prevent the SDK from invoking external tools (e.g. az) during auth resolution.
	t.Setenv("PATH", "")

	t.Run("Prompt if nothing is specified", func(t *testing.T) {
		expectPrompts(t, workspacePromptFn, &config.Config{})
	})

	t.Run("Returns if an account host is specified with valid auth", func(t *testing.T) {
		// If auth succeeds, trust the SDK's resolution. This supports unified
		// hosts where HostType() returns AccountHost but workspace APIs are
		// available.
		expectReturns(t, workspacePromptFn, &config.Config{
			Host:      "https://accounts.azuredatabricks.net/",
			AccountID: "1234",
			Token:     "foobar",
		})
	})

	t.Run("Prompt if no credential provider can be configured", func(t *testing.T) {
		// The SDK probes all auth types when not specified and this fails for the u2m probe on Windows.
		t.Skip("Skipping as of #2920")

		expectPrompts(t, workspacePromptFn, &config.Config{
			Host: "https://adb-1111.11.azuredatabricks.net/",

			// Force SDK to not try and lookup the tenant ID from the host.
			// The host above is invalid and will not be reachable.
			AzureTenantID: "nonempty",
		})
	})

	t.Run("Returns if configuration is valid", func(t *testing.T) {
		expectReturns(t, workspacePromptFn, &config.Config{
			Host:  "https://adb-1111.11.azuredatabricks.net/",
			Token: "foobar",
		})
	})

	t.Run("Returns if a valid profile is specified", func(t *testing.T) {
		expectReturns(t, workspacePromptFn, &config.Config{
			Profile: "workspace-1111",
		})
	})
}

func TestMustAccountClientWorksWithDatabricksCfg(t *testing.T) {
	testutil.CleanupEnvironment(t)

	dir := t.TempDir()
	configFile := filepath.Join(dir, ".databrickscfg")
	err := os.WriteFile(
		configFile,
		[]byte(`
			[account-1111]
			host = https://accounts.azuredatabricks.net/
			account_id = 1111
			token = foobar
			`),
		0o755)
	require.NoError(t, err)

	cmd := New(t.Context())

	t.Setenv("DATABRICKS_CONFIG_FILE", configFile)
	err = MustAccountClient(cmd, []string{})
	require.NoError(t, err)
}

func TestMustAccountClientWorksWithNoDatabricksCfgButEnvironmentVariables(t *testing.T) {
	testutil.CleanupEnvironment(t)

	ctx, tt := cmdio.SetupTest(t.Context(), cmdio.TestOptions{PromptSupported: true})
	t.Cleanup(tt.Done)
	cmd := New(ctx)
	t.Setenv("DATABRICKS_HOST", "https://accounts.azuredatabricks.net/")
	t.Setenv("DATABRICKS_TOKEN", "foobar")
	t.Setenv("DATABRICKS_ACCOUNT_ID", "1111")

	err := MustAccountClient(cmd, []string{})
	require.NoError(t, err)
}

func TestMustAccountClientErrorsWithNoDatabricksCfg(t *testing.T) {
	testutil.CleanupEnvironment(t)

	ctx, tt := cmdio.SetupTest(t.Context(), cmdio.TestOptions{PromptSupported: true})
	t.Cleanup(tt.Done)
	cmd := New(ctx)

	err := MustAccountClient(cmd, []string{})
	require.ErrorContains(t, err, "no configuration file found at")
}

func TestMustAnyClientCanCreateWorkspaceClient(t *testing.T) {
	testutil.CleanupEnvironment(t)
	// Clear PATH to prevent the SDK from invoking external tools (e.g. az) during auth resolution.
	t.Setenv("PATH", "")

	dir := t.TempDir()
	configFile := filepath.Join(dir, ".databrickscfg")
	err := os.WriteFile(
		configFile,
		[]byte(`
			[workspace-1111]
			host = https://adb-1111.11.azuredatabricks.net/
			token = foobar
			`),
		0o755)
	require.NoError(t, err)

	ctx, tt := cmdio.SetupTest(t.Context(), cmdio.TestOptions{PromptSupported: true})
	t.Cleanup(tt.Done)
	cmd := New(ctx)

	t.Setenv("DATABRICKS_CONFIG_FILE", configFile)
	isAccount, err := MustAnyClient(cmd, []string{})
	require.False(t, isAccount)
	require.NoError(t, err)

	w := cmdctx.WorkspaceClient(cmd.Context())
	require.NotNil(t, w)
}

func TestMustAnyClientCanCreateAccountClient(t *testing.T) {
	testutil.CleanupEnvironment(t)
	// Clear PATH to prevent the SDK from invoking external tools (e.g. az) during auth resolution.
	t.Setenv("PATH", "")

	dir := t.TempDir()
	configFile := filepath.Join(dir, ".databrickscfg")
	err := os.WriteFile(
		configFile,
		[]byte(`
			[account-1111]
			host = https://accounts.azuredatabricks.net/
			account_id = 1111
			token = foobar
			`),
		0o755)
	require.NoError(t, err)

	ctx, tt := cmdio.SetupTest(t.Context(), cmdio.TestOptions{PromptSupported: true})
	t.Cleanup(tt.Done)
	cmd := New(ctx)

	t.Setenv("DATABRICKS_CONFIG_FILE", configFile)
	isAccount, err := MustAnyClient(cmd, []string{})
	require.NoError(t, err)
	require.True(t, isAccount)

	a := cmdctx.AccountClient(cmd.Context())
	require.NotNil(t, a)
}

func TestMustAnyClientWithEmptyDatabricksCfg(t *testing.T) {
	testutil.CleanupEnvironment(t)
	// Clear PATH to prevent the SDK from invoking external tools (e.g. az) during auth resolution.
	t.Setenv("PATH", "")

	dir := t.TempDir()
	configFile := filepath.Join(dir, ".databrickscfg")
	err := os.WriteFile(
		configFile,
		[]byte(""), // empty file
		0o755)
	require.NoError(t, err)

	ctx, tt := cmdio.SetupTest(t.Context(), cmdio.TestOptions{PromptSupported: true})
	t.Cleanup(tt.Done)
	cmd := New(ctx)

	t.Setenv("DATABRICKS_CONFIG_FILE", configFile)

	_, err = MustAnyClient(cmd, []string{})
	require.ErrorContains(t, err, "does not contain account profiles")
}

func TestMustWorkspaceClientDefaultProfilePrecedence(t *testing.T) {
	testutil.CleanupEnvironment(t)

	configFile := filepath.Join(t.TempDir(), ".databrickscfg")
	err := os.WriteFile(configFile, []byte(`
[__settings__]
default_profile = settings-profile

[DEFAULT]
host = https://default.cloud.databricks.com
token = default-token

[settings-profile]
host = https://settings.cloud.databricks.com
token = settings-token

[env-profile]
host = https://env.cloud.databricks.com
token = env-token

[flag-profile]
host = https://flag.cloud.databricks.com
token = flag-token
`), 0o600)
	require.NoError(t, err)

	testCases := []struct {
		name        string
		profileFlag string
		envProfile  string
		wantProfile string
		wantHost    string
	}{
		{
			name:        "settings default is used when flag and env are unset",
			wantProfile: "settings-profile",
			wantHost:    "https://settings.cloud.databricks.com",
		},
		{
			name:        "env var takes precedence over settings default",
			envProfile:  "env-profile",
			wantProfile: "env-profile",
			wantHost:    "https://env.cloud.databricks.com",
		},
		{
			name:        "profile flag takes precedence over env var",
			profileFlag: "flag-profile",
			envProfile:  "env-profile",
			wantProfile: "flag-profile",
			wantHost:    "https://flag.cloud.databricks.com",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testutil.CleanupEnvironment(t)
			t.Setenv("DATABRICKS_CONFIG_FILE", configFile)
			if tc.envProfile != "" {
				t.Setenv("DATABRICKS_CONFIG_PROFILE", tc.envProfile)
			}

			ctx := cmdio.MockDiscard(t.Context())
			ctx = SkipLoadBundle(ctx)
			cmd := New(ctx)

			if tc.profileFlag != "" {
				err := cmd.Flag("profile").Value.Set(tc.profileFlag)
				require.NoError(t, err)
			}

			err := MustWorkspaceClient(cmd, []string{})
			require.NoError(t, err)

			w := cmdctx.WorkspaceClient(cmd.Context())
			require.NotNil(t, w)
			assert.Equal(t, tc.wantProfile, w.Config.Profile)
			assert.Equal(t, tc.wantHost, w.Config.Host)
		})
	}
}

func TestAccountClientOrPromptReturnsErrorForWrongHostType(t *testing.T) {
	testutil.CleanupEnvironment(t)
	t.Setenv("PATH", "")

	cfg := &config.Config{
		Host:          "https://adb-1234567.89.azuredatabricks.net/",
		Token:         "foobar",
		HTTPTransport: noNetworkTransport,
	}

	a, err := accountClientOrPrompt(t.Context(), cfg, false)
	assert.NotNil(t, a)
	assert.ErrorIs(t, err, databricks.ErrNotAccountClient)
}

func TestIsPATOnSPOGWithoutWorkspaceID(t *testing.T) {
	tests := []struct {
		name string
		cfg  *config.Config
		want bool
	}{
		{
			name: "pat on spog without workspace_id",
			cfg: &config.Config{
				AuthType:     "pat",
				DiscoveryURL: "https://spog.example.test/oidc/accounts/abc/.well-known/oauth-authorization-server",
			},
			want: true,
		},
		{
			name: "pat on spog with workspace_id is fine",
			cfg: &config.Config{
				AuthType:     "pat",
				WorkspaceID:  "12345",
				DiscoveryURL: "https://spog.example.test/oidc/accounts/abc/.well-known/oauth-authorization-server",
			},
			want: false,
		},
		{
			name: "pat on spog with legacy 'none' sentinel is treated as missing",
			cfg: &config.Config{
				AuthType:     "pat",
				WorkspaceID:  auth.WorkspaceIDNone,
				DiscoveryURL: "https://spog.example.test/oidc/accounts/abc/.well-known/oauth-authorization-server",
			},
			want: true,
		},
		{
			name: "pat on classic workspace host is fine",
			cfg: &config.Config{
				AuthType:     "pat",
				DiscoveryURL: "https://workspace.example.test/oidc/.well-known/oauth-authorization-server",
			},
			want: false,
		},
		{
			name: "u2m on spog is not affected (handled by other paths)",
			cfg: &config.Config{
				AuthType:     "databricks-cli",
				DiscoveryURL: "https://spog.example.test/oidc/accounts/abc/.well-known/oauth-authorization-server",
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, isPATOnSPOGWithoutWorkspaceID(tt.cfg))
		})
	}
}

func TestWorkspaceClientOrPromptRejectsAccountOnlyProfile(t *testing.T) {
	tests := []struct {
		name        string
		workspaceID string
	}{
		// New shape: --skip-workspace omits workspace_id entirely.
		{name: "empty workspace_id", workspaceID: ""},
		// Legacy shape: older CLIs persisted the "none" sentinel.
		{name: "legacy none sentinel", workspaceID: "none"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testutil.CleanupEnvironment(t)
			t.Setenv("PATH", "")

			cfg := &config.Config{
				Host:          "https://example.test/",
				AccountID:     "abc-123",
				WorkspaceID:   tt.workspaceID,
				Token:         "foobar",
				Profile:       "bb",
				HTTPTransport: noNetworkTransport,
			}

			w, err := workspaceClientOrPrompt(t.Context(), cfg, false)
			assert.Nil(t, w)
			require.Error(t, err)
			var accountOnly ErrAccountOnlyProfile
			require.ErrorAs(t, err, &accountOnly)
			assert.Contains(t, err.Error(), `profile "bb"`)
			assert.Contains(t, err.Error(), "account-only")
			assert.Contains(t, err.Error(), "no workspace_id set")
		})
	}
}

func TestErrAccountOnlyProfileMessage(t *testing.T) {
	tests := []struct {
		name string
		err  ErrAccountOnlyProfile
		want string
	}{
		{
			name: "account console host",
			err:  ErrAccountOnlyProfile{profileName: "acc", host: "https://accounts.test"},
			want: "profile \"acc\" points to a Databricks account console host (https://accounts.test), which serves only account-level APIs; " +
				"this command requires a workspace. Run `databricks auth login --host https://<workspace-url>` to create a workspace profile, " +
				"or use `databricks account ...` commands with this profile",
		},
		{
			// On non-account-console hosts (SPOG/unified) workspace APIs are
			// served, so setting workspace_id is still the right fix.
			name: "other host keeps workspace_id advice",
			err:  ErrAccountOnlyProfile{profileName: "spog", host: "https://unified.test"},
			want: "profile \"spog\" has no workspace_id set (account-only); this command requires a workspace. " +
				"Edit the profile to set workspace_id to a real ID, or pass --profile with a workspace-scoped profile",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.err.Error())
		})
	}
}

func TestWorkspaceClientOrPromptAccountOnlyProfileOnAccountConsoleHost(t *testing.T) {
	testutil.CleanupEnvironment(t)
	t.Setenv("PATH", "")

	cfg := &config.Config{
		Host:          "https://accounts.test/",
		AccountID:     "abc-123",
		Token:         "foobar",
		Profile:       "acc",
		HTTPTransport: noNetworkTransport,
	}

	w, err := workspaceClientOrPrompt(t.Context(), cfg, false)
	assert.Nil(t, w)
	require.Error(t, err)
	var accountOnly ErrAccountOnlyProfile
	require.ErrorAs(t, err, &accountOnly)
	assert.Contains(t, err.Error(), "account console host (https://accounts.test)")
	assert.Contains(t, err.Error(), "databricks auth login --host")
	assert.NotContains(t, err.Error(), "set workspace_id to a real ID")
}

func TestWorkspaceClientOrPromptRejectsPATOnSPOGWithoutWorkspaceID(t *testing.T) {
	testutil.CleanupEnvironment(t)
	t.Setenv("PATH", "")

	// No AccountID is set, so the account-only profile detector (which requires
	// AccountID) does not fire and the PAT-on-SPOG detector is exercised.
	cfg := &config.Config{
		Host:          "https://spog.example.test/",
		Token:         "dapi-fake",
		Profile:       "spog-pat",
		DiscoveryURL:  "https://spog.example.test/oidc/accounts/abc-123/.well-known/oauth-authorization-server",
		AuthType:      "pat",
		HTTPTransport: noNetworkTransport,
	}

	w, err := workspaceClientOrPrompt(t.Context(), cfg, false)
	assert.Nil(t, w)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `profile "spog-pat"`)
	assert.Contains(t, err.Error(), "workspace_id")
	assert.Contains(t, err.Error(), "PAT")
}

// TestWorkspaceClientOrPromptRejectsPATOnSPOGFromConfigFile exercises the
// real .databrickscfg shape from the bug bash: `host` + `token` only, no
// `auth_type`, no `workspace_id`. The SDK populates AuthType during
// NewWorkspaceClient via its credential probe, so the PAT-on-SPOG detector
// must keep working after going through that path.
func TestWorkspaceClientOrPromptRejectsPATOnSPOGFromConfigFile(t *testing.T) {
	// testutil.CleanupEnvironment calls os.Clearenv(), which wipes Windows
	// essentials like SystemRoot and breaks Winsock initialization for
	// subsequent net.Listen calls. We only need a clean DATABRICKS_CONFIG_FILE
	// for this test; set it directly with t.Setenv so the rest of the
	// environment (notably the Windows networking stack) keeps working.
	t.Setenv("DATABRICKS_AUTH_TYPE", "")
	t.Setenv("DATABRICKS_HOST", "")
	t.Setenv("DATABRICKS_TOKEN", "")
	t.Setenv("DATABRICKS_CONFIG_PROFILE", "")
	t.Setenv("PATH", "")

	// Mock .well-known/databricks-config to return an account-scoped OIDC
	// endpoint so the SDK populates cfg.DiscoveryURL with the SPOG signal.
	// Omit account_id so AccountID stays unset; otherwise the account-only
	// profile detector would intercept this case before the PAT-on-SPOG check.
	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/databricks-config", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"oidc_endpoint":"https://spog.example.test/oidc/accounts/abc-123"}`))
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	configFile := filepath.Join(t.TempDir(), ".databrickscfg")
	require.NoError(t, os.WriteFile(configFile, fmt.Appendf(nil, `
[spog-pat]
host  = %s
token = dapi-fake
`, server.URL), 0o600))
	t.Setenv("DATABRICKS_CONFIG_FILE", configFile)

	cfg := &config.Config{Profile: "spog-pat"}
	w, err := workspaceClientOrPrompt(t.Context(), cfg, false)
	assert.Nil(t, w)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `profile "spog-pat"`)
	assert.Contains(t, err.Error(), "workspace_id")
	assert.Contains(t, err.Error(), "PAT")
}

func TestMustAnyClientFallsThroughOnAccountOnlyProfile(t *testing.T) {
	testutil.CleanupEnvironment(t)
	t.Setenv("PATH", "")

	configFile := filepath.Join(t.TempDir(), ".databrickscfg")
	err := os.WriteFile(configFile, []byte(`
[skipws]
host         = https://accounts.azuredatabricks.net/
account_id   = abc-123
token        = foobar
workspace_id = none
`), 0o600)
	require.NoError(t, err)
	t.Setenv("DATABRICKS_CONFIG_FILE", configFile)

	ctx, tt := cmdio.SetupTest(t.Context(), cmdio.TestOptions{PromptSupported: true})
	t.Cleanup(tt.Done)
	cmd := New(ctx)
	require.NoError(t, cmd.PersistentFlags().Set("profile", "skipws"))

	// Workspace path returns ErrAccountOnlyProfile. MustAnyClient must
	// recognize the type and fall through to the account client so
	// `auth describe` shows account info for account-only profiles.
	isAccount, err := MustAnyClient(cmd, []string{})
	require.NoError(t, err)
	require.True(t, isAccount, "expected fall-through to account client")
	require.NotNil(t, cmdctx.AccountClient(cmd.Context()))
}

func TestWorkspaceClientOrPromptReturnsSuccessWhenAuthSucceeds(t *testing.T) {
	testutil.CleanupEnvironment(t)
	t.Setenv("PATH", "")

	// If auth succeeds, trust the SDK's resolution regardless of HostType().
	// This supports unified hosts where HostType() returns AccountHost but
	// workspace APIs are available.
	cfg := &config.Config{
		Host:          "https://accounts.azuredatabricks.net/",
		AccountID:     "1234",
		Token:         "foobar",
		HTTPTransport: noNetworkTransport,
	}

	w, err := workspaceClientOrPrompt(t.Context(), cfg, false)
	assert.NotNil(t, w)
	assert.NoError(t, err)
}

func TestAccountClientOrPromptReturnsErrorForMissingAccountID(t *testing.T) {
	testutil.CleanupEnvironment(t)
	t.Setenv("PATH", "")

	cfg := &config.Config{
		Host:          "https://accounts.azuredatabricks.net/",
		Token:         "foobar",
		HTTPTransport: noNetworkTransport,
	}

	a, err := accountClientOrPrompt(t.Context(), cfg, false)
	assert.NotNil(t, a)
	assert.ErrorIs(t, err, databricks.ErrNotAccountClient)
}

func TestMustWorkspaceClientWithoutConfiguredDefaultFallsBackToDefaultSection(t *testing.T) {
	testutil.CleanupEnvironment(t)

	configFile := filepath.Join(t.TempDir(), ".databrickscfg")
	err := os.WriteFile(configFile, []byte(`
[DEFAULT]
host = https://default.cloud.databricks.com
token = default-token

[named-profile]
host = https://named.cloud.databricks.com
token = named-token
`), 0o600)
	require.NoError(t, err)

	t.Setenv("DATABRICKS_CONFIG_FILE", configFile)

	ctx := cmdio.MockDiscard(t.Context())
	ctx = SkipLoadBundle(ctx)
	cmd := New(ctx)

	err = MustWorkspaceClient(cmd, []string{})
	require.NoError(t, err)

	w := cmdctx.WorkspaceClient(cmd.Context())
	require.NotNil(t, w)
	// Pinned so the OAuth cache key matches what `databricks auth login` writes.
	assert.Equal(t, "DEFAULT", w.Config.Profile)
	assert.Equal(t, "https://default.cloud.databricks.com", w.Config.Host)
}
