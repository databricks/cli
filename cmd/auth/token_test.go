package auth

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/databricks/cli/libs/databrickscfg/profile"
	"github.com/databricks/databricks-sdk-go/credentials/oauth"
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
	Status:   401,
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

type MockApiClient struct {
	RefreshTokenResponse http.RoundTripper
}

// GetAccountOAuthEndpoints implements oauth.OAuthClient.
func (m *MockApiClient) GetAccountOAuthEndpoints(ctx context.Context, accountHost string, accountId string) (*oauth.OAuthAuthorizationServer, error) {
	return &oauth.OAuthAuthorizationServer{
		TokenEndpoint:         accountHost + "/token",
		AuthorizationEndpoint: accountHost + "/authorize",
	}, nil
}

// GetHttpClient implements oauth.OAuthClient.
func (m *MockApiClient) GetHttpClient(context.Context) *http.Client {
	return &http.Client{
		Transport: m.RefreshTokenResponse,
	}
}

// GetWorkspaceOAuthEndpoints implements oauth.OAuthClient.
func (m *MockApiClient) GetWorkspaceOAuthEndpoints(ctx context.Context, workspaceHost string) (*oauth.OAuthAuthorizationServer, error) {
	return &oauth.OAuthAuthorizationServer{
		TokenEndpoint:         workspaceHost + "/token",
		AuthorizationEndpoint: workspaceHost + "/authorize",
	}, nil
}

var _ oauth.OAuthClient = (*MockApiClient)(nil)

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
		},
	}
	tokenCache := &InMemoryTokenCache{
		Tokens: map[string]*oauth2.Token{
			"https://accounts.cloud.databricks.com/oidc/accounts/expired": {
				RefreshToken: "expired",
			},
			"https://accounts.cloud.databricks.com/oidc/accounts/active": {
				RefreshToken: "active",
				Expiry:       time.Now().Add(1 * time.Hour), // Hopefully unit tests don't take an hour to run
			},
		},
	}
	makeApiClient := func(f fixtures.HTTPFixture) *MockApiClient {
		return &MockApiClient{
			RefreshTokenResponse: fixtures.SliceTransport{f},
		}
	}
	wantErrors := func(substrings ...string) func(error) {
		return func(err error) {
			for _, s := range substrings {
				assert.ErrorContains(t, err, s)
			}
		}
	}
	validateToken := func(resp *oauth2.Token) {
		assert.Equal(t, "new-access-token", resp.AccessToken)
		assert.Equal(t, "Bearer", resp.TokenType)
	}

	cases := []struct {
		name    string
		args    loadTokenArgs
		want    func(*oauth2.Token)
		wantErr func(error)
	}{
		{
			name: "prints helpful login message on refresh failure when profile is specified",
			args: loadTokenArgs{
				authArguments: &authArguments{},
				profileName:   "expired",
				args:          []string{},
				tokenTimeout:  1 * time.Hour,
				profiler:      profiler,
				persistentAuthOpts: []oauth.PersistentAuthOption{
					oauth.WithTokenCache(tokenCache),
					oauth.WithOAuthClient(makeApiClient(refreshFailureTokenResponse)),
				},
			},
			wantErr: wantErrors(
				"a new access token could not be retrieved because the refresh token is invalid. To reauthenticate, run ",
				"auth login --host https://accounts.cloud.databricks.com --account-id expired",
			),
		},
		{
			name: "prints helpful login message on refresh failure when host is specified",
			args: loadTokenArgs{
				authArguments: &authArguments{
					host:      "https://accounts.cloud.databricks.com",
					accountId: "expired",
				},
				profileName:  "",
				args:         []string{},
				tokenTimeout: 1 * time.Hour,
				profiler:     profiler,
				persistentAuthOpts: []oauth.PersistentAuthOption{
					oauth.WithTokenCache(tokenCache),
					oauth.WithOAuthClient(makeApiClient(refreshFailureTokenResponse)),
				},
			},
			wantErr: wantErrors(
				"a new access token could not be retrieved because the refresh token is invalid. To reauthenticate, run ",
				"auth login --host https://accounts.cloud.databricks.com --account-id expired",
			),
		},
		{
			name: "prints helpful login message on invalid response",
			args: loadTokenArgs{
				authArguments: &authArguments{},
				profileName:   "active",
				args:          []string{},
				tokenTimeout:  1 * time.Hour,
				profiler:      profiler,
				persistentAuthOpts: []oauth.PersistentAuthOption{
					oauth.WithTokenCache(tokenCache),
					oauth.WithOAuthClient(makeApiClient(refreshFailureInvalidResponse)),
				},
			},
			wantErr: wantErrors(
				"unexpected parsing token response: invalid character 'N' looking for beginning of value. Try logging in again with ",
				"auth login --profile active` before retrying. If this fails, please report this issue to the Databricks CLI maintainers at https://github.com/databricks/cli/issues/new",
			),
		},
		{
			name: "prints helpful login message on other error response",
			args: loadTokenArgs{
				authArguments: &authArguments{},
				profileName:   "active",
				args:          []string{},
				tokenTimeout:  1 * time.Hour,
				profiler:      profiler,
				persistentAuthOpts: []oauth.PersistentAuthOption{
					oauth.WithTokenCache(tokenCache),
					oauth.WithOAuthClient(makeApiClient(refreshFailureOtherError)),
				},
			},
			wantErr: wantErrors(
				"unexpected error refreshing token: Databricks is down. Try logging in again with ",
				"auth login --profile active` before retrying. If this fails, please report this issue to the Databricks CLI maintainers at https://github.com/databricks/cli/issues/new",
			),
		},
		{
			name: "succeeds with profile",
			args: loadTokenArgs{
				authArguments: &authArguments{},
				profileName:   "active",
				args:          []string{},
				tokenTimeout:  1 * time.Hour,
				profiler:      profiler,
				persistentAuthOpts: []oauth.PersistentAuthOption{
					oauth.WithTokenCache(tokenCache),
					oauth.WithOAuthClient(makeApiClient(refreshSuccessTokenResponse)),
				},
			},
			want: validateToken,
		},
		{
			name: "succeeds with host",
			args: loadTokenArgs{
				authArguments: &authArguments{host: "https://accounts.cloud.databricks.com", accountId: "active"},
				profileName:   "",
				args:          []string{},
				tokenTimeout:  1 * time.Hour,
				profiler:      profiler,
				persistentAuthOpts: []oauth.PersistentAuthOption{
					oauth.WithTokenCache(tokenCache),
					oauth.WithOAuthClient(makeApiClient(refreshSuccessTokenResponse)),
				},
			},
			want: validateToken,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := loadToken(context.Background(), c.args)
			if c.wantErr != nil {
				c.wantErr(err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, c.want, got)
			}
		})
	}
}
