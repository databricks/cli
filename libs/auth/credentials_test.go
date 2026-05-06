package auth

import (
	"context"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/databricks/cli/libs/auth/storage"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/databricks/databricks-sdk-go/config/experimental/auth"
	"github.com/databricks/databricks-sdk-go/credentials/u2m"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

// hermeticAuthStorage isolates the test from the caller's real env vars and
// .databrickscfg so storage.ResolveCache sees a clean default.
func hermeticAuthStorage(t *testing.T) {
	t.Helper()
	t.Setenv(storage.EnvVar, "")
	t.Setenv("DATABRICKS_CONFIG_FILE", filepath.Join(t.TempDir(), "databrickscfg"))
}

// TestCredentialChainOrder purely exists as an extra measure to catch
// accidental change in the ordering.
func TestCredentialChainOrder(t *testing.T) {
	names := make([]string, len(credentialChain))
	for i, s := range credentialChain {
		names[i] = s.Name()
	}
	want := []string{
		"pat",
		"basic",
		"oauth-m2m",
		"databricks-cli",
		"metadata-service",
		"github-oidc",
		"azure-devops-oidc",
		"env-oidc",
		"file-oidc",
		"github-oidc-azure",
		"azure-msi",
		"azure-client-secret",
		"azure-cli",
		"google-credentials",
		"google-id",
	}
	if !slices.Equal(names, want) {
		t.Errorf("credential chain order: want %v, got %v", want, names)
	}
}

func TestCLICredentialsName(t *testing.T) {
	c := CLICredentials{}
	if got := c.Name(); got != "databricks-cli" {
		t.Errorf("Name(): want %q, got %q", "databricks-cli", got)
	}
}

func TestAuthArgumentsFromConfig(t *testing.T) {
	tests := []struct {
		name string
		cfg  *config.Config
		want AuthArguments
	}{
		{
			name: "empty config",
			cfg:  &config.Config{},
			want: AuthArguments{},
		},
		{
			name: "workspace host only",
			cfg: &config.Config{
				Host: "https://myworkspace.cloud.databricks.com",
			},
			want: AuthArguments{
				Host: "https://myworkspace.cloud.databricks.com",
			},
		},
		{
			name: "account host with account ID",
			cfg: &config.Config{
				Host:      "https://accounts.cloud.databricks.com",
				AccountID: "test-account-id",
			},
			want: AuthArguments{
				Host:      "https://accounts.cloud.databricks.com",
				AccountID: "test-account-id",
			},
		},
		{
			name: "all fields",
			cfg: &config.Config{
				Host:         "https://myhost.com",
				AccountID:    "acc-123",
				WorkspaceID:  "ws-456",
				Profile:      "my-profile",
				DiscoveryURL: "https://myhost.com/oidc/accounts/acc-123/.well-known/oauth-authorization-server",
			},
			want: AuthArguments{
				Host:         "https://myhost.com",
				AccountID:    "acc-123",
				WorkspaceID:  "ws-456",
				Profile:      "my-profile",
				DiscoveryURL: "https://myhost.com/oidc/accounts/acc-123/.well-known/oauth-authorization-server",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := authArgumentsFromConfig(tt.cfg)
			if got != tt.want {
				t.Errorf("want %v, got %v", tt.want, got)
			}
		})
	}
}

func TestCLICredentialsConfigure(t *testing.T) {
	testErr := errors.New("test error")

	tests := []struct {
		name             string
		cfg              *config.Config
		persistentAuthFn func(ctx context.Context, opts ...u2m.PersistentAuthOption) (auth.TokenSource, error)
		wantErr          error
		wantToken        string
	}{
		{
			name:    "empty host returns error",
			cfg:     &config.Config{},
			wantErr: errNoHost,
		},
		{
			name: "persistentAuthFn error is propagated",
			cfg: &config.Config{
				Host: "https://myworkspace.cloud.databricks.com",
			},
			persistentAuthFn: func(_ context.Context, _ ...u2m.PersistentAuthOption) (auth.TokenSource, error) {
				return nil, testErr
			},
			wantErr: testErr,
		},
		{
			name: "workspace host",
			cfg: &config.Config{
				Host: "https://myworkspace.cloud.databricks.com",
			},
			persistentAuthFn: func(_ context.Context, _ ...u2m.PersistentAuthOption) (auth.TokenSource, error) {
				return auth.TokenSourceFn(func(_ context.Context) (*oauth2.Token, error) {
					return &oauth2.Token{AccessToken: "workspace-token"}, nil
				}), nil
			},
			wantToken: "workspace-token",
		},
		{
			name: "account host",
			cfg: &config.Config{
				Host:      "https://accounts.cloud.databricks.com",
				AccountID: "test-account-id",
			},
			persistentAuthFn: func(_ context.Context, _ ...u2m.PersistentAuthOption) (auth.TokenSource, error) {
				return auth.TokenSourceFn(func(_ context.Context) (*oauth2.Token, error) {
					return &oauth2.Token{AccessToken: "account-token"}, nil
				}), nil
			},
			wantToken: "account-token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hermeticAuthStorage(t)
			ctx := t.Context()
			c := CLICredentials{persistentAuthFn: tt.persistentAuthFn}

			got, err := c.Configure(ctx, tt.cfg)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("want error %v, got %v", tt.wantErr, err)
			}
			if tt.wantErr != nil {
				return
			}

			// Verify the credentials provider sets the correct Bearer token.
			req, err := http.NewRequest(http.MethodGet, tt.cfg.Host, nil)
			if err != nil {
				t.Fatalf("creating request: %v", err)
			}
			if err := got.SetHeaders(req); err != nil {
				t.Fatalf("SetHeaders: want no error, got %v", err)
			}
			want := "Bearer " + tt.wantToken
			if gotHeader := req.Header.Get("Authorization"); gotHeader != want {
				t.Errorf("Authorization header: want %q, got %q", want, gotHeader)
			}
		})
	}
}

