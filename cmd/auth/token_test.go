package auth

import (
	"context"
	"testing"
	"time"

	"github.com/databricks/cli/libs/databrickscfg/profile"
	"github.com/databricks/databricks-sdk-go/credentials/cache"
	"github.com/databricks/databricks-sdk-go/credentials/oauth"
	"github.com/databricks/databricks-sdk-go/httpclient"
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
	tokenCache := &cache.InMemoryTokenCache{
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
	makeApiClient := func(f fixtures.HTTPFixture) *httpclient.ApiClient {
		return httpclient.NewApiClient(httpclient.ClientConfig{
			Transport: fixtures.SliceTransport{f},
		})
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
				oauthArgument: oauth.BasicOAuthArgument{},
				profileName:   "expired",
				args:          []string{},
				tokenTimeout:  1 * time.Hour,
				profiler:      profiler,
				persistentAuthOpts: []oauth.PersistentAuthOption{
					oauth.WithTokenCache(tokenCache),
					oauth.WithApiClient(makeApiClient(refreshFailureTokenResponse)),
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
				oauthArgument: oauth.BasicOAuthArgument{
					Host:      "https://accounts.cloud.databricks.com",
					AccountID: "expired",
				},
				profileName:  "",
				args:         []string{},
				tokenTimeout: 1 * time.Hour,
				profiler:     profiler,
				persistentAuthOpts: []oauth.PersistentAuthOption{
					oauth.WithTokenCache(tokenCache),
					oauth.WithApiClient(makeApiClient(refreshFailureTokenResponse)),
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
				oauthArgument: oauth.BasicOAuthArgument{},
				profileName:   "active",
				args:          []string{},
				tokenTimeout:  1 * time.Hour,
				profiler:      profiler,
				persistentAuthOpts: []oauth.PersistentAuthOption{
					oauth.WithTokenCache(tokenCache),
					oauth.WithApiClient(makeApiClient(refreshFailureInvalidResponse)),
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
				oauthArgument: oauth.BasicOAuthArgument{},
				profileName:   "active",
				args:          []string{},
				tokenTimeout:  1 * time.Hour,
				profiler:      profiler,
				persistentAuthOpts: []oauth.PersistentAuthOption{
					oauth.WithTokenCache(tokenCache),
					oauth.WithApiClient(makeApiClient(refreshFailureOtherError)),
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
				oauthArgument: oauth.BasicOAuthArgument{},
				profileName:   "active",
				args:          []string{},
				tokenTimeout:  1 * time.Hour,
				profiler:      profiler,
				persistentAuthOpts: []oauth.PersistentAuthOption{
					oauth.WithTokenCache(tokenCache),
					oauth.WithApiClient(makeApiClient(refreshSuccessTokenResponse)),
				},
			},
			want: validateToken,
		},
		{
			name: "succeeds with host",
			args: loadTokenArgs{
				oauthArgument: oauth.BasicOAuthArgument{Host: "https://accounts.cloud.databricks.com", AccountID: "active"},
				profileName:   "",
				args:          []string{},
				tokenTimeout:  1 * time.Hour,
				profiler:      profiler,
				persistentAuthOpts: []oauth.PersistentAuthOption{
					oauth.WithTokenCache(tokenCache),
					oauth.WithApiClient(makeApiClient(refreshSuccessTokenResponse)),
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
