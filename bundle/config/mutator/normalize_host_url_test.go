package mutator_test

import (
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeHostURL(t *testing.T) {
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
			name:     "empty host",
			host:     "",
			wantHost: "",
		},
		{
			name:     "host without query params",
			host:     "https://api.databricks.com",
			wantHost: "https://api.databricks.com",
		},
		{
			name:            "host with ?o= extracts workspace_id",
			host:            "https://dogfood.staging.databricks.com/?o=6051921418418893",
			wantHost:        "https://dogfood.staging.databricks.com/",
			wantWorkspaceID: "6051921418418893",
		},
		{
			name:            "host with ?o= and ?a= extracts both",
			host:            "https://dogfood.staging.databricks.com/?o=605&a=abc123",
			wantHost:        "https://dogfood.staging.databricks.com/",
			wantWorkspaceID: "605",
			wantAccountID:   "abc123",
		},
		{
			name:          "host with ?account_id= extracts account_id",
			host:          "https://accounts.cloud.databricks.com/?account_id=968367da-1234",
			wantHost:      "https://accounts.cloud.databricks.com/",
			wantAccountID: "968367da-1234",
		},
		{
			name:            "host with ?workspace_id= extracts workspace_id",
			host:            "https://dogfood.staging.databricks.com/?workspace_id=12345",
			wantHost:        "https://dogfood.staging.databricks.com/",
			wantWorkspaceID: "12345",
		},
		{
			name:            "explicit workspace_id takes precedence over ?o=",
			host:            "https://dogfood.staging.databricks.com/?o=999",
			workspaceID:     "explicit-id",
			wantHost:        "https://dogfood.staging.databricks.com/",
			wantWorkspaceID: "explicit-id",
		},
		{
			name:          "explicit account_id takes precedence over ?a=",
			host:          "https://dogfood.staging.databricks.com/?a=from-url",
			accountID:     "explicit-account",
			wantHost:      "https://dogfood.staging.databricks.com/",
			wantAccountID: "explicit-account",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &bundle.Bundle{
				Config: config.Root{
					Workspace: config.Workspace{
						Host:        tt.host,
						WorkspaceID: tt.workspaceID,
						AccountID:   tt.accountID,
					},
				},
			}

			diags := bundle.Apply(t.Context(), b, mutator.NormalizeHostURL())
			require.NoError(t, diags.Error())

			assert.Equal(t, tt.wantHost, b.Config.Workspace.Host)
			assert.Equal(t, tt.wantWorkspaceID, b.Config.Workspace.WorkspaceID)
			assert.Equal(t, tt.wantAccountID, b.Config.Workspace.AccountID)
		})
	}
}
