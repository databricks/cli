package doctor

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/databrickscfg/profile"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/config"
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

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
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

func TestCheckConfigFile(t *testing.T) {
	tests := []struct {
		name       string
		profiler   profile.Profiler
		wantStatus status
		wantMsg    string
	}{
		{
			name: "pass with profiles",
			profiler: &mockProfiler{
				path: "/home/user/.databrickscfg",
				profiles: profile.Profiles{
					{Name: "default", Host: "https://example.com"},
					{Name: "staging", Host: "https://staging.example.com"},
				},
			},
			wantStatus: statusPass,
			wantMsg:    "2 profiles",
		},
		{
			name:       "warn when config file absent",
			profiler:   &noConfigProfiler{path: "/home/user/.databrickscfg"},
			wantStatus: statusWarn,
			wantMsg:    "environment variables",
		},
		{
			name:       "fail when profiler errors",
			profiler:   &mockProfiler{path: "/home/user/.databrickscfg", err: errors.New("boom")},
			wantStatus: statusFail,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := profile.WithProfiler(cmdio.MockDiscard(t.Context()), tt.profiler)
			result := checkConfigFile(ctx)
			assert.Equal(t, "Config File", result.Name)
			assert.Equal(t, tt.wantStatus, result.Status)
			if tt.wantMsg != "" {
				assert.Contains(t, result.Message, tt.wantMsg)
			}
		})
	}
}

func TestCheckCurrentProfile(t *testing.T) {
	tests := []struct {
		name     string
		profile  string
		fromFlag bool
		envValue string
		cfg      *config.Config
		wantMsg  string
	}{
		{
			name:    "default",
			wantMsg: "none (using environment or defaults)",
		},
		{
			name:     "from flag",
			profile:  "staging",
			fromFlag: true,
			wantMsg:  "staging",
		},
		{
			name:     "from env",
			envValue: "from-env",
			wantMsg:  "from-env (from DATABRICKS_CONFIG_PROFILE)",
		},
		{
			name:     "flag overrides env",
			profile:  "from-flag",
			fromFlag: true,
			envValue: "from-env",
			wantMsg:  "from-flag",
		},
		{
			name:    "resolved from config",
			cfg:     &config.Config{Profile: "default"},
			wantMsg: "default (resolved from config file)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearConfigEnv(t)
			ctx := cmdio.MockDiscard(t.Context())
			if tt.envValue != "" {
				ctx = env.Set(ctx, "DATABRICKS_CONFIG_PROFILE", tt.envValue)
			}
			result := checkCurrentProfile(ctx, tt.profile, tt.fromFlag, tt.cfg)
			assert.Equal(t, "Current Profile", result.Name)
			assert.Equal(t, statusInfo, result.Status)
			assert.Equal(t, tt.wantMsg, result.Message)
		})
	}
}

func TestCheckAuth(t *testing.T) {
	okServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer okServer.Close()

	tests := []struct {
		name        string
		env         map[string]string
		wantStatus  status
		wantMsgPart string
		wantAuthCfg bool
	}{
		{
			name: "pass with PAT",
			env: map[string]string{
				"DATABRICKS_HOST":  okServer.URL,
				"DATABRICKS_TOKEN": "test-token",
			},
			wantStatus:  statusPass,
			wantMsgPart: "OK",
			wantAuthCfg: true,
		},
		{
			name: "account level",
			env: map[string]string{
				"DATABRICKS_HOST":       "https://accounts.cloud.databricks.com",
				"DATABRICKS_ACCOUNT_ID": "test-account-123",
				"DATABRICKS_TOKEN":      "test-token",
			},
			wantStatus:  statusPass,
			wantMsgPart: "account-level",
			wantAuthCfg: true,
		},
		{
			name:       "fail without credentials",
			wantStatus: statusFail,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearConfigEnv(t)
			for k, v := range tt.env {
				t.Setenv(k, v)
			}

			ctx := cmdio.MockDiscard(t.Context())
			cfg, resolveErr := resolveConfig(ctx, "", false)
			result, authCfg := checkAuth(ctx, cfg, resolveErr)

			assert.Equal(t, "Authentication", result.Name)
			assert.Equal(t, tt.wantStatus, result.Status)
			if tt.wantMsgPart != "" {
				assert.Contains(t, result.Message, tt.wantMsgPart)
			}
			if tt.wantAuthCfg {
				assert.NotNil(t, authCfg)
			} else {
				assert.Nil(t, authCfg)
			}
		})
	}
}

