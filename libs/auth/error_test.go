package auth

import (
	"errors"
	"strings"
	"testing"

	"github.com/databricks/cli/libs/apierrs"

	"github.com/databricks/databricks-sdk-go/config"
	"github.com/databricks/sdk-go/core/apierr"
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
	result := EnrichAuthError(t.Context(), cfg, original)
	assert.Equal(t, original, result)
}

func TestEnrichAuthError_NonAuthStatusCode(t *testing.T) {
	cfg := &config.Config{Profile: "test", Host: "https://example.com"}
	original := apierr.FromHTTPError(404, nil, []byte(`{"message":"not found"}`))
	result := EnrichAuthError(t.Context(), cfg, original)
	assert.Equal(t, original, result)
}

func TestEnrichAuthError_PreservesOriginalError(t *testing.T) {
	cfg := &config.Config{Host: "https://example.com", AuthType: "pat"}
	original := apierr.FromHTTPError(403, nil, []byte(`{"error_code":"PERMISSION_DENIED","message":"User does not have permission"}`))
	result := EnrichAuthError(t.Context(), cfg, original)

	var unwrapped *apierr.APIError
	require.ErrorAs(t, result, &unwrapped)
	assert.Equal(t, 403, unwrapped.HTTPStatusCode())
	assert.Equal(t, "PERMISSION_DENIED", apierrs.ErrorCode(unwrapped))
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
				"\n  - Regenerate your access token or run: databricks auth login --profile dev" +
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
				"\n  - Check your username/password or run: databricks auth login --profile basic-profile" +
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
				"\n  - Consider setting up a profile: databricks auth login --profile <name>",
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
				"\n  - Consider setting up a profile: databricks auth login --profile <name>",
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
				"\n  - Consider setting up a profile: databricks auth login --profile <name>",
		},
		{
			name: "401 with unified host (resolved DiscoveryURL) and no profile",
			cfg: &config.Config{
				Host:         "https://unified.cloud.databricks.com",
				AccountID:    "acc-123",
				WorkspaceID:  "ws-456",
				AuthType:     AuthTypeDatabricksCli,
				DiscoveryURL: "https://unified.cloud.databricks.com/oidc/accounts/acc-123/.well-known/oauth-authorization-server",
			},
			statusCode: 401,
			wantMsg: "test error message\n" +
				"\nHost:      https://unified.cloud.databricks.com" +
				"\nAuth type: OAuth (databricks-cli)" +
				"\n\nNext steps:" +
				"\n  - Re-authenticate: databricks auth login --host https://unified.cloud.databricks.com --account-id acc-123" +
				"\n  - Check your identity: databricks auth describe" +
				"\n  - Consider setting up a profile: databricks auth login --profile <name>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			original := apierr.FromHTTPError(tt.statusCode, nil, []byte(`{"error_code":"TEST_ERROR","message":"test error message"}`))

			result := EnrichAuthError(t.Context(), tt.cfg, original)
			// The enrichment is appended to the original error's own text.
			assert.Equal(t, strings.Replace(tt.wantMsg, "test error message", original.Error(), 1), result.Error())
		})
	}
}

func TestAppendAccountHostHint(t *testing.T) {
	tests := []struct {
		name string
		cfg  *config.Config
		want string
	}{
		{
			name: "account console host with profile",
			cfg:  &config.Config{Host: "https://accounts.test", Profile: "acc"},
			want: "base error\n\n" +
				"Note: profile \"acc\" points to a Databricks account console host (https://accounts.test), which serves only account-level APIs.\n" +
				"Workspace commands need a workspace host: run `databricks auth login --host https://<workspace-url>`, or use `databricks account ...` commands with this profile",
		},
		{
			name: "account console host without profile",
			cfg:  &config.Config{Host: "https://accounts.test"},
			want: "base error\n\n" +
				"Note: this configuration points to a Databricks account console host (https://accounts.test), which serves only account-level APIs.\n" +
				"Workspace commands need a workspace host: run `databricks auth login --host https://<workspace-url>`, or use `databricks account ...` commands",
		},
		{
			name: "workspace host is left unchanged",
			cfg:  &config.Config{Host: "https://adb-123.test", Profile: "ws"},
			want: "base error",
		},
		{
			name: "empty host is left unchanged",
			cfg:  &config.Config{},
			want: "base error",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := AppendAccountHostHint(tt.cfg, errors.New("base error"))
			assert.Equal(t, tt.want, result.Error())
		})
	}
}

func TestAppendAccountHostHintPreservesErrorChain(t *testing.T) {
	cfg := &config.Config{Host: "https://accounts.test", Profile: "acc"}
	original := apierr.FromHTTPError(400, nil, []byte(`{"message":"Unable to load OAuth Config"}`))

	result := AppendAccountHostHint(cfg, original)

	var unwrapped *apierr.APIError
	require.ErrorAs(t, result, &unwrapped)
	assert.Equal(t, 400, unwrapped.HTTPStatusCode())
}

func TestAppendAccountHostHintComposesWithEnrichAuthError(t *testing.T) {
	cfg := &config.Config{Host: "https://accounts.test", Profile: "acc", AuthType: AuthTypePat}
	original := apierr.FromHTTPError(403, nil, []byte(`{"message":"permission denied"}`))

	// Same composition order as cmd/root.Execute: enrichment first, hint last.
	result := AppendAccountHostHint(cfg, EnrichAuthError(t.Context(), cfg, original))

	msg := result.Error()
	assert.Contains(t, msg, "permission denied")
	assert.Contains(t, msg, "Next steps:")
	assert.Contains(t, msg, "Note: profile \"acc\" points to a Databricks account console host")
	assert.Less(t, strings.Index(msg, "Next steps:"), strings.Index(msg, "Note:"))
}
