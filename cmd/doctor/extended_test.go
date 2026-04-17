package doctor

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/env"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFirstLineVersion(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"git version 2.42.0\n", "2.42.0"},
		{"Python 3.11.5", "3.11.5"},
		{"uv 0.1.0 (unknown)", "0.1.0 (unknown)"},
		{"Terraform v1.5.0\non darwin", "v1.5.0"},
		{"", ""},
		{"weirdtool", "weirdtool"},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			assert.Equal(t, tt.want, firstLineVersion(tt.in))
		})
	}
}

func TestCheckToolchain(t *testing.T) {
	mockExec := func(versions map[string]string) execFunc {
		return func(_ context.Context, name string, _ ...string) (string, error) {
			v, ok := versions[name]
			if !ok {
				return "", errors.New("not found")
			}
			return v, nil
		}
	}

	t.Run("all tools present", func(t *testing.T) {
		run := mockExec(map[string]string{
			"git":       "git version 2.42.0",
			"python3":   "Python 3.11.5",
			"uv":        "uv 0.1.0",
			"terraform": "Terraform v1.5.0",
		})
		result := checkToolchain(t.Context(), run)
		assert.Equal(t, statusInfo, result.Status)
		assert.Contains(t, result.Message, "git 2.42.0")
		assert.Contains(t, result.Message, "python 3.11.5")
		assert.Contains(t, result.Message, "terraform v1.5.0")
	})

	t.Run("missing tools reported inline", func(t *testing.T) {
		run := mockExec(map[string]string{"git": "git version 2.42.0"})
		result := checkToolchain(t.Context(), run)
		assert.Equal(t, statusInfo, result.Status)
		assert.Contains(t, result.Message, "git 2.42.0")
		assert.Contains(t, result.Message, "terraform not found")
		assert.Contains(t, result.Message, "python not found")
	})
}

func TestCheckProxy(t *testing.T) {
	t.Run("none configured", func(t *testing.T) {
		clearConfigEnv(t)
		for _, v := range proxyEnvVars {
			t.Setenv(v, "")
		}
		ctx := cmdio.MockDiscard(t.Context())
		result := checkProxy(ctx)
		assert.Equal(t, statusInfo, result.Status)
		assert.Contains(t, result.Message, "no proxy")
	})

	t.Run("reports set env vars", func(t *testing.T) {
		for _, v := range proxyEnvVars {
			t.Setenv(v, "")
		}
		ctx := cmdio.MockDiscard(t.Context())
		ctx = env.Set(ctx, "HTTPS_PROXY", "http://proxy.example.com:8080")
		ctx = env.Set(ctx, "NO_PROXY", "localhost,127.0.0.1")

		result := checkProxy(ctx)
		assert.Equal(t, statusInfo, result.Status)
		assert.Contains(t, result.Message, "HTTPS_PROXY=http://proxy.example.com:8080")
		assert.Contains(t, result.Message, "NO_PROXY=localhost,127.0.0.1")
	})

	t.Run("masks credentials in proxy URL", func(t *testing.T) {
		for _, v := range proxyEnvVars {
			t.Setenv(v, "")
		}
		ctx := cmdio.MockDiscard(t.Context())
		ctx = env.Set(ctx, "HTTPS_PROXY", "http://user:secret@proxy.example.com:8080")
		result := checkProxy(ctx)
		assert.Contains(t, result.Message, "user:***@proxy.example.com")
		assert.NotContains(t, result.Message, "secret")
	})

	t.Run("deduplicates upper and lower case variants", func(t *testing.T) {
		for _, v := range proxyEnvVars {
			t.Setenv(v, "")
		}
		ctx := cmdio.MockDiscard(t.Context())
		ctx = env.Set(ctx, "HTTPS_PROXY", "http://a.example.com")
		ctx = env.Set(ctx, "https_proxy", "http://b.example.com")

		result := checkProxy(ctx)
		assert.Equal(t, 1, strings.Count(result.Message, "HTTPS_PROXY="))
	})
}