func TestResolveConfig(t *testing.T) {
	tests := []struct {
		name     string
		env      map[string]string
		ctxEnv   map[string]string
		wantHost string
		wantTok  string
	}{
		{
			name: "process env vars",
			env: map[string]string{
				"DATABRICKS_HOST":  "https://env.example.com",
				"DATABRICKS_TOKEN": "env-token",
			},
			wantHost: "https://env.example.com",
			wantTok:  "env-token",
		},
		{
			name: "context-backed env vars",
			ctxEnv: map[string]string{
				"DATABRICKS_HOST":  "https://ctx-env.example.com",
				"DATABRICKS_TOKEN": "ctx-token",
			},
			wantHost: "https://ctx-env.example.com",
			wantTok:  "ctx-token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearConfigEnv(t)
			for k, v := range tt.env {
				t.Setenv(k, v)
			}
			ctx := cmdio.MockDiscard(t.Context())
			for k, v := range tt.ctxEnv {
				ctx = env.Set(ctx, k, v) //nolint:fatcontext // accumulating test env vars onto context
			}

			cfg, err := resolveConfig(ctx, "", false)
			require.NoError(t, err)
			assert.Equal(t, tt.wantHost, cfg.Host)
			assert.Equal(t, tt.wantTok, cfg.Token)
		})
	}
}

func TestCheckIdentity(t *testing.T) {
	workspaceOK := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/scim/v2/Me") {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"userName": "test@example.com"}`))
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer workspaceOK.Close()

	accountOK := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/accounts/") && strings.HasSuffix(r.URL.Path, "/workspaces") {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[{"workspace_id": 1}, {"workspace_id": 2}]`))
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer accountOK.Close()

	unauthorized := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer unauthorized.Close()

	tests := []struct {
		name       string
		cfg        *config.Config
		wantStatus status
		wantMsg    string
	}{
		{
			name:       "workspace success",
			cfg:        &config.Config{Host: workspaceOK.URL, Token: "test-token"},
			wantStatus: statusPass,
			wantMsg:    "test@example.com",
		},
		{
			name:       "workspace failure",
			cfg:        &config.Config{Host: unauthorized.URL, Token: "bad-token"},
			wantStatus: statusFail,
		},
		{
			name: "account success (unified host)",
			cfg: &config.Config{
				Host:                       accountOK.URL,
				AccountID:                  "test-account-123",
				Token:                      "test-token",
				Experimental_IsUnifiedHost: true,
			},
			wantStatus: statusPass,
			wantMsg:    "account test-account-123 (2 workspaces)",
		},
		{
			name: "account failure (unified host)",
			cfg: &config.Config{
				Host:                       unauthorized.URL,
				AccountID:                  "test-account-123",
				Token:                      "bad-token",
				Experimental_IsUnifiedHost: true,
			},
			wantStatus: statusFail,
			wantMsg:    "Cannot list account workspaces",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := cmdio.MockDiscard(t.Context())
			result := checkIdentity(ctx, tt.cfg)
			assert.Equal(t, "Identity", result.Name)
			assert.Equal(t, tt.wantStatus, result.Status)
			if tt.wantMsg != "" {
				assert.Contains(t, result.Message, tt.wantMsg)
			}
		})
	}
}

func TestIsAccountLevelConfig(t *testing.T) {
	tests := []struct {
		name string
		cfg  *config.Config
		want bool
	}{
		{
			name: "classic account host",
			cfg: &config.Config{
				Host:      "https://accounts.cloud.databricks.com",
				AccountID: "test-account-123",
			},
			want: true,
		},
		{
			name: "unified host with account ID",
			cfg: &config.Config{
				Host:                       "https://myhost.databricks.com",
				AccountID:                  "test-account-123",
				Experimental_IsUnifiedHost: true,
			},
			want: true,
		},
		{
			name: "unified host with account and workspace ID is workspace-level",
			cfg: &config.Config{
				Host:                       "https://myhost.databricks.com",
				AccountID:                  "test-account-123",
				WorkspaceID:                "12345",
				Experimental_IsUnifiedHost: true,
			},
			want: false,
		},
		{
			name: "unified host without account ID is workspace",
			cfg: &config.Config{
				Host:                       "https://myhost.databricks.com",
				Experimental_IsUnifiedHost: true,
			},
			want: false,
		},
		{
			name: "workspace host",
			cfg:  &config.Config{Host: "https://myhost.databricks.com"},
			want: false,
		},
		{
			name: "no host",
			cfg:  &config.Config{AccountID: "test-account-123"},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, isAccountLevelConfig(tt.cfg))
		})
	}
}

