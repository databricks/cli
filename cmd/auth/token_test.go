package auth_test

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/databricks/cli/cmd"
	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/auth/cache"
	"github.com/databricks/cli/libs/databrickscfg/profile"
	"github.com/databricks/databricks-sdk-go/httpclient"
	"github.com/databricks/databricks-sdk-go/httpclient/fixtures"
	"github.com/spf13/cobra"
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

func validateToken(t *testing.T, resp string) {
	res := map[string]string{}
	err := json.Unmarshal([]byte(resp), &res)
	assert.NoError(t, err)
	assert.Equal(t, "new-access-token", res["access_token"])
	assert.Equal(t, "Bearer", res["token_type"])
}

func getContextForTest(f fixtures.HTTPFixture) context.Context {
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
	client := httpclient.NewApiClient(httpclient.ClientConfig{
		Transport: fixtures.SliceTransport{f},
	})
	ctx := profile.WithProfiler(context.Background(), profiler)
	ctx = cache.WithTokenCache(ctx, tokenCache)
	ctx = auth.WithApiClientForOAuth(ctx, client)
	return ctx
}

func getCobraCmdForTest(f fixtures.HTTPFixture) (*cobra.Command, *bytes.Buffer) {
	ctx := getContextForTest(f)
	c := cmd.New(ctx)
	output := &bytes.Buffer{}
	c.SetOut(output)
	return c, output
}

func TestTokenCmdWithProfilePrintsHelpfulLoginMessageOnRefreshFailure(t *testing.T) {
	cmd, output := getCobraCmdForTest(refreshFailureTokenResponse)
	cmd.SetArgs([]string{"auth", "token", "--profile", "expired"})
	err := cmd.Execute()

	out := output.String()
	assert.Empty(t, out)
	assert.ErrorContains(t, err, "a new access token could not be retrieved because the refresh token is invalid. To reauthenticate, run ")
	assert.ErrorContains(t, err, "auth login --profile expired")
}

func TestTokenCmdWithHostPrintsHelpfulLoginMessageOnRefreshFailure(t *testing.T) {
	cmd, output := getCobraCmdForTest(refreshFailureTokenResponse)
	cmd.SetArgs([]string{"auth", "token", "--host", "https://accounts.cloud.databricks.com", "--account-id", "expired"})
	err := cmd.Execute()

	out := output.String()
	assert.Empty(t, out)
	assert.ErrorContains(t, err, "a new access token could not be retrieved because the refresh token is invalid. To reauthenticate, run ")
	assert.ErrorContains(t, err, "auth login --host https://accounts.cloud.databricks.com --account-id expired")
}

func TestTokenCmdInvalidResponse(t *testing.T) {
	cmd, output := getCobraCmdForTest(refreshFailureInvalidResponse)
	cmd.SetArgs([]string{"auth", "token", "--profile", "active"})
	err := cmd.Execute()

	out := output.String()
	assert.Empty(t, out)
	assert.ErrorContains(t, err, "unexpected parsing token response: invalid character 'N' looking for beginning of value. Try logging in again with ")
	assert.ErrorContains(t, err, "auth login --profile active` before retrying. If this fails, please report this issue to the Databricks CLI maintainers at https://github.com/databricks/cli/issues/new")
}

func TestTokenCmdOtherErrorResponse(t *testing.T) {
	cmd, output := getCobraCmdForTest(refreshFailureOtherError)
	cmd.SetArgs([]string{"auth", "token", "--profile", "active"})
	err := cmd.Execute()

	out := output.String()
	assert.Empty(t, out)
	assert.ErrorContains(t, err, "unexpected error refreshing token: Databricks is down. Try logging in again with ")
	assert.ErrorContains(t, err, "auth login --profile active` before retrying. If this fails, please report this issue to the Databricks CLI maintainers at https://github.com/databricks/cli/issues/new")
}

func TestTokenCmdWithProfileSuccess(t *testing.T) {
	cmd, output := getCobraCmdForTest(refreshSuccessTokenResponse)
	cmd.SetArgs([]string{"auth", "token", "--profile", "active"})
	err := cmd.Execute()

	out := output.String()
	validateToken(t, out)
	assert.NoError(t, err)
}

func TestTokenCmdWithHostSuccess(t *testing.T) {
	cmd, output := getCobraCmdForTest(refreshSuccessTokenResponse)
	cmd.SetArgs([]string{"auth", "token", "--host", "https://accounts.cloud.databricks.com", "--account-id", "expired"})
	err := cmd.Execute()

	out := output.String()
	validateToken(t, out)
	assert.NoError(t, err)
}
