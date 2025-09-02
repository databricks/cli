package auth

import (
	"testing"

	"github.com/databricks/databricks-sdk-go/credentials/u2m"
	"github.com/stretchr/testify/assert"
)

func TestToOAuthArgument(t *testing.T) {
	tests := []struct {
		name      string
		args      AuthArguments
		wantHost  string
		wantError bool
	}{
		{
			name: "workspace with no scheme",
			args: AuthArguments{
				Host: "my-workspace.cloud.databricks.com",
			},
			wantHost: "https://my-workspace.cloud.databricks.com",
		},
		{
			name: "workspace with https",
			args: AuthArguments{
				Host: "https://my-workspace.cloud.databricks.com",
			},
			wantHost: "https://my-workspace.cloud.databricks.com",
		},
		{
			name: "account with no scheme",
			args: AuthArguments{
				Host:      "accounts.cloud.databricks.com",
				AccountID: "123456789",
			},
			wantHost: "https://accounts.cloud.databricks.com",
		},
		{
			name: "account with https",
			args: AuthArguments{
				Host:      "https://accounts.cloud.databricks.com",
				AccountID: "123456789",
			},
			wantHost: "https://accounts.cloud.databricks.com",
		},
		{
			name: "workspace with query parameter",
			args: AuthArguments{
				Host: "https://my-workspace.cloud.databricks.com?o=123456789",
			},
			wantHost: "https://my-workspace.cloud.databricks.com",
		},
		{
			name: "workspace with query parameter and path",
			args: AuthArguments{
				Host: "https://my-workspace.cloud.databricks.com/path?o=123456789",
			},
			wantHost: "https://my-workspace.cloud.databricks.com",
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

			// Check if we got the right type of argument and verify the hostname
			if tt.args.AccountID != "" {
				arg, ok := got.(u2m.AccountOAuthArgument)
				assert.True(t, ok, "expected AccountOAuthArgument for account ID")
				assert.Equal(t, tt.wantHost, arg.GetAccountHost())
			} else {
				arg, ok := got.(u2m.WorkspaceOAuthArgument)
				assert.True(t, ok, "expected WorkspaceOAuthArgument for workspace")
				assert.Equal(t, tt.wantHost, arg.GetWorkspaceHost())
			}
		})
	}
}
