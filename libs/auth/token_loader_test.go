package auth

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/databricks/databricks-sdk-go/credentials/u2m"
	"github.com/databricks/databricks-sdk-go/credentials/u2m/cache"
	"github.com/databricks/databricks-sdk-go/httpclient/fixtures"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

type fakePersistentAuth struct {
	token    *oauth2.Token
	tokenErr error
	closeErr error
}

func (f *fakePersistentAuth) Token() (*oauth2.Token, error) {
	return f.token, f.tokenErr
}

func (f *fakePersistentAuth) Close() error {
	return f.closeErr
}

func TestAcquireTokenDoesNotMutatePersistentAuthOpts(t *testing.T) {
	defaultFactory := persistentAuthFactory
	t.Cleanup(func() {
		persistentAuthFactory = defaultFactory
	})

	opts := []u2m.PersistentAuthOption{
		func(pa *u2m.PersistentAuth) {},
		func(pa *u2m.PersistentAuth) {},
	}
	initialLen := len(opts)
	factoryCalls := 0

	persistentAuthFactory = func(ctx context.Context, providedOpts ...u2m.PersistentAuthOption) (persistentAuth, error) {
		factoryCalls++
		require.Len(t, providedOpts, initialLen+1)
		require.Len(t, opts, initialLen) // original slice must not change
		return &fakePersistentAuth{
			token: &oauth2.Token{
				AccessToken: "token",
			},
		}, nil
	}

	req := AcquireTokenRequest{
		AuthArguments: &AuthArguments{
			Host:      "https://accounts.cloud.databricks.com",
			AccountID: "active",
		},
		PersistentAuthOpts: opts,
	}

	_, err := AcquireToken(context.Background(), req)
	require.NoError(t, err)
	require.Len(t, opts, initialLen)

	_, err = AcquireToken(context.Background(), req)
	require.NoError(t, err)
	require.Len(t, opts, initialLen)
	require.Equal(t, 2, factoryCalls)
}

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

type mockOAuthEndpointSupplier struct{}

func (m *mockOAuthEndpointSupplier) GetAccountOAuthEndpoints(ctx context.Context, accountHost, accountId string) (*u2m.OAuthAuthorizationServer, error) {
	return &u2m.OAuthAuthorizationServer{
		TokenEndpoint:         accountHost + "/token",
		AuthorizationEndpoint: accountHost + "/authorize",
	}, nil
}

func (m *mockOAuthEndpointSupplier) GetWorkspaceOAuthEndpoints(ctx context.Context, workspaceHost string) (*u2m.OAuthAuthorizationServer, error) {
	return &u2m.OAuthAuthorizationServer{
		TokenEndpoint:         workspaceHost + "/token",
		AuthorizationEndpoint: workspaceHost + "/authorize",
	}, nil
}

var _ u2m.OAuthEndpointSupplier = (*mockOAuthEndpointSupplier)(nil)

func TestAcquireTokenRefreshFailureWithProfileShowsLoginHint(t *testing.T) {
	_, err := AcquireToken(context.Background(), AcquireTokenRequest{
		AuthArguments: &AuthArguments{
			Host:      "https://accounts.cloud.databricks.com",
			AccountID: "expired",
		},
		ProfileName: "expired",
		PersistentAuthOpts: []u2m.PersistentAuthOption{
			u2m.WithTokenCache(newTokenCache()),
			u2m.WithOAuthEndpointSupplier(&mockOAuthEndpointSupplier{}),
			u2m.WithHttpClient(&http.Client{Transport: fixtures.SliceTransport{refreshFailureTokenResponse}}),
		},
	})
	require.EqualError(t, err, `A new access token could not be retrieved because the refresh token is invalid. To reauthenticate, run the following command:
  $ databricks auth login --profile expired`)
}

