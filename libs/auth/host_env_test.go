package auth

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizeDatabricksHostEnv(t *testing.T) {
	tests := []struct {
		name            string
		host            string
		workspaceID     string
		accountID       string
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
			name:     "host without query is left alone",
			host:     "https://acme.databricks.net",
			wantHost: "https://acme.databricks.net",
		},
		{
			name:            "existing workspace id is preserved",
			host:            "https://acme.databricks.net/?o=12345",
			workspaceID:     "99999",
			wantHost:        "https://acme.databricks.net",
			wantWorkspaceID: "99999",
		},
		{
			name: "empty host is a no-op",
		},
		{
			name:     "non-numeric o is dropped, host trailing slash trimmed",
			host:     "https://acme.databricks.net/?o=notanumber",
			wantHost: "https://acme.databricks.net",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(envHost, tt.host)
			t.Setenv(envWorkspaceID, tt.workspaceID)
			t.Setenv(envAccountID, tt.accountID)

			NormalizeDatabricksHostEnv()

			assert.Equal(t, tt.wantHost, os.Getenv(envHost))
			assert.Equal(t, tt.wantWorkspaceID, os.Getenv(envWorkspaceID))
			assert.Equal(t, tt.wantAccountID, os.Getenv(envAccountID))
		})
	}
}
