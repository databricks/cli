package auth

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizeDatabricksHostEnv(t *testing.T) {
	tests := []struct {
		name              string
		host              string
		workspaceID       string
		accountID         string
		wantHost          string
		wantHostSet       bool
		wantWorkspaceID   string
		wantWorkspaceSet  bool
		wantAccountID     string
		wantAccountIDSet  bool
		preserveHostUnset bool
	}{
		{
			name:             "spog url promotes workspace id",
			host:             "https://acme.databricks.net/?o=12345",
			wantHost:         "https://acme.databricks.net",
			wantHostSet:      true,
			wantWorkspaceID:  "12345",
			wantWorkspaceSet: true,
		},
		{
			name:             "spog url with account id",
			host:             "https://acme.databricks.net/?a=abc&o=12345",
			wantHost:         "https://acme.databricks.net",
			wantHostSet:      true,
			wantWorkspaceID:  "12345",
			wantWorkspaceSet: true,
			wantAccountID:    "abc",
			wantAccountIDSet: true,
		},
		{
			name:        "host without query is left alone",
			host:        "https://acme.databricks.net",
			wantHost:    "https://acme.databricks.net",
			wantHostSet: true,
		},
		{
			name:             "existing workspace id is preserved",
			host:             "https://acme.databricks.net/?o=12345",
			workspaceID:      "99999",
			wantHost:         "https://acme.databricks.net",
			wantHostSet:      true,
			wantWorkspaceID:  "99999",
			wantWorkspaceSet: true,
		},
		{
			name:              "no host env unsets nothing",
			preserveHostUnset: true,
		},
		{
			name:             "non-numeric o is dropped, host trailing slash trimmed",
			host:             "https://acme.databricks.net/?o=notanumber",
			wantHost:         "https://acme.databricks.net",
			wantHostSet:      true,
			wantWorkspaceID:  "",
			wantWorkspaceSet: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.preserveHostUnset {
				os.Unsetenv(envHost)
			} else {
				t.Setenv(envHost, tt.host)
			}
			if tt.workspaceID != "" {
				t.Setenv(envWorkspaceID, tt.workspaceID)
			} else {
				os.Unsetenv(envWorkspaceID)
			}
			if tt.accountID != "" {
				t.Setenv(envAccountID, tt.accountID)
			} else {
				os.Unsetenv(envAccountID)
			}

			NormalizeDatabricksHostEnv()

			gotHost, hostSet := os.LookupEnv(envHost)
			assert.Equal(t, tt.wantHostSet, hostSet)
			if tt.wantHostSet {
				assert.Equal(t, tt.wantHost, gotHost)
			}

			gotWS, wsSet := os.LookupEnv(envWorkspaceID)
			assert.Equal(t, tt.wantWorkspaceSet, wsSet)
			if tt.wantWorkspaceSet {
				assert.Equal(t, tt.wantWorkspaceID, gotWS)
			}

			gotAcc, accSet := os.LookupEnv(envAccountID)
			assert.Equal(t, tt.wantAccountIDSet, accSet)
			if tt.wantAccountIDSet {
				assert.Equal(t, tt.wantAccountID, gotAcc)
			}
		})
	}
}
