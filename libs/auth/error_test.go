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
	assert.Equal(t, "OAuth Machine-to-Machine (oauth-m2m)", AuthTypeDisplayName("oauth-m2m"))
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
	cfg := &config.Config{Host: "https://example.com", AuthType: "pat"}
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
		wantMsg    string
	}{
		{
			name: "401 with profile and databricks-cli auth",
			cfg: &config.Config{
				Host:     "https://my-workspace.cloud.databricks.com",
				Profile:  "dev",
				AuthType: AuthTypeDatabricksCli,
			},
			statusCode: 401,
			wantMsg: "test error message\n" +
				"\nProfile:   dev" +
				"\nHost:      https://my-workspace.cloud.databricks.com" +
				"\nAuth type: OAuth (databricks-cli)" +
				"\n\nNext steps:" +
				"\n  - Re-authenticate: databricks auth login --profile dev" +
				"\n  - Check your identity: databricks auth describe --profile dev",
		},
		{
			name: "401 with profile and pat auth",
			cfg: &config.Config{
				Host:     "https://my-workspace.cloud.databricks.com",
				Profile:  "dev",
				AuthType: AuthTypePat,
			},
			statusCode: 401,
			wantMsg: "test error message\n" +
				"\nProfile:   dev" +
				"\nHost:      https://my-workspace.cloud.databricks.com" +
				"\nAuth type: Personal Access Token (pat)" +
				"\n\nNext steps:" +
				"\n  - Regenerate your access token or run: databricks configure --profile dev" +
				"\n  - Check your identity: databricks auth describe --profile dev",
		},
		{
			name: "401 with profile and azure-cli auth",
			cfg: &config.Config{
				Host:     "https://adb-123.azuredatabricks.net",
				Profile:  "azure",
				AuthType: AuthTypeAzureCli,
			},
			statusCode: 401,
			wantMsg: "test error message\n" +
				"\nProfile:   azure" +
				"\nHost:      https://adb-123.azuredatabricks.net" +
				"\nAuth type: Azure CLI (azure-cli)" +
				"\n\nNext steps:" +
				"\n  - Re-authenticate with Azure: az login" +
				"\n  - Check your identity: databricks auth describe --profile azure",
		},
		{
			name: "401 with profile and oauth-m2m auth",
			cfg: &config.Config{
				Host:     "https://my-workspace.cloud.databricks.com",
				Profile:  "sp",
				AuthType: AuthTypeOAuthM2M,
			},
			statusCode: 401,
			wantMsg: "test error message\n" +
				"\nProfile:   sp" +
				"\nHost:      https://my-workspace.cloud.databricks.com" +
				"\nAuth type: OAuth Machine-to-Machine (oauth-m2m)" +
				"\n\nNext steps:" +
				"\n  - Check your service principal client ID and secret" +
				"\n  - Check your identity: databricks auth describe --profile sp",
		},
		{
			name: "401 with profile and basic auth",
			cfg: &config.Config{
				Host:     "https://my-workspace.cloud.databricks.com",
				Profile:  "basic-profile",
				AuthType: AuthTypeBasic,
			},
			statusCode: 401,
			wantMsg: "test error message\n" +
				"\nProfile:   basic-profile" +
				"\nHost:      https://my-workspace.cloud.databricks.com" +
				"\nAuth type: Basic" +
				"\n\nNext steps:" +
				"\n  - Check your username/password or run: databricks configure --profile basic-profile" +
				"\n  - Check your identity: databricks auth describe --profile basic-profile",
		},
		{
			name: "401 with unknown auth type falls back to raw name",
			cfg: &config.Config{
				Host:     "https://my-workspace.cloud.databricks.com",
				Profile:  "dev",
				AuthType: "some-future-auth",
			},
			statusCode: 401,
			wantMsg: "test error message\n" +
				"\nProfile:   dev" +
				"\nHost:      https://my-workspace.cloud.databricks.com" +
				"\nAuth type: some-future-auth" +
				"\n\nNext steps:" +
				"\n  - Check your authentication credentials" +
				"\n  - Check your identity: databricks auth describe --profile dev",
		},
		{
			name: "403 with profile",
			cfg: &config.Config{
				Host:     "https://my-workspace.cloud.databricks.com",
				Profile:  "dev",
				AuthType: AuthTypePat,
			},
			statusCode: 403,
			wantMsg: "test error message\n" +
				"\nProfile:   dev" +
				"\nHost:      https://my-workspace.cloud.databricks.com" +
				"\nAuth type: Personal Access Token (pat)" +
				"\n\nNext steps:" +
				"\n  - Verify you have the required permissions for this operation" +
				"\n  - Check your identity: databricks auth describe --profile dev",
		},
		{
			name: "401 without profile (env var auth)",
			cfg: &config.Config{
				Host:     "https://my-workspace.cloud.databricks.com",
				AuthType: AuthTypePat,
			},
			statusCode: 401,
			wantMsg: "test error message\n" +
				"\nHost:      https://my-workspace.cloud.databricks.com" +
				"\nAuth type: Personal Access Token (pat)" +
				"\n\nNext steps:" +
				"\n  - Regenerate your access token" +
				"\n  - Check your identity: databricks auth describe" +
				"\n  - Consider configuring a profile: databricks configure --profile <name>",
		},
		{
			name: "403 without profile (env var auth)",
			cfg: &config.Config{
				Host:     "https://my-workspace.cloud.databricks.com",
				AuthType: AuthTypePat,
			},
			statusCode: 403,
			wantMsg: "test error message\n" +
				"\nHost:      https://my-workspace.cloud.databricks.com" +
				"\nAuth type: Personal Access Token (pat)" +
				"\n\nNext steps:" +
				"\n  - Verify you have the required permissions for this operation" +
				"\n  - Check your identity: databricks auth describe" +
				"\n  - Consider configuring a profile: databricks configure --profile <name>",
		},
		{
			name: "401 with account host and no profile",
			cfg: &config.Config{
				Host:      "https://accounts.cloud.databricks.com",
				AccountID: "abc123",
				AuthType:  AuthTypeDatabricksCli,
			},
			statusCode: 401,
			wantMsg: "test error message\n" +
				"\nHost:      https://accounts.cloud.databricks.com" +
				"\nAuth type: OAuth (databricks-cli)" +
				"\n\nNext steps:" +
				"\n  - Re-authenticate: databricks auth login --host https://accounts.cloud.databricks.com --account-id abc123" +
				"\n  - Check your identity: databricks auth describe" +
				"\n  - Consider configuring a profile: databricks configure --profile <name>",
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
			wantMsg: "test error message\n" +
				"\nHost:      https://unified.cloud.databricks.com" +
				"\nAuth type: OAuth (databricks-cli)" +
				"\n\nNext steps:" +
				"\n  - Re-authenticate: databricks auth login --host https://unified.cloud.databricks.com --account-id acc-123 --experimental-is-unified-host --workspace-id ws-456" +
				"\n  - Check your identity: databricks auth describe" +
				"\n  - Consider configuring a profile: databricks configure --profile <name>",
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
			assert.Equal(t, tt.wantMsg, result.Error())
		})
	}
}
