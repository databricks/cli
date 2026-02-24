package auth

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/databrickscfg/profile"
	"github.com/databricks/databricks-sdk-go/credentials/u2m"
	"github.com/databricks/databricks-sdk-go/httpclient/fixtures"
	"github.com/stretchr/testify/assert"
	"golang.org/x/oauth2"
)

var refreshFailureTokenResponse = fixtures.HTTPFixture{
	MatchAny: true,
	Status:   401,
	Response: map[string]string{
		"error":             "invalid_request",
		"error_description": "Refresh token is invalid",
	},
}

var refreshFailureInvalidResponse = fixtures.HTTPFixture{
	MatchAny: true,
	Status:   200,
	Response: "Not json",
}

var refreshFailureOtherError = fixtures.HTTPFixture{
	MatchAny: true,
	Status:   401,
	Response: map[string]string{
		"error":             "other_error",
		"error_description": "Databricks is down",
	},
}

var refreshSuccessTokenResponse = fixtures.HTTPFixture{
	MatchAny: true,
	Status:   200,
	Response: map[string]string{
		"access_token": "new-access-token",
		"token_type":   "Bearer",
		"expires_in":   "3600",
	},
}

type MockApiClient struct{}

// GetAccountOAuthEndpoints implements u2m.OAuthEndpointSupplier.
func (m *MockApiClient) GetAccountOAuthEndpoints(ctx context.Context, accountHost, accountId string) (*u2m.OAuthAuthorizationServer, error) {
	return &u2m.OAuthAuthorizationServer{
		TokenEndpoint:         accountHost + "/token",
		AuthorizationEndpoint: accountHost + "/authorize",
	}, nil
}

// GetWorkspaceOAuthEndpoints implements u2m.OAuthEndpointSupplier.
func (m *MockApiClient) GetWorkspaceOAuthEndpoints(ctx context.Context, workspaceHost string) (*u2m.OAuthAuthorizationServer, error) {
	return &u2m.OAuthAuthorizationServer{
		TokenEndpoint:         workspaceHost + "/token",
		AuthorizationEndpoint: workspaceHost + "/authorize",
	}, nil
}

// GetUnifiedOAuthEndpoints implements u2m.OAuthEndpointSupplier.
func (m *MockApiClient) GetUnifiedOAuthEndpoints(ctx context.Context, host, accountId string) (*u2m.OAuthAuthorizationServer, error) {
	return &u2m.OAuthAuthorizationServer{
		TokenEndpoint:         host + "/token",
		AuthorizationEndpoint: host + "/authorize",
	}, nil
}

var _ u2m.OAuthEndpointSupplier = (*MockApiClient)(nil)

