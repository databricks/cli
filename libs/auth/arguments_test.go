package auth

import (
	"testing"

	"github.com/databricks/databricks-sdk-go/credentials/u2m"
	"github.com/stretchr/testify/assert"
)

func TestToOAuthArgument(t *testing.T) {
	tests := []struct {
		name         string
		args         AuthArguments
		wantHost     string
		wantCacheKey string
		wantError    bool
	}{
		{
			name: "workspace with no scheme",
			args: AuthArguments{
				Host: "my-workspace.cloud.databricks.com",
			},
			wantHost:     "https://my-workspace.cloud.databricks.com",
			wantCacheKey: "https://my-workspace.cloud.databricks.com",
		},
		{
			name: "workspace with https",
			args: AuthArguments{
				Host: "https://my-workspace.cloud.databricks.com",
			},
			wantHost:     "https://my-workspace.cloud.databricks.com",
			wantCacheKey: "https://my-workspace.cloud.databricks.com",
		},
		{
			name: "workspace with profile uses profile-based cache key",
			args: AuthArguments{
				Host:    "https://my-workspace.cloud.databricks.com",
				Profile: "my-profile",
			},
			wantHost:     "https://my-workspace.cloud.databricks.com",
			wantCacheKey: "my-profile",
		},
		{
			name: "account with no scheme",
			args: AuthArguments{
				Host:      "accounts.cloud.databricks.com",
				AccountID: "123456789",
			},
			wantHost:     "https://accounts.cloud.databricks.com",
			wantCacheKey: "https://accounts.cloud.databricks.com/oidc/accounts/123456789",
		},
		{
			name: "account with https",
			args: AuthArguments{
				Host:      "https://accounts.cloud.databricks.com",
				AccountID: "123456789",
			},
			wantHost:     "https://accounts.cloud.databricks.com",
			wantCacheKey: "https://accounts.cloud.databricks.com/oidc/accounts/123456789",
		},
		{
			name: "account with profile uses profile-based cache key",
			args: AuthArguments{
				Host:      "https://accounts.cloud.databricks.com",
				AccountID: "123456789",
				Profile:   "my-account-profile",
			},
			wantHost:     "https://accounts.cloud.databricks.com",
			wantCacheKey: "my-account-profile",
		},
		{
			name: "workspace with query parameter",
			args: AuthArguments{
				Host: "https://my-workspace.cloud.databricks.com?o=123456789",
			},
			wantHost:     "https://my-workspace.cloud.databricks.com",
			wantCacheKey: "https://my-workspace.cloud.databricks.com",
		},
		{
			name: "workspace with query parameter and path",
			args: AuthArguments{
				Host: "https://my-workspace.cloud.databricks.com/path?o=123456789",
			},
			wantHost:     "https://my-workspace.cloud.databricks.com",
			wantCacheKey: "https://my-workspace.cloud.databricks.com",
		},
		{
			name: "unified host with account ID only",
			args: AuthArguments{
				Host:          "https://unified.cloud.databricks.com",
				AccountID:     "123456789",
				IsUnifiedHost: true,
			},
			wantHost:     "https://unified.cloud.databricks.com",
			wantCacheKey: "https://unified.cloud.databricks.com/oidc/accounts/123456789",
		},
		{
			name: "unified host with both account ID and workspace ID",
			args: AuthArguments{
				Host:          "https://unified.cloud.databricks.com",
				AccountID:     "123456789",
				WorkspaceID:   "123456789",
				IsUnifiedHost: true,
			},
			wantHost:     "https://unified.cloud.databricks.com",
			wantCacheKey: "https://unified.cloud.databricks.com/oidc/accounts/123456789",
		},
		{
			name: "unified host with profile uses profile-based cache key",
			args: AuthArguments{
				Host:          "https://unified.cloud.databricks.com",
				AccountID:     "123456789",
				IsUnifiedHost: true,
				Profile:       "my-unified-profile",
			},
			wantHost:     "https://unified.cloud.databricks.com",
			wantCacheKey: "my-unified-profile",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.args.ToOAuthArgument()
			if tt.wantError {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.wantCacheKey, got.GetCacheKey())

			// Check if we got the right type of argument and verify the hostname
			if tt.args.IsUnifiedHost {
				arg, ok := got.(u2m.UnifiedOAuthArgument)
				assert.True(t, ok, "expected UnifiedOAuthArgument for unified host")
				assert.Equal(t, tt.wantHost, arg.GetHost())
			} else if tt.args.AccountID != "" {
				arg, ok := got.(u2m.AccountOAuthArgument)
				assert.True(t, ok, "expected AccountOAuthArgument for account host")
				assert.Equal(t, tt.wantHost, arg.GetAccountHost())
			} else {
				arg, ok := got.(u2m.WorkspaceOAuthArgument)
				assert.True(t, ok, "expected WorkspaceOAuthArgument for workspace host")
				assert.Equal(t, tt.wantHost, arg.GetWorkspaceHost())
			}
		})
	}
}
