package direct_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/ucm/deploy/direct"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadState_MissingFileReturnsEmpty(t *testing.T) {
	path := filepath.Join(t.TempDir(), "resources.json")
	s, err := direct.LoadState(path)
	require.NoError(t, err)
	require.NotNil(t, s)
	assert.Empty(t, s.Catalogs)
	assert.Empty(t, s.Schemas)
	assert.Empty(t, s.Grants)
	assert.Empty(t, s.StorageCredentials)
	assert.Empty(t, s.ExternalLocations)
	assert.Empty(t, s.Volumes)
	assert.Empty(t, s.Connections)
}

func TestLoadState_RoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "resources.json")
	in := direct.NewState()
	in.Catalogs["main"] = &direct.CatalogState{Name: "main", Comment: "prod"}
	in.Schemas["raw"] = &direct.SchemaState{Name: "raw", Catalog: "main"}
	in.Grants["analysts"] = &direct.GrantState{
		SecurableType: "schema",
		SecurableName: "main.raw",
		Principal:     "analysts",
		Privileges:    []string{"SELECT"},
	}
	in.StorageCredentials["prod"] = &direct.StorageCredentialState{
		Name:       "prod",
		Comment:    "prod credential",
		AwsIamRole: &direct.AwsIamRoleState{RoleArn: "arn:aws:iam::1:role/uc"},
		ReadOnly:   true,
	}
	in.ExternalLocations["prod"] = &direct.ExternalLocationState{
		Name:           "prod",
		Url:            "s3://bucket/prefix",
		CredentialName: "prod",
		Comment:        "prod location",
		ReadOnly:       true,
	}

	require.NoError(t, direct.SaveState(path, in))

	out, err := direct.LoadState(path)
	require.NoError(t, err)
	assert.Equal(t, in.Catalogs, out.Catalogs)
	assert.Equal(t, in.Schemas, out.Schemas)
	assert.Equal(t, in.Grants, out.Grants)
	assert.Equal(t, in.StorageCredentials, out.StorageCredentials)
	assert.Equal(t, in.ExternalLocations, out.ExternalLocations)
}

func TestLoadState_RejectsFutureVersion(t *testing.T) {
	path := filepath.Join(t.TempDir(), "resources.json")
	require.NoError(t, os.WriteFile(path, []byte(`{"version":99}`), 0o644))

	_, err := direct.LoadState(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "version 99")
}
