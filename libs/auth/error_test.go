package auth

import (
	"context"
	"errors"
	"testing"

	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildDescribeCommand(t *testing.T) {
	assert.Equal(t,
		"databricks auth describe --profile my-profile",
		BuildDescribeCommand(&config.Config{Profile: "my-profile"}),
	)
	assert.Equal(t,
		"databricks auth describe",
		BuildDescribeCommand(&config.Config{Host: "https://example.com"}),
	)
	assert.Equal(t,
		"databricks auth describe",
		BuildDescribeCommand(&config.Config{}),
	)
}

func TestAuthTypeDisplayName(t *testing.T) {
	assert.Equal(t, "Personal Access Token (pat)", AuthTypeDisplayName("pat"))
	assert.Equal(t, "OAuth (databricks-cli)", AuthTypeDisplayName("databricks-cli"))
	assert.Equal(t, "Azure CLI (azure-cli)", AuthTypeDisplayName("azure-cli"))
	assert.Equal(t, "some-future-auth", AuthTypeDisplayName("some-future-auth"))
}

func TestEnrichAuthError_NonAPIError(t *testing.T) {
	cfg := &config.Config{Profile: "test", Host: "https://example.com"}
	original := errors.New("some random error")
	result := EnrichAuthError(context.Background(), cfg, original)
	assert.Equal(t, original, result)
}

func TestEnrichAuthError_NonAuthStatusCode(t *testing.T) {
	cfg := &config.Config{Profile: "test", Host: "https://example.com"}
	original := &apierr.APIError{StatusCode: 404, Message: "not found"}
	result := EnrichAuthError(context.Background(), cfg, original)
	assert.Equal(t, original, result)
}

func TestEnrichAuthError_PreservesOriginalError(t *testing.T) {
	cfg := &config.Config{
		Host:     "https://example.com",
		AuthType: "pat",
	}
	original := &apierr.APIError{
		StatusCode: 403,
		ErrorCode:  "PERMISSION_DENIED",
		Message:    "User does not have permission",
	}
	result := EnrichAuthError(context.Background(), cfg, original)

	var unwrapped *apierr.APIError
	require.ErrorAs(t, result, &unwrapped)
	assert.Equal(t, 403, unwrapped.StatusCode)
	assert.Equal(t, "PERMISSION_DENIED", unwrapped.ErrorCode)
}

func TestEnrichAuthError(t *testing.T) {
	tests := []struct {
		name       string
		cfg        *config.Config
		statusCode int
		contains   []string
		notContain []string
	}{
		{
			name: "401 with profile and databricks-cli auth",
			cfg: &config.Config{
				Host:     "https://my-workspace.cloud.databricks.com",
				Profile:  "dev",
				AuthType: AuthTypeDatabricksCli,
			},
			statusCode: 401,
			contains: []string{
				"Profile:   dev",
				"Host:      https://my-workspace.cloud.databricks.com",
				"Auth type: OAuth (databricks-cli)",
				"Re-authenticate: databricks auth login --profile dev",
				"Check your identity: databricks auth describe --profile dev",
			},
			notContain: []string{
				"Consider configuring a profile",
			},
		},
		{
			name: "401 with profile and pat auth",
			cfg: &config.Config{
				Host:     "https://my-workspace.cloud.databricks.com",
				Profile:  "dev",
				AuthType: AuthTypePat,
			},
			statusCode: 401,
			contains: []string{
				"Profile:   dev",
				"Auth type: Personal Access Token (pat)",
				"Regenerate your access token or run: databricks configure --profile dev",
				"Check your identity: databricks auth describe --profile dev",
			},
		},
		{
			name: "401 with profile and azure-cli auth",
			cfg: &config.Config{
				Host:     "https://adb-123.azuredatabricks.net",
				Profile:  "azure",
				AuthType: AuthTypeAzureCli,
			},
			statusCode: 401,
			contains: []string{
				"Auth type: Azure CLI (azure-cli)",
				"Re-authenticate with Azure: az login",
			},
		},
		{
			name: "401 with profile and oauth-m2m auth",
			cfg: &config.Config{
				Host:     "https://my-workspace.cloud.databricks.com",
				Profile:  "sp",
				AuthType: AuthTypeOAuthM2M,
			},
			statusCode: 401,
			contains: []string{
				"Auth type: OAuth Machine-to-Machine (oauth-m2m)",
				"Check your service principal client ID and secret",
			},
		},
		{
			name: "401 with profile and basic auth",
			cfg: &config.Config{
				Host:     "https://my-workspace.cloud.databricks.com",
				Profile:  "basic-profile",
				AuthType: AuthTypeBasic,
			},
			statusCode: 401,
			contains: []string{
				"Auth type: Basic (username/password)",
				"Check your username/password or run: databricks configure --profile basic-profile",
			},
		},
		{
			name: "401 with unknown auth type falls back to raw name",
			cfg: &config.Config{
				Host:     "https://my-workspace.cloud.databricks.com",
				Profile:  "dev",
				AuthType: "some-future-auth",
			},
			statusCode: 401,
			contains: []string{
				"Auth type: some-future-auth",
				"Check your authentication credentials",
			},
		},
		{
			name: "403 with profile",
			cfg: &config.Config{
				Host:     "https://my-workspace.cloud.databricks.com",
				Profile:  "dev",
				AuthType: AuthTypePat,
			},
			statusCode: 403,
			contains: []string{
				"Profile:   dev",
				"Verify you have the required permissions for this operation",
				"Check your identity: databricks auth describe --profile dev",
			},
			notContain: []string{
				"Re-authenticate",
				"Regenerate",
				"Consider configuring a profile",
			},
		},
		{
			name: "401 without profile (env var auth)",
			cfg: &config.Config{
				Host:     "https://my-workspace.cloud.databricks.com",
				AuthType: AuthTypePat,
			},
			statusCode: 401,
			contains: []string{
				"Host:      https://my-workspace.cloud.databricks.com",
				"Auth type: Personal Access Token (pat)",
				"Regenerate your access token",
				"Check your identity: databricks auth describe",
				"Consider configuring a profile: databricks configure --profile <name>",
			},
			notContain: []string{
				"Profile:",
				"--host",
			},
		},
		{
			name: "403 without profile (env var auth)",
			cfg: &config.Config{
				Host:     "https://my-workspace.cloud.databricks.com",
				AuthType: AuthTypePat,
			},
			statusCode: 403,
			contains: []string{
				"Verify you have the required permissions for this operation",
				"Consider configuring a profile: databricks configure --profile <name>",
			},
			notContain: []string{
				"Profile:",
			},
		},
		{
			name: "401 with account host and no profile",
			cfg: &config.Config{
				Host:      "https://accounts.cloud.databricks.com",
				AccountID: "abc123",
				AuthType:  AuthTypeDatabricksCli,
			},
			statusCode: 401,
			contains: []string{
				"Re-authenticate: databricks auth login --host https://accounts.cloud.databricks.com --account-id abc123",
				"Check your identity: databricks auth describe",
				"Consider configuring a profile",
			},
			notContain: []string{
				"describe --host",
			},
		},
		{
			name: "401 with unified host includes workspace-id in login",
			cfg: &config.Config{
				Host:                       "https://unified.cloud.databricks.com",
				AccountID:                  "acc-123",
				WorkspaceID:                "ws-456",
				AuthType:                   AuthTypeDatabricksCli,
				Experimental_IsUnifiedHost: true,
			},
			statusCode: 401,
			contains: []string{
				"Re-authenticate: databricks auth login --host https://unified.cloud.databricks.com --account-id acc-123 --experimental-is-unified-host --workspace-id ws-456",
				"Check your identity: databricks auth describe",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			original := &apierr.APIError{
				StatusCode: tt.statusCode,
				ErrorCode:  "TEST_ERROR",
				Message:    "test error message",
			}

			result := EnrichAuthError(context.Background(), tt.cfg, original)
			msg := result.Error()

			// Original error message is preserved.
			assert.Contains(t, msg, "test error message")

			for _, s := range tt.contains {
				assert.Contains(t, msg, s)
			}
			for _, s := range tt.notContain {
				assert.NotContains(t, msg, s)
			}
		})
	}
}
