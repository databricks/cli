package profilehash

import (
	"testing"

	"github.com/databricks/cli/libs/databrickscfg/profile"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCompute pins the fingerprint of a fully populated simplified profile.
func TestCompute(t *testing.T) {
	p := profile.Profile{
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

	got, err := Compute(p)
	require.NoError(t, err)

	assert.Equal(t, "3cbfa51f978ce7e9b6ad71d89c0d902e6ce4abfe9549c9bbc7495b830b6fd655", got)
}