func TestAcquireTokenRefreshFailureWithHostShowsLoginHint(t *testing.T) {
	_, err := AcquireToken(context.Background(), AcquireTokenRequest{
		AuthArguments: &AuthArguments{
			Host:      "https://accounts.cloud.databricks.com",
			AccountID: "expired",
		},
		PersistentAuthOpts: []u2m.PersistentAuthOption{
			u2m.WithTokenCache(newTokenCache()),
			u2m.WithOAuthEndpointSupplier(&mockOAuthEndpointSupplier{}),
			u2m.WithHttpClient(&http.Client{Transport: fixtures.SliceTransport{refreshFailureTokenResponse}}),
		},
	})
	require.EqualError(t, err, `A new access token could not be retrieved because the refresh token is invalid. To reauthenticate, run the following command:
  $ databricks auth login --host https://accounts.cloud.databricks.com --account-id expired`)
}

func TestAcquireTokenInvalidRefreshResponseShowsHelp(t *testing.T) {
	_, err := AcquireToken(context.Background(), AcquireTokenRequest{
		AuthArguments: &AuthArguments{
			Host:      "https://accounts.cloud.databricks.com",
			AccountID: "active",
		},
		ProfileName: "active",
		PersistentAuthOpts: []u2m.PersistentAuthOption{
			u2m.WithTokenCache(newTokenCache()),
			u2m.WithOAuthEndpointSupplier(&mockOAuthEndpointSupplier{}),
			u2m.WithHttpClient(&http.Client{Transport: fixtures.SliceTransport{refreshFailureInvalidResponse}}),
		},
	})
	require.EqualError(t, err, "token refresh: oauth2: cannot parse json: invalid character 'N' looking for beginning of value. Try logging in again with `databricks auth login --profile active` before retrying. If this fails, please report this issue to the Databricks CLI maintainers at https://github.com/databricks/cli/issues/new")
}

func TestAcquireTokenOtherErrorShowsHelp(t *testing.T) {
	_, err := AcquireToken(context.Background(), AcquireTokenRequest{
		AuthArguments: &AuthArguments{
			Host:      "https://accounts.cloud.databricks.com",
			AccountID: "active",
		},
		ProfileName: "active",
		PersistentAuthOpts: []u2m.PersistentAuthOption{
			u2m.WithTokenCache(newTokenCache()),
			u2m.WithOAuthEndpointSupplier(&mockOAuthEndpointSupplier{}),
			u2m.WithHttpClient(&http.Client{Transport: fixtures.SliceTransport{refreshFailureOtherError}}),
		},
	})
	require.EqualError(t, err, "token refresh: Databricks is down (error code: other_error). Try logging in again with `databricks auth login --profile active` before retrying. If this fails, please report this issue to the Databricks CLI maintainers at https://github.com/databricks/cli/issues/new")
}

func TestAcquireTokenSuccessReturnsAccessToken(t *testing.T) {
	token, err := AcquireToken(context.Background(), AcquireTokenRequest{
		AuthArguments: &AuthArguments{
			Host:      "https://accounts.cloud.databricks.com",
			AccountID: "active",
		},
		ProfileName: "active",
		PersistentAuthOpts: []u2m.PersistentAuthOption{
			u2m.WithTokenCache(newTokenCache()),
			u2m.WithOAuthEndpointSupplier(&mockOAuthEndpointSupplier{}),
			u2m.WithHttpClient(&http.Client{Transport: fixtures.SliceTransport{refreshSuccessTokenResponse}}),
		},
	})
	require.NoError(t, err)
	assert.Equal(t, "new-access-token", token.AccessToken)
	assert.Equal(t, "Bearer", token.TokenType)
}

type memoryTokenCache struct {
	tokens map[string]*oauth2.Token
}

func newTokenCache() *memoryTokenCache {
	return &memoryTokenCache{
		tokens: map[string]*oauth2.Token{
			"https://accounts.cloud.databricks.com/oidc/accounts/expired": {
				RefreshToken: "expired",
			},
			"https://accounts.cloud.databricks.com/oidc/accounts/active": {
				RefreshToken: "active",
				Expiry:       time.Now().Add(1 * time.Hour),
			},
		},
	}
}

func (m *memoryTokenCache) Lookup(key string) (*oauth2.Token, error) {
	if tok, ok := m.tokens[key]; ok {
		return tok, nil
	}
	return nil, cache.ErrNotFound
}

func (m *memoryTokenCache) Store(key string, t *oauth2.Token) error {
	m.tokens[key] = t
	return nil
}

var _ cache.TokenCache = (*memoryTokenCache)(nil)
