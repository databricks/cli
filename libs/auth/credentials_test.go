package auth

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"testing"

	"github.com/databricks/databricks-sdk-go/config"
	"github.com/databricks/databricks-sdk-go/config/experimental/auth"
	"github.com/databricks/databricks-sdk-go/credentials/u2m"
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
				Host:                       "https://myhost.com",
				AccountID:                  "acc-123",
				WorkspaceID:                "ws-456",
				Profile:                    "my-profile",
				Experimental_IsUnifiedHost: true,
			},
			want: AuthArguments{
				Host:          "https://myhost.com",
				AccountID:     "acc-123",
				WorkspaceID:   "ws-456",
				Profile:       "my-profile",
				IsUnifiedHost: true,
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
			ctx := context.Background()
			c := CLICredentials{persistentAuthFn: tt.persistentAuthFn}

			got, err := c.Configure(ctx, tt.cfg)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("want error %v, got %v", tt.wantErr, err)
			}
			if tt.wantErr != nil {
				return
			}

			// Verify the credentials provider sets the correct Bearer token.
			req, err := http.NewRequest("GET", tt.cfg.Host, nil)
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