func TestMaskProxyValue(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"http://proxy.example.com:8080", "http://proxy.example.com:8080"},
		{"http://user:secret@proxy.example.com:8080", "http://user:***@proxy.example.com:8080"},
		{"not a url", "not a url"},
		{"localhost,127.0.0.1", "localhost,127.0.0.1"},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			assert.Equal(t, tt.want, maskProxyValue(tt.in))
		})
	}
}

func TestCheckUpdates(t *testing.T) {
	t.Run("dev build", func(t *testing.T) {
		// current version is "0.0.0-dev+<sha>" in tests
		result := checkUpdates(t.Context(), http.DefaultClient, "http://unused")
		assert.Equal(t, statusInfo, result.Status)
		assert.Contains(t, result.Message, "development build")
	})

	t.Run("up to date", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = io.WriteString(w, `{"tag_name": "v1.0.0"}`)
		}))
		defer srv.Close()

		result := checkUpdatesWithVersion(t.Context(), http.DefaultClient, srv.URL, "1.0.0")
		assert.Equal(t, statusPass, result.Status)
		assert.Contains(t, result.Message, "up to date")
	})

	t.Run("newer available", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = io.WriteString(w, `{"tag_name": "v1.2.3"}`)
		}))
		defer srv.Close()

		result := checkUpdatesWithVersion(t.Context(), http.DefaultClient, srv.URL, "1.0.0")
		assert.Equal(t, statusWarn, result.Status)
		assert.Contains(t, result.Message, "1.2.3")
		assert.Contains(t, result.Message, "1.0.0")
	})

	t.Run("network failure warns", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer srv.Close()

		result := checkUpdatesWithVersion(t.Context(), http.DefaultClient, srv.URL, "1.0.0")
		assert.Equal(t, statusWarn, result.Status)
		assert.Contains(t, result.Message, "HTTP 500")
	})

	t.Run("malformed response warns", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = io.WriteString(w, `not-json`)
		}))
		defer srv.Close()

		result := checkUpdatesWithVersion(t.Context(), http.DefaultClient, srv.URL, "1.0.0")
		assert.Equal(t, statusWarn, result.Status)
	})
}

func TestCheckLogFile(t *testing.T) {
	t.Run("not configured", func(t *testing.T) {
		t.Setenv("DATABRICKS_LOG_FILE", "")
		ctx := cmdio.MockDiscard(t.Context())
		result := checkLogFile(ctx)
		assert.Equal(t, statusInfo, result.Status)
		assert.Contains(t, result.Message, "not configured")
	})

	t.Run("configured path", func(t *testing.T) {
		t.Setenv("DATABRICKS_LOG_FILE", "")
		ctx := cmdio.MockDiscard(t.Context())
		ctx = env.Set(ctx, "DATABRICKS_LOG_FILE", "/var/log/databricks.log")
		result := checkLogFile(ctx)
		assert.Equal(t, statusInfo, result.Status)
		assert.Equal(t, "/var/log/databricks.log", result.Message)
	})
}

// Extra sanity: the full command includes all new checks.
func TestRunChecksIncludesExtended(t *testing.T) {
	clearConfigEnv(t)
	ctx := cmdio.MockDiscard(t.Context())
	results := runChecks(ctx, "", false)

	names := map[string]bool{}
	for _, r := range results {
		names[r.Name] = true
	}
	for _, want := range []string{"CLI Version", "Updates", "Toolchain", "Proxy/TLS", "Log File", "Config File", "Current Profile", "Authentication", "Identity", "Network"} {
		assert.True(t, names[want], "expected check %q in results", want)
	}
}

// require the test server to return JSON (sanity).
func TestUpdatesServerShape(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"tag_name": "v0.1.0"}`)
	}))
	defer srv.Close()

	resp, err := http.Get(srv.URL)
	require.NoError(t, err)
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(body), "tag_name")
}
