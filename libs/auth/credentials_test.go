package auth_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/databricks/databricks-sdk-go/credentials/u2m"
	"github.com/databricks/databricks-sdk-go/credentials/u2m/cache"
	"github.com/databricks/databricks-sdk-go/httpclient/fixtures"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

type inMemoryTokenCache struct {
	Tokens map[string]*oauth2.Token
}

func (c *inMemoryTokenCache) Lookup(key string) (*oauth2.Token, error) {
	t, ok := c.Tokens[key]
	if !ok {
		return nil, cache.ErrNotFound
	}
	return t, nil
}

func (c *inMemoryTokenCache) Store(key string, t *oauth2.Token) error {
	c.Tokens[key] = t
	return nil
}

var _ cache.TokenCache = (*inMemoryTokenCache)(nil)

type mockEndpointSupplier struct{}

func (m *mockEndpointSupplier) GetAccountOAuthEndpoints(_ context.Context, accountHost, _ string) (*u2m.OAuthAuthorizationServer, error) {
	return &u2m.OAuthAuthorizationServer{
		TokenEndpoint:         accountHost + "/token",
		AuthorizationEndpoint: accountHost + "/authorize",
	}, nil
}

func (m *mockEndpointSupplier) GetWorkspaceOAuthEndpoints(_ context.Context, workspaceHost string) (*u2m.OAuthAuthorizationServer, error) {
	return &u2m.OAuthAuthorizationServer{
		TokenEndpoint:         workspaceHost + "/token",
		AuthorizationEndpoint: workspaceHost + "/authorize",
	}, nil
}

func (m *mockEndpointSupplier) GetUnifiedOAuthEndpoints(_ context.Context, host, _ string) (*u2m.OAuthAuthorizationServer, error) {
	return &u2m.OAuthAuthorizationServer{
		TokenEndpoint:         host + "/token",
		AuthorizationEndpoint: host + "/authorize",
	}, nil
}

var _ u2m.OAuthEndpointSupplier = (*mockEndpointSupplier)(nil)

func TestCLICredentialsName(t *testing.T) {
	c := auth.CLICredentials{}
	assert.Equal(t, "databricks-cli", c.Name())
}

func TestCLICredentialsConfigure(t *testing.T) {
	refreshSuccess := fixtures.HTTPFixture{
		MatchAny: true,
		Status:   200,
		Response: map[string]string{
			"access_token": "refreshed-access-token",
			"token_type":   "Bearer",
			"expires_in":   "3600",
		},
	}
	refreshFailure := fixtures.HTTPFixture{
		MatchAny: true,
		Status:   401,
		Response: map[string]string{
			"error":             "invalid_request",
			"error_description": "Refresh token is invalid",
		},
	}

	tests := []struct {
		name    string
		cfg     *config.Config
		cache   map[string]*oauth2.Token
		http    []fixtures.HTTPFixture
		wantNil bool
		wantErr string
	}{
		{
			name:    "empty host returns error",
			cfg:     &config.Config{},
			wantErr: "no host provided",
		},
		{
			name: "workspace host with valid token",
			cfg: &config.Config{
				Host: "https://myworkspace.cloud.databricks.com",
			},
			cache: map[string]*oauth2.Token{
				"https://myworkspace.cloud.databricks.com": {
					AccessToken: "valid-token",
					Expiry:      time.Now().Add(time.Hour),
				},
			},
		},
		{
			name: "account host with valid token",
			cfg: &config.Config{
				Host:      "https://accounts.cloud.databricks.com",
				AccountID: "test-account-id",
			},
			cache: map[string]*oauth2.Token{
				"https://accounts.cloud.databricks.com/oidc/accounts/test-account-id": {
					AccessToken: "valid-account-token",
					Expiry:      time.Now().Add(time.Hour),
				},
			},
		},
		{
			name: "no cached token",
			cfg: &config.Config{
				Host: "https://myworkspace.cloud.databricks.com",
			},
			cache: map[string]*oauth2.Token{},
		},
		{
			name: "expired token with successful refresh",
			cfg: &config.Config{
				Host: "https://myworkspace.cloud.databricks.com",
			},
			cache: map[string]*oauth2.Token{
				"https://myworkspace.cloud.databricks.com": {
					RefreshToken: "valid-refresh",
					Expiry:       time.Now().Add(-time.Hour),
				},
			},
			http: []fixtures.HTTPFixture{refreshSuccess},
		},
		{
			name: "expired token with failed refresh",
			cfg: &config.Config{
				Host: "https://myworkspace.cloud.databricks.com",
			},
			cache: map[string]*oauth2.Token{
				"https://myworkspace.cloud.databricks.com": {
					RefreshToken: "bad-refresh",
					Expiry:       time.Now().Add(-time.Hour),
				},
			},
			http: []fixtures.HTTPFixture{refreshFailure},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			// Pre-populate the cache using the real cache keys when a cache
			// map is provided. Test cases use hard-coded keys for readability;
			// we re-key them here if the config produces a valid OAuthArgument.
			tokenCache := &inMemoryTokenCache{Tokens: make(map[string]*oauth2.Token)}
			if tt.cache != nil && tt.cfg.Host != "" {
				key, err := func() (string, error) {
					arg, err := auth.AuthArguments{
						Host:      tt.cfg.Host,
						AccountID: tt.cfg.AccountID,
					}.ToOAuthArgument()
					if err != nil {
						return "", err
					}
					return arg.GetCacheKey(), nil
				}()
				if err == nil {
					for _, tok := range tt.cache {
						tokenCache.Tokens[key] = tok
						break // only one token per test case
					}
				}
			}

			opts := []u2m.PersistentAuthOption{
				u2m.WithTokenCache(tokenCache),
				u2m.WithOAuthEndpointSupplier(&mockEndpointSupplier{}),
			}
			if len(tt.http) > 0 {
				transport := fixtures.SliceTransport(tt.http)
				opts = append(opts, u2m.WithHttpClient(&http.Client{Transport: transport}))
			}

			c := auth.CLICredentials{PersistentAuthOptions: opts}
			cp, err := c.Configure(ctx, tt.cfg)

			if tt.wantErr != "" {
				assert.ErrorContains(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			if tt.wantNil {
				assert.Nil(t, cp)
				return
			}
			require.NotNil(t, cp)
		})
	}
}
