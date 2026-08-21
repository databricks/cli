package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/auth/storage"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/databrickscfg"
	"github.com/databricks/cli/libs/databrickscfg/profile"
	"github.com/databricks/cli/libs/dockercredentials"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/databricks/databricks-sdk-go/credentials/u2m"
	"github.com/databricks/databricks-sdk-go/httpclient/fixtures"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

// upgradeHintStore returns a notFoundHint-wrapped ErrNotFound on
// Lookup, mirroring what storage.notFoundHintCache produces in
// production when ~/.databricks/token-cache.json has entries and the
// resolver picked secure mode by default. Used by TestToken_loadToken
// to verify that auth token surfaces the upgrade-specific hint instead
// of dropping it for the SDK-compat constant string.
type upgradeHintStore struct{}

func (upgradeHintStore) Put(string, storage.Entry) error { return nil }
func (upgradeHintStore) Delete(string) error             { return nil }
func (upgradeHintStore) Lookup(string) (storage.Entry, error) {
	return storage.Entry{}, storage.NewNotFoundHint(
		"stored credentials from older CLI versions are no longer used; run `databricks auth login` to sign in again, or set DATABRICKS_AUTH_STORAGE=plaintext to keep using the file cache",
	)
}

var _ storage.Store = upgradeHintStore{}

type failOnCallTransport struct{}