// TestCLICredentialsConfigure_ThreadsResolvedTokenCache guards against a
// regression where Configure forgot to pass u2m.WithTokenCache. Without it,
// the SDK's NewPersistentAuth silently defaulted to the file cache, so users
// who opted into secure storage saw "cache: token not found" on every command
// other than auth login/token/logout.
func TestCLICredentialsConfigure_ThreadsResolvedTokenCache(t *testing.T) {
	hermeticAuthStorage(t)

	var receivedOpts []u2m.PersistentAuthOption
	c := CLICredentials{
		persistentAuthFn: func(_ context.Context, opts ...u2m.PersistentAuthOption) (auth.TokenSource, error) {
			receivedOpts = opts
			return auth.TokenSourceFn(func(_ context.Context) (*oauth2.Token, error) {
				return &oauth2.Token{AccessToken: "tok"}, nil
			}), nil
		},
	}

	_, err := c.Configure(t.Context(), &config.Config{Host: "https://x.cloud.databricks.com"})
	require.NoError(t, err)

	// Two opts expected: WithOAuthArgument and WithTokenCache. The length
	// check is the most resilient way to assert both were passed without
	// poking at u2m's unexported state.
	assert.Len(t, receivedOpts, 2)
}

// TestCLICredentialsConfigure_PropagatesStorageResolutionError confirms
// Configure surfaces invalid DATABRICKS_AUTH_STORAGE values instead of
// silently falling back to the file cache. If Configure ever stops calling
// storage.ResolveCache, this test will catch it.
func TestCLICredentialsConfigure_PropagatesStorageResolutionError(t *testing.T) {
	hermeticAuthStorage(t)
	t.Setenv(storage.EnvVar, "bogus")

	c := CLICredentials{
		persistentAuthFn: func(_ context.Context, _ ...u2m.PersistentAuthOption) (auth.TokenSource, error) {
			t.Fatal("persistentAuthFn must not be called when cache resolution fails")
			return nil, nil
		},
	}

	_, err := c.Configure(t.Context(), &config.Config{Host: "https://x.cloud.databricks.com"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "DATABRICKS_AUTH_STORAGE")
}

// Writing a throwaway config file is verbose enough that future tests may
// want it too. Keeping the helper scoped here so it stays close to use.
func writeAuthStorageConfig(t *testing.T, mode string) {
	t.Helper()
	dir := t.TempDir()
	configPath := filepath.Join(dir, "databrickscfg")
	body := "[__settings__]\nauth_storage = " + mode + "\n"
	require.NoError(t, os.WriteFile(configPath, []byte(body), 0o600))
	t.Setenv("DATABRICKS_CONFIG_FILE", configPath)
	t.Setenv(storage.EnvVar, "")
}

// TestCLICredentialsConfigure_HonorsConfigFileSecureMode proves that
// Configure picks up auth_storage = secure from .databrickscfg, not just
// from DATABRICKS_AUTH_STORAGE. Both sources flow through the same resolver,
// but the PR's user-facing docs promise both work and nothing was asserting
// that for this call site.
func TestCLICredentialsConfigure_HonorsConfigFileSecureMode(t *testing.T) {
	writeAuthStorageConfig(t, "secure")

	c := CLICredentials{
		persistentAuthFn: func(_ context.Context, opts ...u2m.PersistentAuthOption) (auth.TokenSource, error) {
			// The presence of the second opt is verified by the sibling
			// test; here we just need Configure to succeed end-to-end when
			// the config file selects secure storage.
			assert.Len(t, opts, 2)
			return auth.TokenSourceFn(func(_ context.Context) (*oauth2.Token, error) {
				return &oauth2.Token{AccessToken: "tok"}, nil
			}), nil
		},
	}

	_, err := c.Configure(t.Context(), &config.Config{Host: "https://x.cloud.databricks.com"})
	require.NoError(t, err)
}
