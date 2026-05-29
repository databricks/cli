package auth

import (
	"testing"

	"github.com/databricks/cli/libs/env"
	sdkconfig "github.com/databricks/databricks-sdk-go/config"
	"github.com/stretchr/testify/assert"
)

func TestNormalizeDatabricksConfigFromEnv(t *testing.T) {
	tests := []struct {
		name            string
		host            string
		envWorkspaceID  string
		envAccountID    string
		cfgInHost       string
		wantHost        string
		wantWorkspaceID string
		wantAccountID   string
	}{
		{
			name:            "spog url promotes workspace id",
			host:            "https://acme.databricks.net/?o=12345",
			wantHost:        "https://acme.databricks.net",
			wantWorkspaceID: "12345",
		},
		{
			name:            "spog url with account id",
			host:            "https://acme.databricks.net/?a=abc&o=12345",
			wantHost:        "https://acme.databricks.net",
			wantWorkspaceID: "12345",
			wantAccountID:   "abc",
		},
		{
			name: "host without query is a no-op",
			host: "https://acme.databricks.net",
		},
		{
			name:            "env workspace id wins over query param",
			host:            "https://acme.databricks.net/?o=12345",
			envWorkspaceID:  "99999",
			wantHost:        "https://acme.databricks.net",
			wantWorkspaceID: "",
		},
		{
			name:      "cfg host already set leaves env alone",
			host:      "https://other.databricks.net/?o=12345",
			cfgInHost: "https://acme.databricks.net",
			wantHost:  "https://acme.databricks.net",
		},
		{
			name: "no host env is a no-op",
		},
		{
			name:            "non-numeric o is passed through, host trailing slash trimmed",
			host:            "https://acme.databricks.net/?o=notanumber",
			wantHost:        "https://acme.databricks.net",
			wantWorkspaceID: "notanumber",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := env.Set(t.Context(), "DATABRICKS_HOST", tt.host)
			ctx = env.Set(ctx, "DATABRICKS_WORKSPACE_ID", tt.envWorkspaceID)
			ctx = env.Set(ctx, "DATABRICKS_ACCOUNT_ID", tt.envAccountID)

			cfg := &sdkconfig.Config{Host: tt.cfgInHost}
			NormalizeDatabricksConfigFromEnv(ctx, cfg)

			assert.Equal(t, tt.wantHost, cfg.Host)
			assert.Equal(t, tt.wantWorkspaceID, cfg.WorkspaceID)
			assert.Equal(t, tt.wantAccountID, cfg.AccountID)
		})
	}
}