func (failOnCallTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, errors.New("unexpected HTTP call")
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

// GetEndpointsFromURL implements u2m.OAuthEndpointSupplier.
func (m *MockApiClient) GetEndpointsFromURL(_ context.Context, _ string) (*u2m.OAuthAuthorizationServer, error) {
	return nil, u2m.ErrOAuthNotSupported
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
			{
				Name:                 "m2m-profile",
				Host:                 "https://m2m.cloud.databricks.com",
				HasClientCredentials: true,
			},
			{
				Name: "valid-token",
				Host: "https://valid-token.cloud.databricks.com",
			},
		},
	}
	tokenStore := &inMemoryStore{
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
			"dup1": {
				RefreshToken: "dup1",
				Expiry:       time.Now().Add(1 * time.Hour),
			},
			"valid-token": {
				AccessToken:  "cached-access-token",
				RefreshToken: "valid-token",
				Expiry:       time.Now().Add(1 * time.Hour),
			},
		},
	}
	validateToken := func(got *oauth2.Token) {
		assert.Equal(t, "new-access-token", got.AccessToken)
		assert.Equal(t, "Bearer", got.TokenType)
	}

	cases := []struct {
		name          string
		setupCtx      func(context.Context) context.Context
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
				tokenStore:    tokenStore,
				persistentAuthOpts: []u2m.PersistentAuthOption{
					u2m.WithTokenCache(storage.ToU2MTokenCache(tokenStore)),
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
				tokenStore:   tokenStore,
				persistentAuthOpts: []u2m.PersistentAuthOption{
					u2m.WithTokenCache(storage.ToU2MTokenCache(tokenStore)),
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
				tokenStore:    tokenStore,
				persistentAuthOpts: []u2m.PersistentAuthOption{
					u2m.WithTokenCache(storage.ToU2MTokenCache(tokenStore)),
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
				tokenStore:    tokenStore,
				persistentAuthOpts: []u2m.PersistentAuthOption{
					u2m.WithTokenCache(storage.ToU2MTokenCache(tokenStore)),
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
				tokenStore:    tokenStore,
				persistentAuthOpts: []u2m.PersistentAuthOption{
					u2m.WithTokenCache(storage.ToU2MTokenCache(tokenStore)),
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
				tokenStore:    tokenStore,
				persistentAuthOpts: []u2m.PersistentAuthOption{
					u2m.WithTokenCache(storage.ToU2MTokenCache(tokenStore)),
					u2m.WithOAuthEndpointSupplier(&MockApiClient{}),
					u2m.WithHttpClient(&http.Client{Transport: fixtures.SliceTransport{refreshSuccessTokenResponse}}),
				},
			},
			validateToken: validateToken,
		},
		{
			name: "host with trailing slash is stripped",
			args: loadTokenArgs{
				authArguments: &auth.AuthArguments{Host: "https://accounts.cloud.databricks.com/", AccountID: "active"},
				profileName:   "",
				args:          []string{},
				tokenTimeout:  1 * time.Hour,
				profiler:      profiler,
				tokenStore:    tokenStore,
				persistentAuthOpts: []u2m.PersistentAuthOption{
					u2m.WithTokenCache(storage.ToU2MTokenCache(tokenStore)),
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
				tokenStore:    tokenStore,
				persistentAuthOpts: []u2m.PersistentAuthOption{
					u2m.WithTokenCache(storage.ToU2MTokenCache(tokenStore)),
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
				tokenStore:    tokenStore,
				persistentAuthOpts: []u2m.PersistentAuthOption{
					u2m.WithTokenCache(storage.ToU2MTokenCache(tokenStore)),
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
				tokenStore:    tokenStore,
				persistentAuthOpts: []u2m.PersistentAuthOption{
					u2m.WithTokenCache(storage.ToU2MTokenCache(tokenStore)),
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
				args:          []string{"nonexistent.cloud.databricks.com"},
				tokenTimeout:  1 * time.Hour,
				profiler:      profiler,
				tokenStore:    tokenStore,
				persistentAuthOpts: []u2m.PersistentAuthOption{
					u2m.WithTokenCache(storage.ToU2MTokenCache(tokenStore)),
					u2m.WithOAuthEndpointSupplier(&MockApiClient{}),
				},
			},
			wantErr: "cache: databricks OAuth is not configured for this host. " +
				"Try logging in again with `databricks auth login --host https://nonexistent.cloud.databricks.com` before retrying. " +
				"If this fails, please report this issue to the Databricks CLI maintainers at https://github.com/databricks/cli/issues/new",
		},
		{
			// Regression test: when notFoundHintCache wraps ErrNotFound
			// with the upgrade copy (post-upgrade default-secure user
			// with a populated token-cache.json), `auth token` must
			// surface that hint instead of dropping it for the SDK-compat
			// constant string. The combined message keeps the
			// "OAuth is not configured for this host" substring older
			// SDK versions look for and skips the generic "Try logging
			// in again ... If this fails, please report this issue"
			// trailer, which would mislead users into reporting expected
			// post-upgrade behavior.
			name: "ErrNotFound carrying upgrade hint surfaces it",
			args: loadTokenArgs{
				authArguments: &auth.AuthArguments{},
				profileName:   "",
				args:          []string{"nonexistent.cloud.databricks.com"},
				tokenTimeout:  1 * time.Hour,
				profiler:      profiler,
				tokenStore:    upgradeHintStore{},
				persistentAuthOpts: []u2m.PersistentAuthOption{
					u2m.WithTokenCache(storage.ToU2MTokenCache(upgradeHintStore{})),
					u2m.WithOAuthEndpointSupplier(&MockApiClient{}),
				},
			},
			wantErr: "cache: databricks OAuth is not configured for this host. " +
				"stored credentials from older CLI versions are no longer used; " +
				"run `databricks auth login` to sign in again, " +
				"or set DATABRICKS_AUTH_STORAGE=plaintext to keep using the file cache",
		},
		{
			name: "errors with clear message for non-host non-profile positional arg",
			args: loadTokenArgs{
				authArguments: &auth.AuthArguments{},
				profileName:   "",
				args:          []string{"e2-logfood"},
				tokenTimeout:  1 * time.Hour,
				profiler:      profiler,
			},
			wantErr: `no matching profile found: "e2-logfood"`,
		},
		{
			name: "scheme-less account host ambiguity detected correctly",
			args: loadTokenArgs{
				authArguments: &auth.AuthArguments{
					Host:      "accounts.cloud.databricks.com",
					AccountID: "same-account",
				},
				profileName:  "",
				args:         []string{},
				tokenTimeout: 1 * time.Hour,
				profiler:     profiler,
				tokenStore:   tokenStore,
				persistentAuthOpts: []u2m.PersistentAuthOption{
					u2m.WithTokenCache(storage.ToU2MTokenCache(tokenStore)),
					u2m.WithOAuthEndpointSupplier(&MockApiClient{}),
				},
			},
			wantErr: "acct-dup1 and acct-dup2 match accounts.cloud.databricks.com in <in memory>. Use --profile to specify which profile to use",
		},
		{
			name: "workspace host ambiguity — multiple profiles, non-interactive",
			args: loadTokenArgs{
				authArguments: &auth.AuthArguments{
					Host: "https://shared.cloud.databricks.com",
				},
				profileName:  "",
				args:         []string{},
				tokenTimeout: 1 * time.Hour,
				profiler:     profiler,
				tokenStore:   tokenStore,
				persistentAuthOpts: []u2m.PersistentAuthOption{
					u2m.WithTokenCache(storage.ToU2MTokenCache(tokenStore)),
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
				tokenStore:   tokenStore,
				persistentAuthOpts: []u2m.PersistentAuthOption{
					u2m.WithTokenCache(storage.ToU2MTokenCache(tokenStore)),
					u2m.WithOAuthEndpointSupplier(&MockApiClient{}),
					u2m.WithHttpClient(&http.Client{Transport: fixtures.SliceTransport{refreshSuccessTokenResponse}}),
				},
			},
			validateToken: validateToken,
		},
		{
			name: "account host — same host AND same account ID — ambiguity",
			args: loadTokenArgs{
				authArguments: &auth.AuthArguments{
					Host:      "https://accounts.cloud.databricks.com",
					AccountID: "same-account",
				},
				profileName:  "",
				args:         []string{},
				tokenTimeout: 1 * time.Hour,
				profiler:     profiler,
				tokenStore:   tokenStore,
				persistentAuthOpts: []u2m.PersistentAuthOption{
					u2m.WithTokenCache(storage.ToU2MTokenCache(tokenStore)),
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
				tokenStore:   tokenStore,
				persistentAuthOpts: []u2m.PersistentAuthOption{
					u2m.WithTokenCache(storage.ToU2MTokenCache(tokenStore)),
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
				tokenStore:   tokenStore,
				persistentAuthOpts: []u2m.PersistentAuthOption{
					u2m.WithTokenCache(storage.ToU2MTokenCache(tokenStore)),
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
				tokenStore:    tokenStore,
				persistentAuthOpts: []u2m.PersistentAuthOption{
					u2m.WithTokenCache(storage.ToU2MTokenCache(tokenStore)),
					u2m.WithOAuthEndpointSupplier(&MockApiClient{}),
				},
			},
			wantErr: `argument "workspace-a" cannot be combined with --host or --profile. Use the --host and --profile flags instead`,
		},
		{
			name: "no args, profiles exist, non-interactive — error with profile hint",
			args: loadTokenArgs{
				authArguments:      &auth.AuthArguments{},
				profileName:        "",
				args:               []string{},
				tokenTimeout:       1 * time.Hour,
				profiler:           profiler,
				tokenStore:         tokenStore,
				persistentAuthOpts: nil,
			},
			wantErr: "no profile specified. Use --profile <name> to specify which profile to use",
		},
		{
			name: "no args, no profiles, non-interactive — error with login hint",
			args: loadTokenArgs{
				authArguments:      &auth.AuthArguments{},
				profileName:        "",
				args:               []string{},
				tokenTimeout:       1 * time.Hour,
				profiler:           profile.InMemoryProfiler{},
				tokenStore:         tokenStore,
				persistentAuthOpts: nil,
			},
			wantErr: "no profiles configured. Run 'databricks auth login' to create a profile",
		},
		{
			name: "no args, no config file, non-interactive — error with login hint",
			args: loadTokenArgs{
				authArguments:      &auth.AuthArguments{},
				profileName:        "",
				args:               []string{},
				tokenTimeout:       1 * time.Hour,
				profiler:           errProfiler{err: profile.ErrNoConfiguration},
				tokenStore:         tokenStore,
				persistentAuthOpts: nil,
			},
			wantErr: "no profiles configured. Run 'databricks auth login' to create a profile",
		},
		{
			name: "M2M profile returns clear error",
			args: loadTokenArgs{
				authArguments: &auth.AuthArguments{},
				profileName:   "m2m-profile",
				args:          []string{},
				tokenTimeout:  1 * time.Hour,
				profiler:      profiler,
			},
			wantErr: `profile "m2m-profile" uses M2M authentication (client_id/client_secret). ` +
				"`databricks auth token` only supports U2M (user-to-machine) authentication tokens. " +
				"To authenticate as a service principal, use the Databricks SDK directly",
		},
		{
			name: "M2M profile detected via positional arg",
			args: loadTokenArgs{
				authArguments: &auth.AuthArguments{},
				profileName:   "",
				args:          []string{"m2m-profile"},
				tokenTimeout:  1 * time.Hour,
				profiler:      profiler,
			},
			wantErr: `profile "m2m-profile" uses M2M authentication (client_id/client_secret). ` +
				"`databricks auth token` only supports U2M (user-to-machine) authentication tokens. " +
				"To authenticate as a service principal, use the Databricks SDK directly",
		},
		{
			name: "M2M profile detected via host resolution",
			args: loadTokenArgs{
				authArguments: &auth.AuthArguments{
					Host: "https://m2m.cloud.databricks.com",
				},
				profileName:  "",
				args:         []string{},
				tokenTimeout: 1 * time.Hour,
				profiler:     profiler,
			},
			wantErr: `profile "m2m-profile" uses M2M authentication (client_id/client_secret). ` +
				"`databricks auth token` only supports U2M (user-to-machine) authentication tokens. " +
				"To authenticate as a service principal, use the Databricks SDK directly",
		},
		{
			name: "no args, DATABRICKS_HOST env resolves",
			setupCtx: func(ctx context.Context) context.Context {
				ctx = env.Set(ctx, "DATABRICKS_HOST", "https://workspace-a.cloud.databricks.com")
				return ctx
			},
			args: loadTokenArgs{
				authArguments: &auth.AuthArguments{},
				profileName:   "",
				args:          []string{},
				tokenTimeout:  1 * time.Hour,
				profiler:      profiler,
				tokenStore:    tokenStore,
				persistentAuthOpts: []u2m.PersistentAuthOption{
					u2m.WithTokenCache(storage.ToU2MTokenCache(tokenStore)),
					u2m.WithOAuthEndpointSupplier(&MockApiClient{}),
					u2m.WithHttpClient(&http.Client{Transport: fixtures.SliceTransport{refreshSuccessTokenResponse}}),
				},
			},
			validateToken: validateToken,
		},
		{
			name: "no args, DATABRICKS_HOST env with trailing slash resolves",
			setupCtx: func(ctx context.Context) context.Context {
				ctx = env.Set(ctx, "DATABRICKS_HOST", "https://workspace-a.cloud.databricks.com/")
				return ctx
			},
			args: loadTokenArgs{
				authArguments: &auth.AuthArguments{},
				profileName:   "",
				args:          []string{},
				tokenTimeout:  1 * time.Hour,
				profiler:      profiler,
				tokenStore:    tokenStore,
				persistentAuthOpts: []u2m.PersistentAuthOption{
					u2m.WithTokenCache(storage.ToU2MTokenCache(tokenStore)),
					u2m.WithOAuthEndpointSupplier(&MockApiClient{}),
					u2m.WithHttpClient(&http.Client{Transport: fixtures.SliceTransport{refreshSuccessTokenResponse}}),
				},
			},
			validateToken: validateToken,
		},
		{
			name: "no args, DATABRICKS_CONFIG_PROFILE env resolves",
			setupCtx: func(ctx context.Context) context.Context {
				ctx = env.Set(ctx, "DATABRICKS_CONFIG_PROFILE", "active")
				return ctx
			},
			args: loadTokenArgs{
				authArguments: &auth.AuthArguments{},
				profileName:   "",
				args:          []string{},
				tokenTimeout:  1 * time.Hour,
				profiler:      profiler,
				tokenStore:    tokenStore,
				persistentAuthOpts: []u2m.PersistentAuthOption{
					u2m.WithTokenCache(storage.ToU2MTokenCache(tokenStore)),
					u2m.WithOAuthEndpointSupplier(&MockApiClient{}),
					u2m.WithHttpClient(&http.Client{Transport: fixtures.SliceTransport{refreshSuccessTokenResponse}}),
				},
			},
			validateToken: validateToken,
		},
		{
			name: "no args, DATABRICKS_CONFIG_PROFILE env takes precedence over DATABRICKS_HOST",
			setupCtx: func(ctx context.Context) context.Context {
				ctx = env.Set(ctx, "DATABRICKS_HOST", "https://workspace-a.cloud.databricks.com")
				ctx = env.Set(ctx, "DATABRICKS_CONFIG_PROFILE", "expired")
				return ctx
			},
			args: loadTokenArgs{
				authArguments: &auth.AuthArguments{},
				profileName:   "",
				args:          []string{},
				tokenTimeout:  1 * time.Hour,
				profiler:      profiler,
				tokenStore:    tokenStore,
				persistentAuthOpts: []u2m.PersistentAuthOption{
					u2m.WithTokenCache(storage.ToU2MTokenCache(tokenStore)),
					u2m.WithOAuthEndpointSupplier(&MockApiClient{}),
					u2m.WithHttpClient(&http.Client{Transport: fixtures.SliceTransport{refreshSuccessTokenResponse}}),
				},
			},
			validateToken: validateToken,
		},
		{
			name: "DATABRICKS_CONFIG_PROFILE with positional typo runs resolver first",
			setupCtx: func(ctx context.Context) context.Context {
				return env.Set(ctx, "DATABRICKS_CONFIG_PROFILE", "active")
			},
			args: loadTokenArgs{
				authArguments: &auth.AuthArguments{},
				profileName:   "",
				args:          []string{"e2-logfood"},
				tokenTimeout:  1 * time.Hour,
				profiler:      profiler,
			},
			wantErr: `no matching profile found: "e2-logfood"`,
		},
		{
			name: "host flag with profile env var disambiguates multi-profile",
			setupCtx: func(ctx context.Context) context.Context {
				return env.Set(ctx, "DATABRICKS_CONFIG_PROFILE", "dup1")
			},
			args: loadTokenArgs{
				authArguments: &auth.AuthArguments{
					Host: "https://shared.cloud.databricks.com",
				},
				profileName:  "",
				args:         []string{},
				tokenTimeout: 1 * time.Hour,
				profiler:     profiler,
				tokenStore:   tokenStore,
				persistentAuthOpts: []u2m.PersistentAuthOption{
					u2m.WithTokenCache(storage.ToU2MTokenCache(tokenStore)),
					u2m.WithOAuthEndpointSupplier(&MockApiClient{}),
					u2m.WithHttpClient(&http.Client{Transport: fixtures.SliceTransport{refreshSuccessTokenResponse}}),
				},
			},
			validateToken: validateToken,
		},
		{
			name: "default path reuses valid cached token without refresh",
			args: loadTokenArgs{
				authArguments: &auth.AuthArguments{},
				profileName:   "valid-token",
				args:          []string{},
				tokenTimeout:  1 * time.Hour,
				profiler:      profiler,
				tokenStore:    tokenStore,
				persistentAuthOpts: []u2m.PersistentAuthOption{
					u2m.WithTokenCache(storage.ToU2MTokenCache(tokenStore)),
					u2m.WithOAuthEndpointSupplier(&MockApiClient{}),
					u2m.WithHttpClient(&http.Client{Transport: failOnCallTransport{}}),
				},
			},
			validateToken: func(got *oauth2.Token) {
				assert.Equal(t, "cached-access-token", got.AccessToken)
			},
		},
		{
			name: "force refresh refreshes valid cached token",
			args: loadTokenArgs{
				authArguments: &auth.AuthArguments{},
				profileName:   "valid-token",
				args:          []string{},
				tokenTimeout:  1 * time.Hour,
				forceRefresh:  true,
				profiler:      profiler,
				tokenStore:    tokenStore,
				persistentAuthOpts: []u2m.PersistentAuthOption{
					u2m.WithTokenCache(storage.ToU2MTokenCache(tokenStore)),
					u2m.WithOAuthEndpointSupplier(&MockApiClient{}),
					u2m.WithHttpClient(&http.Client{Transport: fixtures.SliceTransport{refreshSuccessTokenResponse}}),
				},
			},
			validateToken: validateToken,
		},
		{
			name: "force refresh preserves error handling on refresh failure",
			args: loadTokenArgs{
				authArguments: &auth.AuthArguments{},
				profileName:   "valid-token",
				args:          []string{},
				tokenTimeout:  1 * time.Hour,
				forceRefresh:  true,
				profiler:      profiler,
				tokenStore:    tokenStore,
				persistentAuthOpts: []u2m.PersistentAuthOption{
					u2m.WithTokenCache(storage.ToU2MTokenCache(tokenStore)),
					u2m.WithOAuthEndpointSupplier(&MockApiClient{}),
					u2m.WithHttpClient(&http.Client{Transport: fixtures.SliceTransport{refreshFailureTokenResponse}}),
				},
			},
			wantErr: `A new access token could not be retrieved because the refresh token is invalid. To reauthenticate, run the following command:
  $ databricks auth login --profile valid-token`,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ctx := cmdio.MockDiscard(t.Context())
			if c.setupCtx != nil {
				ctx = c.setupCtx(ctx)
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

// errProfiler is a Profiler that always returns the configured error.
type errProfiler struct {
	err error
}

func (e errProfiler) LoadProfiles(context.Context, profile.ProfileMatchFunction) (profile.Profiles, error) {
	return nil, e.err
}

func (e errProfiler) GetPath(context.Context) (string, error) {
	return "<error>", nil
}

func TestTokenDockerFormatEmitsGetResponse(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	dir := t.TempDir()
	configFile := filepath.Join(dir, ".databrickscfg")
	require.NoError(t, databrickscfg.SaveToProfile(ctx, &config.Config{
		ConfigFile:  configFile,
		Profile:     "workspace",
		Host:        "https://workspace.cloud.databricks.test",
		WorkspaceID: "123456789",
		AuthType:    authTypeDatabricksCLI,
	}))

	t.Setenv("DATABRICKS_CONFIG_FILE", configFile)
	t.Setenv(storage.EnvVar, string(storage.StorageModePlaintext))
	t.Setenv("HOME", dir)

	var gotProfile string
	loadToken := func(_ context.Context, args loadTokenArgs) (*oauth2.Token, error) {
		gotProfile = args.profileName
		return &oauth2.Token{AccessToken: "access-token"}, nil
	}

	registryHost := "123456789.container.us-west-2.cloud.databricks.com"
	var stdout bytes.Buffer
	cmd := newTokenCommandWithRegistryHost(&auth.AuthArguments{}, loadToken, func(workspaceID, region, workspaceHost string) (string, error) {
		require.Equal(t, "123456789", workspaceID)
		require.Equal(t, "us-west-2", region)
		require.Equal(t, "https://workspace.cloud.databricks.test", workspaceHost)
		return registryHost, nil
	})
	cmd.Flags().StringP("profile", "p", "", "~/.databrickscfg profile")
	cmd.SetContext(ctx)
	cmd.SetIn(strings.NewReader(registryHost + "\n"))
	cmd.SetOut(&stdout)
	cmd.SetArgs([]string{"--format=docker"})

	require.NoError(t, cmd.Execute())
	require.Equal(t, "workspace", gotProfile)

	var got map[string]string
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &got))
	require.Equal(t, map[string]string{
		"Username": "oauthtoken",
		"Secret":   "access-token",
	}, got)
}

func TestWriteDockerTokenOutputUsesConfiguredProfiler(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	profiler := profile.InMemoryProfiler{
		Profiles: profile.Profiles{
			{
				Name:        "workspace",
				Host:        "https://workspace.cloud.databricks.test",
				WorkspaceID: "123456789",
				AuthType:    authTypeDatabricksCLI,
			},
		},
	}

	var gotProfile string
	loadToken := func(_ context.Context, args loadTokenArgs) (*oauth2.Token, error) {
		gotProfile = args.profileName
		return &oauth2.Token{AccessToken: "access-token"}, nil
	}

	cmd := &cobra.Command{Use: "token"}
	var stdout bytes.Buffer
	cmd.SetContext(ctx)
	cmd.SetIn(strings.NewReader("123456789.container.us-west-2.cloud.databricks.com\n"))
	cmd.SetOut(&stdout)

	err := writeDockerTokenOutput(ctx, cmd, loadTokenArgs{
		authArguments: &auth.AuthArguments{},
		profiler:      profiler,
	}, loadToken, func(string, string, string) (string, error) {
		return "123456789.container.us-west-2.cloud.databricks.com", nil
	})
	require.NoError(t, err)
	require.Equal(t, "workspace", gotProfile)

	var got dockerGetResponse
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &got))
}

func TestDockerTokenProfileNameRejectsDifferentEnvironment(t *testing.T) {
	registry := dockercredentials.Registry{
		WorkspaceID: "123456789",
		Region:      "us-west-2",
		Host:        "123456789.container.us-west-2.cloud.databricks.com",
	}
	profiler := profile.InMemoryProfiler{
		Profiles: profile.Profiles{{
			Name:        "workspace",
			Host:        "https://workspace.dev.cloud.databricks.test",
			WorkspaceID: registry.WorkspaceID,
			AuthType:    authTypeDatabricksCLI,
		}},
	}
	registryHost := func(workspaceID, region, workspaceHost string) (string, error) {
		require.Equal(t, registry.WorkspaceID, workspaceID)
		require.Equal(t, registry.Region, region)
		require.Equal(t, "https://workspace.dev.cloud.databricks.test", workspaceHost)
		return "123456789.container.us-west-2.dev.cloud.databricks.com", nil
	}

	_, err := dockerTokenProfileName(t.Context(), registry, profiler, registryHost)
	require.ErrorContains(t, err, "does not match profile")
	require.ErrorContains(t, err, "workspace host")
}

func TestDockerTokenProfileNameAllowsSameWorkspaceIDInDifferentEnvironment(t *testing.T) {
	registry := dockercredentials.Registry{
		WorkspaceID: "123456789",
		Region:      "us-west-2",
		Host:        "123456789.container.us-west-2.cloud.databricks.com",
	}
	profiler := profile.InMemoryProfiler{
		Profiles: profile.Profiles{
			{
				Name:        "prod",
				Host:        "https://workspace.cloud.databricks.com",
				WorkspaceID: registry.WorkspaceID,
				AuthType:    authTypeDatabricksCLI,
			},
			{
				Name:        "dev",
				Host:        "https://workspace.dev.cloud.databricks.com",
				WorkspaceID: registry.WorkspaceID,
				AuthType:    authTypeDatabricksCLI,
			},
		},
	}
	registryHost := func(workspaceID, region, workspaceHost string) (string, error) {
		zone := ".cloud.databricks.com"
		if workspaceHost == "https://workspace.dev.cloud.databricks.com" {
			zone = ".dev.cloud.databricks.com"
		}
		return workspaceID + ".container." + region + zone, nil
	}

	profileName, err := dockerTokenProfileName(t.Context(), registry, profiler, registryHost)
	require.NoError(t, err)
	require.Equal(t, "prod", profileName)
}

func TestDockerTokenProfileNameIgnoresUnsupportedDuplicateProfile(t *testing.T) {
	registry := dockercredentials.Registry{
		WorkspaceID: "123456789",
		Region:      "us-west-2",
		Host:        "123456789.container.us-west-2.cloud.databricks.com",
	}
	profiler := profile.InMemoryProfiler{
		Profiles: profile.Profiles{
			{
				Name:        "workspace",
				Host:        "https://workspace.cloud.databricks.com",
				WorkspaceID: registry.WorkspaceID,
				AuthType:    authTypeDatabricksCLI,
			},
			{
				Name:                 "m2m",
				Host:                 "https://workspace.cloud.databricks.com",
				WorkspaceID:          registry.WorkspaceID,
				HasClientCredentials: true,
			},
		},
	}
	registryHost := func(workspaceID, region, _ string) (string, error) {
		return workspaceID + ".container." + region + ".cloud.databricks.com", nil
	}

	profileName, err := dockerTokenProfileName(t.Context(), registry, profiler, registryHost)
	require.NoError(t, err)
	require.Equal(t, "workspace", profileName)
}

func TestTokenDockerFormatRejectsPositionalArgs(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	t.Setenv(storage.EnvVar, string(storage.StorageModePlaintext))
	t.Setenv("HOME", t.TempDir())

	cmd := newTokenCommandWithLoader(&auth.AuthArguments{}, func(context.Context, loadTokenArgs) (*oauth2.Token, error) {
		t.Fatal("loadToken should not be called")
		return nil, nil
	})
	cmd.Flags().StringP("profile", "p", "", "~/.databrickscfg profile")
	cmd.SetContext(ctx)
	cmd.SetIn(strings.NewReader("123456789.container.us-west-2.cloud.databricks.com\n"))
	cmd.SetArgs([]string{"--format=docker", "DEFAULT"})

	err := cmd.Execute()
	require.ErrorContains(t, err, "--format=docker does not accept positional arguments")
}

func TestTokenDockerFormatValidatesBeforeResolvingTokenStore(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	t.Setenv(storage.EnvVar, "invalid")

	cmd := newTokenCommandWithLoader(&auth.AuthArguments{}, func(context.Context, loadTokenArgs) (*oauth2.Token, error) {
		t.Fatal("loadToken should not be called")
		return nil, nil
	})
	cmd.Flags().StringP("profile", "p", "", "~/.databrickscfg profile")
	cmd.SetContext(ctx)
	cmd.SetArgs([]string{"--format=docker", "DEFAULT"})

	err := cmd.Execute()
	require.ErrorContains(t, err, "--format=docker does not accept positional arguments")
}

func TestTokenDockerFormatRejectsAuthSelectionFlags(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	dir := t.TempDir()
	configFile := filepath.Join(dir, ".databrickscfg")
	require.NoError(t, databrickscfg.SaveToProfile(ctx, &config.Config{
		ConfigFile: configFile,
		Profile:    "DEFAULT",
		Host:       "https://profile.cloud.databricks.test",
		AuthType:   authTypeDatabricksCLI,
	}))
	t.Setenv("DATABRICKS_CONFIG_FILE", configFile)
	t.Setenv(storage.EnvVar, string(storage.StorageModePlaintext))
	t.Setenv("HOME", dir)

	cases := [][]string{
		{"--format=docker", "--profile", "DEFAULT"},
		{"--format=docker", "--host", "https://workspace.cloud.databricks.test"},
		{"--format=docker", "--profile", "DEFAULT", "--host", "https://workspace.cloud.databricks.test"},
		{"--format=docker", "--account-id", "abc"},
		{"--format=docker", "--workspace-id", "123456789"},
	}

	for _, args := range cases {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			var authArgs auth.AuthArguments
			cmd := &cobra.Command{Use: "auth"}
			cmd.PersistentFlags().StringVar(&authArgs.Host, "host", "", "Databricks Host")
			cmd.PersistentFlags().StringVar(&authArgs.AccountID, "account-id", "", "Databricks Account ID")
			cmd.PersistentFlags().StringVar(&authArgs.WorkspaceID, "workspace-id", "", "Databricks Workspace ID")
			cmd.AddCommand(newTokenCommandWithLoader(&authArgs, func(context.Context, loadTokenArgs) (*oauth2.Token, error) {
				t.Fatal("loadToken should not be called")
				return nil, nil
			}))
			cmd.PersistentFlags().StringP("profile", "p", "", "~/.databrickscfg profile")
			cmd.SetContext(ctx)
			cmd.SetIn(strings.NewReader("123456789.container.us-west-2.cloud.databricks.com\n"))
			cmd.SetArgs(append([]string{"token"}, args...))

			err := cmd.Execute()
			require.ErrorContains(t, err, "--format=docker does not support")
		})
	}
}

func TestTokenDockerFormatRejectsNonDARHost(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	t.Setenv(storage.EnvVar, string(storage.StorageModePlaintext))
	t.Setenv("HOME", t.TempDir())

	cmd := newTokenCommandWithLoader(&auth.AuthArguments{}, func(context.Context, loadTokenArgs) (*oauth2.Token, error) {
		t.Fatal("loadToken should not be called")
		return nil, nil
	})
	cmd.Flags().StringP("profile", "p", "", "~/.databrickscfg profile")
	cmd.SetContext(ctx)
	cmd.SetIn(strings.NewReader("registry.example.com\n"))
	cmd.SetArgs([]string{"--format=docker"})

	err := cmd.Execute()
	require.ErrorContains(t, err, "is not a Databricks Artifact Registry host")
}

func TestTokenDockerFormatErrorsWithoutMatchingProfile(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	dir := t.TempDir()
	configFile := filepath.Join(dir, ".databrickscfg")
	require.NoError(t, os.WriteFile(configFile, []byte(""), 0o600))

	t.Setenv("DATABRICKS_CONFIG_FILE", configFile)
	t.Setenv(storage.EnvVar, string(storage.StorageModePlaintext))
	t.Setenv("HOME", dir)

	cmd := newTokenCommandWithLoader(&auth.AuthArguments{}, func(context.Context, loadTokenArgs) (*oauth2.Token, error) {
		t.Fatal("loadToken should not be called")
		return nil, nil
	})
	cmd.Flags().StringP("profile", "p", "", "~/.databrickscfg profile")
	cmd.SetContext(ctx)
	cmd.SetIn(strings.NewReader("123456789.container.us-west-2.cloud.databricks.com\n"))
	cmd.SetArgs([]string{"--format=docker"})

	err := cmd.Execute()
	require.ErrorContains(t, err, "no Databricks profile found for workspace ID 123456789")
	require.ErrorContains(t, err, "databricks auth login --host <workspace-url>")
	require.ErrorContains(t, err, "workspace_id")
}

func TestTokenDockerFormatErrorsWithMultipleMatchingProfiles(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	dir := t.TempDir()
	configFile := filepath.Join(dir, ".databrickscfg")
	for _, name := range []string{"one", "two"} {
		require.NoError(t, databrickscfg.SaveToProfile(ctx, &config.Config{
			ConfigFile:  configFile,
			Profile:     name,
			Host:        "https://" + name + ".cloud.databricks.test",
			WorkspaceID: "123456789",
			AuthType:    authTypeDatabricksCLI,
		}))
	}

	t.Setenv("DATABRICKS_CONFIG_FILE", configFile)
	t.Setenv(storage.EnvVar, string(storage.StorageModePlaintext))
	t.Setenv("HOME", dir)

	cmd := newTokenCommandWithRegistryHost(&auth.AuthArguments{}, func(context.Context, loadTokenArgs) (*oauth2.Token, error) {
		t.Fatal("loadToken should not be called")
		return nil, nil
	}, func(workspaceID, region, _ string) (string, error) {
		return workspaceID + ".container." + region + ".cloud.databricks.com", nil
	})
	cmd.Flags().StringP("profile", "p", "", "~/.databrickscfg profile")
	cmd.SetContext(ctx)
	cmd.SetIn(strings.NewReader("123456789.container.us-west-2.cloud.databricks.com\n"))
	cmd.SetArgs([]string{"--format=docker"})

	err := cmd.Execute()
	require.ErrorContains(t, err, "multiple Databricks profiles match workspace ID 123456789")
	require.ErrorContains(t, err, "one and two")
	require.ErrorContains(t, err, "Remove duplicate workspace_id entries")
}

func TestTokenDockerFormatRejectsUnsupportedProfile(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	profiler := profile.InMemoryProfiler{
		Profiles: profile.Profiles{
			{
				Name:        "pat",
				Host:        "https://workspace.cloud.databricks.test",
				WorkspaceID: "123456789",
				AuthType:    "pat",
			},
			{
				Name:                 "m2m",
				Host:                 "https://m2m.cloud.databricks.test",
				WorkspaceID:          "987654321",
				HasClientCredentials: true,
			},
			{
				Name:        "blank-auth",
				Host:        "https://blank-auth.cloud.databricks.test",
				WorkspaceID: "111222333",
			},
			{
				Name:        "account",
				Host:        "https://accounts.cloud.databricks.test",
				AccountID:   "account-id",
				WorkspaceID: "444555666",
				AuthType:    authTypeDatabricksCLI,
			},
		},
	}

	for _, tc := range []struct {
		registryHost string
		wantError    string
	}{
		{"123456789.container.us-west-2.cloud.databricks.com", "requires a profile created by databricks auth login"},
		{"987654321.container.us-west-2.cloud.databricks.com", "requires a profile created by databricks auth login"},
		{"111222333.container.us-west-2.cloud.databricks.com", "requires a profile created by databricks auth login"},
		{"444555666.container.us-west-2.cloud.databricks.com", "does not target a workspace"},
	} {
		t.Run(tc.registryHost, func(t *testing.T) {
			cmd := &cobra.Command{Use: "token"}
			cmd.SetContext(ctx)
			cmd.SetIn(strings.NewReader(tc.registryHost + "\n"))

			err := writeDockerTokenOutput(ctx, cmd, loadTokenArgs{
				authArguments: &auth.AuthArguments{},
				profiler:      profiler,
			}, func(context.Context, loadTokenArgs) (*oauth2.Token, error) {
				t.Fatal("loadToken should not be called")
				return nil, nil
			}, dockercredentials.RegistryHost)
			require.ErrorContains(t, err, tc.wantError)
		})
	}
}

func TestWriteTokenOutput(t *testing.T) {
	token := &oauth2.Token{
		AccessToken: "my-access-token",
		TokenType:   "Bearer",
	}

	t.Run("json mode", func(t *testing.T) {
		var buf bytes.Buffer
		err := writeTokenOutput(&buf, token, false)
		assert.NoError(t, err)

		raw, err := json.MarshalIndent(token, "", "  ")
		assert.NoError(t, err)
		assert.Equal(t, string(raw), buf.String())
	})

	t.Run("text mode", func(t *testing.T) {
		var buf bytes.Buffer
		err := writeTokenOutput(&buf, token, true)
		assert.NoError(t, err)
		assert.Equal(t, "my-access-token\n", buf.String())
	})
}
