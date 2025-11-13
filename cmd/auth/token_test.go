package auth

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/databrickscfg/profile"
	"github.com/databricks/databricks-sdk-go/credentials/u2m"
	"github.com/databricks/databricks-sdk-go/httpclient/fixtures"
	"github.com/stretchr/testify/assert"
	"golang.org/x/oauth2"
)

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

func TestLoadTokenHappyPath(t *testing.T) {
	profiler := profile.InMemoryProfiler{
		Profiles: profile.Profiles{
			{
				Name:      "active",
				Host:      "https://accounts.cloud.databricks.com",
				AccountID: "active",
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
		},
	}
	args := loadTokenArgs{
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
	}

	token, err := loadToken(context.Background(), args)
	assert.NoError(t, err)
	assert.Equal(t, "new-access-token", token.AccessToken)
	assert.Equal(t, "Bearer", token.TokenType)
}