func TestToken_loadToken(t *testing.T) {
	profiler := profile.InMemoryProfiler{
		Profiles: profile.Profiles{
			{
				Name:      "expired",
				Host:      "https://accounts.cloud.databricks.com",
				AccountID: "expired",
			},
			{
				Name:      "active",
				Host:      "https://accounts.cloud.databricks.com",
				AccountID: "active",
			},
			{
				Name: "workspace-a",
				Host: "https://workspace-a.cloud.databricks.com",
			},
			{
				Name: "dup1",
				Host: "https://shared.cloud.databricks.com",
			},
			{
				Name: "dup2",
				Host: "https://shared.cloud.databricks.com",
			},
			{
				Name:      "acct-dup1",
				Host:      "https://accounts.cloud.databricks.com",
				AccountID: "same-account",
			},
			{
				Name:      "acct-dup2",
				Host:      "https://accounts.cloud.databricks.com",
				AccountID: "same-account",
			},
			{
				Name: "default.dev",
				Host: "https://dev.cloud.databricks.com",
			},
			{
				Name: "unique-ws",
				Host: "https://unique-ws.cloud.databricks.com",
			},
			{
				Name: "legacy-ws",
				Host: "https://legacy-ws.cloud.databricks.com",
			},
		},
	}
	tokenCache := &inMemoryTokenCache{
		Tokens: map[string]*oauth2.Token{
			"https://accounts.cloud.databricks.com/oidc/accounts/expired": {
				RefreshToken: "expired",
			},
			"https://accounts.cloud.databricks.com/oidc/accounts/active": {
				RefreshToken: "active",
				Expiry:       time.Now().Add(1 * time.Hour), // Hopefully unit tests don't take an hour to run
			},
			"expired": {
				RefreshToken: "expired",
			},
			"active": {
				RefreshToken: "active",
				Expiry:       time.Now().Add(1 * time.Hour),
			},
			"workspace-a": {
				RefreshToken: "workspace-a",
				Expiry:       time.Now().Add(1 * time.Hour),
			},
			"https://workspace-a.cloud.databricks.com": {
				RefreshToken: "workspace-a",
				Expiry:       time.Now().Add(1 * time.Hour),
			},
			"default.dev": {
				RefreshToken: "default.dev",
				Expiry:       time.Now().Add(1 * time.Hour),
			},
			"unique-ws": {
				RefreshToken: "unique-ws",
				Expiry:       time.Now().Add(1 * time.Hour),
			},
			"https://no-profile.cloud.databricks.com": {
				RefreshToken: "no-profile",
				Expiry:       time.Now().Add(1 * time.Hour),
			},
			"https://legacy-ws.cloud.databricks.com": {
				RefreshToken: "legacy-ws",
				Expiry:       time.Now().Add(1 * time.Hour),
			},
		},
	}
	validateToken := func(resp *oauth2.Token) {
		assert.Equal(t, "new-access-token", resp.AccessToken)
		assert.Equal(t, "Bearer", resp.TokenType)
	}

	cases := []struct {
		name          string
		ctx           context.Context
		args          loadTokenArgs
		validateToken func(*oauth2.Token)
		wantErr       string
	}{
		{
			name: "prints helpful login message on refresh failure when profile is specified",
			args: loadTokenArgs{
				authArguments: &auth.AuthArguments{},
				profileName:   "expired",
				args:          []string{},
				tokenTimeout:  1 * time.Hour,
				profiler:      profiler,
				persistentAuthOpts: []u2m.PersistentAuthOption{
					u2m.WithTokenCache(tokenCache),
					u2m.WithOAuthEndpointSupplier(&MockApiClient{}),
					u2m.WithHttpClient(&http.Client{Transport: fixtures.SliceTransport{refreshFailureTokenResponse}}),
				},
			},
			wantErr: `A new access token could not be retrieved because the refresh token is invalid. To reauthenticate, run the following command:
  $ databricks auth login --profile expired`,
		},
		{
			name: "prints helpful login message on refresh failure when host is specified",
			args: loadTokenArgs{
				authArguments: &auth.AuthArguments{
					Host:      "https://accounts.cloud.databricks.com",
					AccountID: "expired",
				},
				profileName:  "",
				args:         []string{},
				tokenTimeout: 1 * time.Hour,
				profiler:     profiler,
				persistentAuthOpts: []u2m.PersistentAuthOption{
					u2m.WithTokenCache(tokenCache),
					u2m.WithOAuthEndpointSupplier(&MockApiClient{}),
					u2m.WithHttpClient(&http.Client{Transport: fixtures.SliceTransport{refreshFailureTokenResponse}}),
				},
			},
			wantErr: `A new access token could not be retrieved because the refresh token is invalid. To reauthenticate, run the following command:
  $ databricks auth login --profile expired`,
		},
		{
			name: "prints helpful login message on invalid response",
			args: loadTokenArgs{
				authArguments: &auth.AuthArguments{},
				profileName:   "active",
				args:          []string{},
				tokenTimeout:  1 * time.Hour,
				profiler:      profiler,
				persistentAuthOpts: []u2m.PersistentAuthOption{
					u2m.WithTokenCache(tokenCache),
					u2m.WithOAuthEndpointSupplier(&MockApiClient{}),
					u2m.WithHttpClient(&http.Client{Transport: fixtures.SliceTransport{refreshFailureInvalidResponse}}),
				},
			},
			wantErr: "token refresh: oauth2: cannot parse json: invalid character 'N' looking for beginning of value. Try logging in again with " +
				"`databricks auth login --profile active` before retrying. If this fails, please report this issue to the Databricks CLI maintainers at https://github.com/databricks/cli/issues/new",
		},
		{
			name: "prints helpful login message on other error response",
			args: loadTokenArgs{
				authArguments: &auth.AuthArguments{},
				profileName:   "active",
				args:          []string{},
				tokenTimeout:  1 * time.Hour,
				profiler:      profiler,
				persistentAuthOpts: []u2m.PersistentAuthOption{
					u2m.WithTokenCache(tokenCache),
					u2m.WithOAuthEndpointSupplier(&MockApiClient{}),
					u2m.WithHttpClient(&http.Client{Transport: fixtures.SliceTransport{refreshFailureOtherError}}),
				},
			},
			wantErr: "token refresh: Databricks is down (error code: other_error). Try logging in again with " +
				"`databricks auth login --profile active` before retrying. If this fails, please report this issue to the Databricks CLI maintainers at https://github.com/databricks/cli/issues/new",
		},
		{
			name: "succeeds with profile",
			args: loadTokenArgs{
				authArguments: &auth.AuthArguments{},
				profileName:   "active",
				args:          []string{},
				tokenTimeout:  1 * time.Hour,
				profiler:      profiler,
				persistentAuthOpts: []u2m.PersistentAuthOption{
					u2m.WithTokenCache(tokenCache),
					u2m.WithOAuthEndpointSupplier(&MockApiClient{}),
					u2m.WithHttpClient(&http.Client{Transport: fixtures.SliceTransport{refreshSuccessTokenResponse}}),
				},
			},
			validateToken: validateToken,
		},
		{
			name: "succeeds with host",
			args: loadTokenArgs{
				authArguments: &auth.AuthArguments{Host: "https://accounts.cloud.databricks.com", AccountID: "active"},
				profileName:   "",
				args:          []string{},
				tokenTimeout:  1 * time.Hour,
				profiler:      profiler,
				persistentAuthOpts: []u2m.PersistentAuthOption{
					u2m.WithTokenCache(tokenCache),
					u2m.WithOAuthEndpointSupplier(&MockApiClient{}),
					u2m.WithHttpClient(&http.Client{Transport: fixtures.SliceTransport{refreshSuccessTokenResponse}}),
				},
			},
			validateToken: validateToken,
		},
		{
			name: "positional arg resolved as profile name",
			args: loadTokenArgs{
				authArguments: &auth.AuthArguments{},
				profileName:   "",
				args:          []string{"workspace-a"},
				tokenTimeout:  1 * time.Hour,
				profiler:      profiler,
				persistentAuthOpts: []u2m.PersistentAuthOption{
					u2m.WithTokenCache(tokenCache),
					u2m.WithOAuthEndpointSupplier(&MockApiClient{}),
					u2m.WithHttpClient(&http.Client{Transport: fixtures.SliceTransport{refreshSuccessTokenResponse}}),
				},
			},
			validateToken: validateToken,
		},
		{
			name: "positional arg with dot treated as host when no profile matches",
			args: loadTokenArgs{
				authArguments: &auth.AuthArguments{},
				profileName:   "",
				args:          []string{"workspace-a.cloud.databricks.com"},
				tokenTimeout:  1 * time.Hour,
				profiler:      profiler,
				persistentAuthOpts: []u2m.PersistentAuthOption{
					u2m.WithTokenCache(tokenCache),
					u2m.WithOAuthEndpointSupplier(&MockApiClient{}),
					u2m.WithHttpClient(&http.Client{Transport: fixtures.SliceTransport{refreshSuccessTokenResponse}}),
				},
			},
			validateToken: validateToken,
		},
		{
			name: "dotted profile name resolved as profile not host",
			args: loadTokenArgs{
				authArguments: &auth.AuthArguments{},
				profileName:   "",
				args:          []string{"default.dev"},
				tokenTimeout:  1 * time.Hour,
				profiler:      profiler,
				persistentAuthOpts: []u2m.PersistentAuthOption{
					u2m.WithTokenCache(tokenCache),
					u2m.WithOAuthEndpointSupplier(&MockApiClient{}),
					u2m.WithHttpClient(&http.Client{Transport: fixtures.SliceTransport{refreshSuccessTokenResponse}}),
				},
			},
			validateToken: validateToken,
		},
		{
			name: "positional arg not a profile falls through to host",
			args: loadTokenArgs{
				authArguments: &auth.AuthArguments{},
				profileName:   "",
				args:          []string{"nonexistent"},
				tokenTimeout:  1 * time.Hour,
				profiler:      profiler,
				persistentAuthOpts: []u2m.PersistentAuthOption{
					u2m.WithTokenCache(tokenCache),
					u2m.WithOAuthEndpointSupplier(&MockApiClient{}),
				},
			},
			wantErr: "cache: databricks OAuth is not configured for this host. " +
				"Try logging in again with `databricks auth login --host https://nonexistent` before retrying. " +
				"If this fails, please report this issue to the Databricks CLI maintainers at https://github.com/databricks/cli/issues/new",
		},
		{
			name: "scheme-less account host ambiguity detected correctly",
			ctx:  cmdio.MockDiscard(context.Background()),
			args: loadTokenArgs{
				authArguments: &auth.AuthArguments{
					Host:      "accounts.cloud.databricks.com",
					AccountID: "same-account",
				},
				profileName:  "",
				args:         []string{},
				tokenTimeout: 1 * time.Hour,
				profiler:     profiler,
				persistentAuthOpts: []u2m.PersistentAuthOption{
					u2m.WithTokenCache(tokenCache),
					u2m.WithOAuthEndpointSupplier(&MockApiClient{}),
				},
			},
			wantErr: "acct-dup1 and acct-dup2 match accounts.cloud.databricks.com in <in memory>. Use --profile to specify which profile to use",
		},
		{
			name: "workspace host ambiguity — multiple profiles, non-interactive",
			ctx:  cmdio.MockDiscard(context.Background()),
			args: loadTokenArgs{
				authArguments: &auth.AuthArguments{
					Host: "https://shared.cloud.databricks.com",
				},
				profileName:  "",
				args:         []string{},
				tokenTimeout: 1 * time.Hour,
				profiler:     profiler,
				persistentAuthOpts: []u2m.PersistentAuthOption{
					u2m.WithTokenCache(tokenCache),
					u2m.WithOAuthEndpointSupplier(&MockApiClient{}),
				},
			},
			wantErr: "dup1 and dup2 match https://shared.cloud.databricks.com in <in memory>. Use --profile to specify which profile to use",
		},
		{
			name: "account host — same host, different account IDs — no ambiguity",
			args: loadTokenArgs{
				authArguments: &auth.AuthArguments{
					Host:      "https://accounts.cloud.databricks.com",
					AccountID: "active",
				},
				profileName:  "",
				args:         []string{},
				tokenTimeout: 1 * time.Hour,
				profiler:     profiler,
				persistentAuthOpts: []u2m.PersistentAuthOption{
					u2m.WithTokenCache(tokenCache),
					u2m.WithOAuthEndpointSupplier(&MockApiClient{}),
					u2m.WithHttpClient(&http.Client{Transport: fixtures.SliceTransport{refreshSuccessTokenResponse}}),
				},
			},
			validateToken: validateToken,
		},
		{
			name: "account host — same host AND same account ID — ambiguity",
			ctx:  cmdio.MockDiscard(context.Background()),
			args: loadTokenArgs{
				authArguments: &auth.AuthArguments{
					Host:      "https://accounts.cloud.databricks.com",
					AccountID: "same-account",
				},
				profileName:  "",
				args:         []string{},
				tokenTimeout: 1 * time.Hour,
				profiler:     profiler,
				persistentAuthOpts: []u2m.PersistentAuthOption{
					u2m.WithTokenCache(tokenCache),
					u2m.WithOAuthEndpointSupplier(&MockApiClient{}),
				},
			},
			wantErr: "acct-dup1 and acct-dup2 match https://accounts.cloud.databricks.com in <in memory>. Use --profile to specify which profile to use",
		},
		{
			name: "host with one matching profile resolves to profile key",
			args: loadTokenArgs{
				authArguments: &auth.AuthArguments{
					Host: "https://unique-ws.cloud.databricks.com",
				},
				profileName:  "",
				args:         []string{},
				tokenTimeout: 1 * time.Hour,
				profiler:     profiler,
				persistentAuthOpts: []u2m.PersistentAuthOption{
					u2m.WithTokenCache(tokenCache),
					u2m.WithOAuthEndpointSupplier(&MockApiClient{}),
					u2m.WithHttpClient(&http.Client{Transport: fixtures.SliceTransport{refreshSuccessTokenResponse}}),
				},
			},
			validateToken: validateToken,
		},
		{
			name: "host with no matching profile uses host key",
			args: loadTokenArgs{
				authArguments: &auth.AuthArguments{
					Host: "https://no-profile.cloud.databricks.com",
				},
				profileName:  "",
				args:         []string{},
				tokenTimeout: 1 * time.Hour,
				profiler:     profiler,
				persistentAuthOpts: []u2m.PersistentAuthOption{
					u2m.WithTokenCache(tokenCache),
					u2m.WithOAuthEndpointSupplier(&MockApiClient{}),
					u2m.WithHttpClient(&http.Client{Transport: fixtures.SliceTransport{refreshSuccessTokenResponse}}),
				},
			},
			validateToken: validateToken,
		},
		{
			name: "host with one matching profile and host-key-only token found via SDK fallback",
			args: loadTokenArgs{
				authArguments: &auth.AuthArguments{
					Host: "https://legacy-ws.cloud.databricks.com",
				},
				profileName:  "",
				args:         []string{},
				tokenTimeout: 1 * time.Hour,
				profiler:     profiler,
				persistentAuthOpts: []u2m.PersistentAuthOption{
					u2m.WithTokenCache(tokenCache),
					u2m.WithOAuthEndpointSupplier(&MockApiClient{}),
					u2m.WithHttpClient(&http.Client{Transport: fixtures.SliceTransport{refreshSuccessTokenResponse}}),
				},
			},
			validateToken: validateToken,
		},
		{
			name: "profile flag + positional non-host arg still errors",
			args: loadTokenArgs{
				authArguments: &auth.AuthArguments{},
				profileName:   "active",
				args:          []string{"workspace-a"},
				tokenTimeout:  1 * time.Hour,
				profiler:      profiler,
				persistentAuthOpts: []u2m.PersistentAuthOption{
					u2m.WithTokenCache(tokenCache),
					u2m.WithOAuthEndpointSupplier(&MockApiClient{}),
				},
			},
			wantErr: "providing both a profile and host is not supported",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ctx := c.ctx
			if ctx == nil {
				ctx = context.Background()
			}
			got, err := loadToken(ctx, c.args)
			if c.wantErr != "" {
				assert.Equal(t, c.wantErr, err.Error())
			} else {
				assert.NoError(t, err)
				c.validateToken(got)
			}
		})
	}
}