func TestCheckNetwork(t *testing.T) {
	okServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer okServer.Close()

	t.Run("reachable via explicit host", func(t *testing.T) {
		ctx := cmdio.MockDiscard(t.Context())
		result := checkNetworkWithHost(ctx, okServer.URL, http.DefaultClient)
		assert.Equal(t, statusPass, result.Status)
		assert.Contains(t, result.Message, "reachable")
	})

	t.Run("no host fails", func(t *testing.T) {
		ctx := cmdio.MockDiscard(t.Context())
		result := checkNetworkWithHost(ctx, "", http.DefaultClient)
		assert.Equal(t, statusFail, result.Status)
		assert.Contains(t, result.Message, "No host configured")
	})

	t.Run("uses auth config transport", func(t *testing.T) {
		cfg := &config.Config{
			Host: "https://example.com",
			HTTPTransport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				assert.Equal(t, http.MethodHead, r.Method)
				assert.Equal(t, "https://example.com", r.URL.String())
				return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody, Header: http.Header{}, Request: r}, nil
			}),
		}
		ctx := cmdio.MockDiscard(t.Context())
		result := checkNetwork(ctx, cfg, nil, cfg)
		assert.Equal(t, statusPass, result.Status)
	})

	t.Run("fallback uses config transport", func(t *testing.T) {
		called := false
		cfg := &config.Config{
			Host: "https://example.com",
			HTTPTransport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				called = true
				return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody, Header: http.Header{}, Request: r}, nil
			}),
		}
		ctx := cmdio.MockDiscard(t.Context())
		result := checkNetwork(ctx, cfg, nil, nil)
		assert.True(t, called)
		assert.Equal(t, statusPass, result.Status)
	})

	t.Run("resolution failure without host", func(t *testing.T) {
		clearConfigEnv(t)
		configFile := filepath.Join(t.TempDir(), ".databrickscfg")
		require.NoError(t, os.WriteFile(configFile, []byte("[DEFAULT]\nhost = https://example.com\n"), 0o600))
		t.Setenv("DATABRICKS_CONFIG_FILE", configFile)
		t.Setenv("DATABRICKS_CONFIG_PROFILE", "missing")

		ctx := cmdio.MockDiscard(t.Context())
		cfg, resolveErr := resolveConfig(ctx, "", false)
		require.Error(t, resolveErr)

		result := checkNetwork(ctx, cfg, resolveErr, nil)
		assert.Equal(t, statusFail, result.Status)
		assert.Equal(t, "No host configured", result.Message)
	})

	t.Run("resolution failure with host still checks", func(t *testing.T) {
		cfg := &config.Config{Host: okServer.URL}
		ctx := cmdio.MockDiscard(t.Context())
		result := checkNetwork(ctx, cfg, errors.New("validate: missing credentials"), nil)
		assert.Equal(t, statusPass, result.Status)
		assert.Contains(t, result.Message, "reachable")
	})
}

func TestRender(t *testing.T) {
	results := []CheckResult{
		{Name: "Test", Status: statusPass, Message: "all good"},
		{Name: "Broken", Status: statusFail, Message: "oops", Detail: "details here"},
		{Name: "Version", Status: statusInfo, Message: "1.0.0"},
		{Name: "Config", Status: statusWarn, Message: "not found"},
	}

	t.Run("text", func(t *testing.T) {
		var buf bytes.Buffer
		require.NoError(t, render(&buf, results, flags.OutputText))
		out := buf.String()
		assert.Contains(t, out, "Test")
		assert.Contains(t, out, "all good")
		assert.Contains(t, out, "oops")
		assert.Contains(t, out, "details here")
	})

	t.Run("json round-trips and ends with newline", func(t *testing.T) {
		var buf bytes.Buffer
		require.NoError(t, render(&buf, results, flags.OutputJSON))
		assert.Equal(t, byte('\n'), buf.Bytes()[buf.Len()-1])

		var parsed DoctorReport
		require.NoError(t, json.Unmarshal(buf.Bytes(), &parsed))
		assert.Len(t, parsed.Results, 4)
		assert.Equal(t, "Test", parsed.Results[0].Name)
		assert.Equal(t, "details here", parsed.Results[1].Detail)
	})

	t.Run("json omits empty detail", func(t *testing.T) {
		var buf bytes.Buffer
		require.NoError(t, render(&buf, []CheckResult{{Name: "Test", Status: statusPass, Message: "ok"}}, flags.OutputJSON))
		assert.NotContains(t, buf.String(), "detail")
	})
}

func TestHasFailedChecks(t *testing.T) {
	tests := []struct {
		name    string
		results []CheckResult
		want    bool
	}{
		{
			name: "no failures",
			results: []CheckResult{
				{Status: statusPass}, {Status: statusInfo}, {Status: statusWarn}, {Status: statusSkip},
			},
			want: false,
		},
		{
			name:    "has failure",
			results: []CheckResult{{Status: statusPass}, {Status: statusFail}},
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, hasFailedChecks(tt.results))
		})
	}
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

	cmd := NewDoctorCmd()
	cmd.SetContext(ctx)

	outputFlag := flags.OutputText
	cmd.PersistentFlags().VarP(&outputFlag, "output", "o", "output type: text or json")

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"--output", "json"})

	err := cmd.Execute()
	require.ErrorContains(t, err, "one or more checks failed")
	assert.Equal(t, byte('\n'), buf.Bytes()[buf.Len()-1])

	var report DoctorReport
	require.NoError(t, json.Unmarshal(buf.Bytes(), &report))
	assert.GreaterOrEqual(t, len(report.Results), 4)
	assert.Equal(t, "CLI Version", report.Results[0].Name)
	assert.Equal(t, statusInfo, report.Results[0].Status)
}
