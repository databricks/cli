package auth

import (
	"bytes"
	"context"
	"testing"
	"time"

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

var refreshSuccessTokenResponse = fixtures.HTTPFixture{
	MatchAny: true,
	Status:   200,
	Response: map[string]string{
		"access_token": "new-access-token",
		"token_type":   "Bearer",
		"expires_in":   "3600",
	},
}

func persistentAuthForTest(f fixtures.HTTPFixture) *auth.PersistentAuth {
	pa := &auth.PersistentAuth{}
	client := httpclient.NewApiClient(httpclient.ClientConfig{
		Transport: fixtures.SliceTransport{f},
	})
	pa.SetTokenCache(&cache.InMemoryTokenCache{
		Tokens: map[string]*oauth2.Token{
			"https://accounts.cloud.databricks.com/oidc/accounts/expired": {
				RefreshToken: "expired",
			},
			"https://accounts.cloud.databricks.com/oidc/accounts/active": {
				RefreshToken: "active",
				Expiry:       time.Now().Add(1 * time.Hour), // Hopefully unit tests don't take an hour to run
			},
		},
	})
	pa.SetApiClient(client)
	return pa
}

func getContextForTest() context.Context {
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
	ctx := profile.WithProfiler(context.Background(), profiler)
	return ctx
}

func getCobraCmdForTest(args []string) (*cobra.Command, *bytes.Buffer) {
	cmd := &cobra.Command{}
	cmd.SetContext(getContextForTest())
	cmd.SetArgs(args)
	output := &bytes.Buffer{}
	cmd.SetOut(output)
	return cmd, output
}

func TestTokenCmdPrintsHelpfulLoginMessageOnRefreshFailure(t *testing.T) {
	cmd, output := getCobraCmdForTest([]string{"auth", "token", "--profile", "expired"})

	pa := persistentAuthForTest(refreshFailureTokenResponse)
	tokenCmd := newTokenCommand(pa)
	err := tokenCmd.RunE(cmd, []string{})

	assert.ErrorContains(t, err, "a new access token could not be retrieved because the refresh token is invalid. To reauthenticate, run ")
	assert.ErrorContains(t, err, "auth login --host https://accounts.cloud.databricks.com --account-id abc-1234")
	out := output.String()
	assert.Empty(t, out)
}
