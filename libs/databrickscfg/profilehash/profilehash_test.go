package profilehash

import (
	"testing"

	"github.com/databricks/cli/libs/databrickscfg/profile"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestComputeIncludesAllProfileFields verifies that every field in the
// simplified profile contributes to its fingerprint.
func TestComputeIncludesAllProfileFields(t *testing.T) {
	base := profile.Profile{
		Name:                 "TEST",
		Host:                 "https://workspace.example.test",
		AccountID:            "account-id",
		WorkspaceID:          "workspace-id",
		ClusterID:            "cluster-id",
		ServerlessComputeID:  "serverless-compute-id",
		HasClientCredentials: true,
		Scopes:               "all-apis",
		AuthType:             "databricks-cli",
	}

	want, err := Compute(base)
	require.NoError(t, err)

	tests := []struct {
		name   string
		change func(*profile.Profile)
	}{
		{
			name: "name",
			change: func(p *profile.Profile) {
				p.Name = "OTHER"
			},
		},
		{
			name: "host",
			change: func(p *profile.Profile) {
				p.Host = "https://other.example.test"
			},
		},
		{
			name: "account ID",
			change: func(p *profile.Profile) {
				p.AccountID = "other-account"
			},
		},
		{
			name: "workspace ID",
			change: func(p *profile.Profile) {
				p.WorkspaceID = "other-workspace"
			},
		},
		{
			name: "cluster ID",
			change: func(p *profile.Profile) {
				p.ClusterID = "other-cluster"
			},
		},
		{
			name: "serverless compute ID",
			change: func(p *profile.Profile) {
				p.ServerlessComputeID = "other-serverless"
			},
		},
		{
			name: "client credentials",
			change: func(p *profile.Profile) {
				p.HasClientCredentials = false
			},
		},
		{
			name: "scopes",
			change: func(p *profile.Profile) {
				p.Scopes = "jobs"
			},
		},
		{
			name: "auth type",
			change: func(p *profile.Profile) {
				p.AuthType = "pat"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			changed := base
			tt.change(&changed)

			got, err := Compute(changed)
			require.NoError(t, err)

			assert.NotEqual(t, want, got)
		})
	}
}

// TestComputeCanonicalizesScopes verifies that semantically equivalent scope
// lists produce the same profile fingerprint.
func TestComputeCanonicalizesScopes(t *testing.T) {
	want, err := Compute(profile.Profile{Scopes: "all-apis,sql"})
	require.NoError(t, err)

	tests := []struct {
		name   string
		scopes string
	}{
		{
			name:   "different order",
			scopes: "sql,all-apis",
		},
		{
			name:   "surrounding whitespace",
			scopes: " sql , all-apis ",
		},
		{
			name:   "duplicates and empty values",
			scopes: "sql,,all-apis,sql,",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Compute(profile.Profile{Scopes: tt.scopes})
			require.NoError(t, err)

			assert.Equal(t, want, got)
		})
	}
}
