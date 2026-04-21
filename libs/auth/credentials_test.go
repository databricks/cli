package auth

import (
	"context"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/databricks/databricks-sdk-go/config"
	"github.com/databricks/databricks-sdk-go/config/experimental/auth"
	"github.com/databricks/databricks-sdk-go/credentials/u2m"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

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
	// Point config file at a nonexistent path so legacyUnifiedHostFromProfile
	// doesn't read the developer's real ~/.databrickscfg.
	t.Setenv("DATABRICKS_CONFIG_FILE", filepath.Join(t.TempDir(), "nonexistent.cfg"))

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
			got := authArgumentsFromConfig(t.Context(), tt.cfg)
			if got != tt.want {
				t.Errorf("want %v, got %v", tt.want, got)
			}
		})
	}
}

func TestLegacyUnifiedHostFromProfile(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, ".databrickscfg")
	require.NoError(t, os.WriteFile(cfgPath, []byte(`
[unified]
host = https://unified.example.com
account_id = acc-1
experimental_is_unified_host = true

[plain]
host = https://plain.example.com
`), 0o600))

	cases := []struct {
		name    string
		cfg     *config.Config
		envFile string
		want    bool
	}{
		{name: "no profile", cfg: &config.Config{ConfigFile: cfgPath}, want: false},
		{name: "unified profile", cfg: &config.Config{Profile: "unified", ConfigFile: cfgPath}, want: true},
		{name: "plain profile", cfg: &config.Config{Profile: "plain", ConfigFile: cfgPath}, want: false},
		{name: "missing profile", cfg: &config.Config{Profile: "nope", ConfigFile: cfgPath}, want: false},
		{name: "unreadable file", cfg: &config.Config{Profile: "unified", ConfigFile: filepath.Join(dir, "nope.cfg")}, want: false},
		{name: "picks up DATABRICKS_CONFIG_FILE env", cfg: &config.Config{Profile: "unified"}, envFile: cfgPath, want: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.envFile != "" {
				t.Setenv("DATABRICKS_CONFIG_FILE", tc.envFile)
			} else {
				t.Setenv("DATABRICKS_CONFIG_FILE", filepath.Join(dir, "nonexistent.cfg"))
			}
			assert.Equal(t, tc.want, legacyUnifiedHostFromProfile(t.Context(), tc.cfg))
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
