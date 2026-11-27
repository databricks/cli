package config

import (
	"io/fs"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/databricks/cli/bundle/env"
	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/cli/libs/databrickscfg"
	envlib "github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/httptransport"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/databricks/databricks-sdk-go/useragent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupWorkspaceTest(t *testing.T) string {
	testutil.CleanupEnvironment(t)

	home := t.TempDir()
	t.Setenv("HOME", home)
	if runtime.GOOS == "windows" {
		t.Setenv("USERPROFILE", home)
	}

	return home
}

func TestWorkspaceResolveProfileFromHost(t *testing.T) {
	// If only a workspace host is specified, try to find a profile that uses
	// the same workspace host (unambiguously).
	w := Workspace{
		Host: "https://abc.cloud.databricks.com",
	}

	t.Run("no config file", func(t *testing.T) {
		setupWorkspaceTest(t)
		_, err := w.Client(t.Context())
		assert.NoError(t, err)
	})

	t.Run("default config file", func(t *testing.T) {
		setupWorkspaceTest(t)

		// This works if there is a config file with a matching profile.
		err := databrickscfg.SaveToProfile(t.Context(), &config.Config{
			Profile: "default",
			Host:    "https://abc.cloud.databricks.com",
			Token:   "123",
		})
		require.NoError(t, err)

		client, err := w.Client(t.Context())
		assert.NoError(t, err)
		assert.Equal(t, "default", client.Config.Profile)
	})

	t.Run("custom config file", func(t *testing.T) {
		home := setupWorkspaceTest(t)

		// This works if there is a config file with a matching profile.
		err := databrickscfg.SaveToProfile(t.Context(), &config.Config{
			ConfigFile: filepath.Join(home, "customcfg"),
			Profile:    "custom",
			Host:       "https://abc.cloud.databricks.com",
			Token:      "123",
		})
		require.NoError(t, err)

		t.Setenv("DATABRICKS_CONFIG_FILE", filepath.Join(home, "customcfg"))
		client, err := w.Client(t.Context())
		assert.NoError(t, err)
		assert.Equal(t, "custom", client.Config.Profile)
	})
}

func TestWorkspaceNormalizeHostURL(t *testing.T) {
	t.Run("extracts workspace_id from query param", func(t *testing.T) {
		w := Workspace{
			Host: "https://spog.databricks.com/?o=12345",
		}
		w.NormalizeHostURL()
		assert.Equal(t, "https://spog.databricks.com", w.Host)
		assert.Equal(t, "12345", w.WorkspaceID)
	})

	t.Run("extracts both workspace_id and account_id", func(t *testing.T) {
		w := Workspace{
			Host: "https://spog.databricks.com/?o=605&a=abc123",
		}
		w.NormalizeHostURL()
		assert.Equal(t, "https://spog.databricks.com", w.Host)
		assert.Equal(t, "605", w.WorkspaceID)
		assert.Equal(t, "abc123", w.AccountID)
	})

	t.Run("explicit workspace_id takes precedence", func(t *testing.T) {
		w := Workspace{
			Host:        "https://spog.databricks.com/?o=999",
			WorkspaceID: "explicit",
		}
		w.NormalizeHostURL()
		assert.Equal(t, "https://spog.databricks.com", w.Host)
		assert.Equal(t, "explicit", w.WorkspaceID)
	})

	t.Run("explicit account_id takes precedence", func(t *testing.T) {
		w := Workspace{
			Host:      "https://spog.databricks.com/?a=from-url",
			AccountID: "explicit-account",
		}
		w.NormalizeHostURL()
		assert.Equal(t, "https://spog.databricks.com", w.Host)
		assert.Equal(t, "explicit-account", w.AccountID)
	})

	t.Run("no-op for host without query params", func(t *testing.T) {
		w := Workspace{
			Host: "https://normal.databricks.com",
		}
		w.NormalizeHostURL()
		assert.Equal(t, "https://normal.databricks.com", w.Host)
		assert.Empty(t, w.WorkspaceID)
	})
}

func TestWorkspaceClientNormalizesHostBeforeProfileResolution(t *testing.T) {
	// Regression test: Client() must normalize the host URL (strip ?o= and
	// populate WorkspaceID) before building the SDK config and resolving
	// profiles. This ensures workspace_id is available for disambiguation.
	setupWorkspaceTest(t)

	err := databrickscfg.SaveToProfile(t.Context(), &config.Config{
		Profile:     "ws1",
		Host:        "https://spog.databricks.com",
		Token:       "token1",
		WorkspaceID: "111",
	})
	require.NoError(t, err)

	err = databrickscfg.SaveToProfile(t.Context(), &config.Config{
		Profile:     "ws2",
		Host:        "https://spog.databricks.com",
		Token:       "token2",
		WorkspaceID: "222",
	})
	require.NoError(t, err)

	// Host with ?o= should be normalized and workspace_id used to disambiguate.
	w := Workspace{
		Host: "https://spog.databricks.com/?o=222",
	}
	client, err := w.Client(t.Context())
	require.NoError(t, err)
	assert.Equal(t, "ws2", client.Config.Profile)
}

func TestWorkspaceClientProfileOverridesAuthEnv(t *testing.T) {
	// An explicit profile must win over auth env vars (#5096).
	setupWorkspaceTest(t)

	err := databrickscfg.SaveToProfile(t.Context(), &config.Config{
		Profile: "tst",
		Host:    "https://tst.cloud.databricks.test",
		Token:   "tst-token",
	})
	require.NoError(t, err)

	// direnv-style auth env vars pointing at a different (dev) workspace.
	t.Setenv("DATABRICKS_HOST", "https://dev.cloud.databricks.test")
	t.Setenv("DATABRICKS_TOKEN", "dev-token")

	w := Workspace{Profile: "tst"}
	client, err := w.Client(t.Context())
	require.NoError(t, err)
	assert.Equal(t, "tst", client.Config.Profile)
	assert.Equal(t, "https://tst.cloud.databricks.test", client.Config.Host)
	assert.Equal(t, "tst-token", client.Config.Token)
}

func TestWorkspaceClientProfileFillsAuthFromEnv(t *testing.T) {
	// Host-only profile: the profile wins for the host, but env fills the token
	// it omits (#5096).
	setupWorkspaceTest(t)

	err := databrickscfg.SaveToProfile(t.Context(), &config.Config{
		Profile: "host-only",
		Host:    "https://tst.cloud.databricks.test",
	})
	require.NoError(t, err)

	t.Setenv("DATABRICKS_TOKEN", "env-token")

	w := Workspace{Profile: "host-only"}
	client, err := w.Client(t.Context())
	require.NoError(t, err)
	assert.Equal(t, "host-only", client.Config.Profile)
	assert.Equal(t, "https://tst.cloud.databricks.test", client.Config.Host)
	// The token is not in the profile, so it is filled from the environment.
	assert.Equal(t, "env-token", client.Config.Token)
}

func TestWorkspaceClientHostAndProfileOverridesAuthEnv(t *testing.T) {
	// Bundle pins both workspace.host and workspace.profile: the profile wins
	// for auth and the host check passes because they agree (#5096).
	setupWorkspaceTest(t)

	err := databrickscfg.SaveToProfile(t.Context(), &config.Config{
		Profile: "tst",
		Host:    "https://tst.cloud.databricks.test",
		Token:   "tst-token",
	})
	require.NoError(t, err)

	// direnv-style auth env vars pointing at a different (dev) workspace.
	t.Setenv("DATABRICKS_HOST", "https://dev.cloud.databricks.test")
	t.Setenv("DATABRICKS_TOKEN", "dev-token")

	w := Workspace{
		Host:    "https://tst.cloud.databricks.test",
		Profile: "tst",
	}
	client, err := w.Client(t.Context())
	require.NoError(t, err)
	assert.Equal(t, "tst", client.Config.Profile)
	assert.Equal(t, "https://tst.cloud.databricks.test", client.Config.Host)
	assert.Equal(t, "tst-token", client.Config.Token)
}

func TestWorkspaceConfigHTTPTimeout(t *testing.T) {
	for _, tc := range []struct {
		envVal string
		want   int
	}{
		{"", 90},
		{"5", 5},
		{"not-a-number", 90},
	} {
		t.Run(tc.envVal, func(t *testing.T) {
			if tc.envVal != "" {
				t.Setenv(env.HTTPTimeoutSecondsVariable, tc.envVal)
			}
			w := Workspace{}
			assert.Equal(t, tc.want, w.Config(t.Context()).HTTPTimeoutSeconds)
		})
	}
}

func TestWorkspaceVerifyProfileForHost(t *testing.T) {
	// If both a workspace host and a profile are specified,
	// verify that the host configured in the profile matches
	// the host configured in the bundle configuration.
	w := Workspace{
		Host:    "https://abc.cloud.databricks.com",
		Profile: "abc",
	}

	t.Run("no config file", func(t *testing.T) {
		setupWorkspaceTest(t)
		_, err := w.Client(t.Context())
		assert.ErrorIs(t, err, fs.ErrNotExist)
	})

	t.Run("default config file with match", func(t *testing.T) {
		setupWorkspaceTest(t)

		// This works if there is a config file with a matching profile.
		err := databrickscfg.SaveToProfile(t.Context(), &config.Config{
			Profile: "abc",
			Host:    "https://abc.cloud.databricks.com",
		})
		require.NoError(t, err)

		_, err = w.Client(t.Context())
		assert.NoError(t, err)
	})

	t.Run("default config file with match on the default port", func(t *testing.T) {
		setupWorkspaceTest(t)

		err := databrickscfg.SaveToProfile(t.Context(), &config.Config{
			Profile: "abc",
			Host:    "https://abc.test",
		})
		require.NoError(t, err)

		w := Workspace{
			Host:    "https://abc.test:443",
			Profile: "abc",
		}
		_, err = w.Client(t.Context())
		assert.NoError(t, err)
	})

	t.Run("default config file with mismatch", func(t *testing.T) {
		setupWorkspaceTest(t)

		// This works if there is a config file with a matching profile.
		err := databrickscfg.SaveToProfile(t.Context(), &config.Config{
			Profile: "abc",
			Host:    "https://def.cloud.databricks.com",
		})
		require.NoError(t, err)

		_, err = w.Client(t.Context())
		assert.ErrorContains(t, err, "doesn’t match the host configured in the bundle")
	})

	t.Run("custom config file with match", func(t *testing.T) {
		home := setupWorkspaceTest(t)

		// This works if there is a config file with a matching profile.
		err := databrickscfg.SaveToProfile(t.Context(), &config.Config{
			ConfigFile: filepath.Join(home, "customcfg"),
			Profile:    "abc",
			Host:       "https://abc.cloud.databricks.com",
		})
		require.NoError(t, err)

		t.Setenv("DATABRICKS_CONFIG_FILE", filepath.Join(home, "customcfg"))
		_, err = w.Client(t.Context())
		assert.NoError(t, err)
	})

	t.Run("custom config file with mismatch", func(t *testing.T) {
		home := setupWorkspaceTest(t)

		// This works if there is a config file with a matching profile.
		err := databrickscfg.SaveToProfile(t.Context(), &config.Config{
			ConfigFile: filepath.Join(home, "customcfg"),
			Profile:    "abc",
			Host:       "https://def.cloud.databricks.com",
		})
		require.NoError(t, err)

		t.Setenv("DATABRICKS_CONFIG_FILE", filepath.Join(home, "customcfg"))
		_, err = w.Client(t.Context())
		assert.ErrorContains(t, err, "doesn’t match the host configured in the bundle")
	})
}

func TestWorkspaceClientUsesHealthCheckedTransport(t *testing.T) {
	setupWorkspaceTest(t)

	err := databrickscfg.SaveToProfile(t.Context(), &config.Config{
		Profile: "secure",
		Host:    "https://secure.cloud.databricks.test",
		Token:   "token",
	})
	require.NoError(t, err)
	// skip_verify comes from the profile, so it is only known after the config is resolved.
	err = databrickscfg.SaveToProfile(t.Context(), &config.Config{
		Profile:            "insecure",
		Host:               "https://insecure.cloud.databricks.test",
		Token:              "token",
		InsecureSkipVerify: true,
	})
	require.NoError(t, err)

	w := Workspace{Profile: "secure"}
	client, err := w.Client(t.Context())
	require.NoError(t, err)
	assert.Same(t, httptransport.HealthChecked(false), client.Config.HTTPTransport)

	w = Workspace{Profile: "insecure"}
	client, err = w.Client(t.Context())
	require.NoError(t, err)
	assert.Same(t, httptransport.HealthChecked(true), client.Config.HTTPTransport)
}

func TestWorkspaceClientNotAnAgentDropsAgentHeader(t *testing.T) {
	setupWorkspaceTest(t)
	t.Setenv("CLAUDECODE", "1")
	t.Setenv("DATABRICKS_TOKEN", "token")
	useragent.ClearCache()
	t.Cleanup(useragent.ClearCache)

	cases := []struct {
		name    string
		noAgent bool
	}{
		{name: "variable set", noAgent: true},
		{name: "variable unset", noAgent: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			userAgents := make(chan string, 1)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// The SDK first asks for the host's well-known config; only the Me call is checked.
				if r.URL.Path != "/api/2.0/preview/scim/v2/Me" {
					w.WriteHeader(http.StatusNotFound)
					return
				}
				userAgents <- r.Header.Get("User-Agent")
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"id":"1","userName":"user@example.com"}`))
			}))
			t.Cleanup(server.Close)

			ctx := t.Context()
			if tc.noAgent {
				ctx = envlib.Set(ctx, "DATABRICKS_NOT_AN_AGENT", "1")
			}
			w := Workspace{Host: server.URL}
			client, err := w.Client(ctx)
			require.NoError(t, err)
			_, err = client.CurrentUser.Me(ctx, iam.MeRequest{})
			require.NoError(t, err)

			ua := <-userAgents
			if tc.noAgent {
				assert.NotContains(t, ua, "agent/")
			} else {
				assert.Contains(t, ua, "agent/claude-code")
			}
		})
	}
}
