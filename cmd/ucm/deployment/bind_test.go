package deployment

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/ucm/deploy/direct"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBindResourceDirect_WritesCatalogStateAndLeavesConfigUntouched(t *testing.T) {
	u := setupUcmFixture(t)
	before, err := os.ReadFile(filepath.Join(u.RootPath, "ucm.yml"))
	require.NoError(t, err)

	client := newFakeDirectClient()
	client.catalogs["team_alpha"] = &catalog.CatalogInfo{
		Name:        "team_alpha",
		Comment:     "adopted",
		StorageRoot: "s3://b/team_alpha",
		Properties:  map[string]string{"owner": "alice"},
	}

	require.NoError(t, bindResourceDirect(t.Context(), u, client, kindCatalog, "my_catalog", "team_alpha"))

	// ucm.yml must not be mutated by bind.
	after, err := os.ReadFile(filepath.Join(u.RootPath, "ucm.yml"))
	require.NoError(t, err)
	assert.Equal(t, string(before), string(after))

	// The recorded state must reflect the UC response.
	state, err := direct.LoadState(direct.StatePath(u))
	require.NoError(t, err)
	got, ok := state.Catalogs["my_catalog"]
	require.True(t, ok, "expected catalog state for key my_catalog")
	assert.Equal(t, "team_alpha", got.Name)
	assert.Equal(t, "adopted", got.Comment)
	assert.Equal(t, "s3://b/team_alpha", got.StorageRoot)
	assert.Equal(t, map[string]string{"owner": "alice"}, got.Tags)
}

func TestBindResourceDirect_WritesAllKinds(t *testing.T) {
	u := setupUcmFixture(t)
	client := newFakeDirectClient()
	client.catalogs["team_alpha"] = &catalog.CatalogInfo{Name: "team_alpha"}
	client.schemas["team_alpha.bronze"] = &catalog.SchemaInfo{Name: "bronze", CatalogName: "team_alpha"}
	client.storageCredentials["sc1"] = &catalog.StorageCredentialInfo{
		Name:       "sc1",
		AwsIamRole: &catalog.AwsIamRoleResponse{RoleArn: "arn:aws:iam::1:role/x"},
	}
	client.externalLocations["loc1"] = &catalog.ExternalLocationInfo{
		Name:           "loc1",
		Url:            "s3://b/x",
		CredentialName: "sc1",
	}
	client.volumes["team_alpha.bronze.vol1"] = &catalog.VolumeInfo{
		Name:        "vol1",
		CatalogName: "team_alpha",
		SchemaName:  "bronze",
		VolumeType:  catalog.VolumeTypeManaged,
	}
	client.connections["conn1"] = &catalog.ConnectionInfo{
		Name:           "conn1",
		ConnectionType: catalog.ConnectionTypePostgresql,
		Options:        map[string]string{"host": "db"},
	}

	cases := []struct {
		key, ucName string
		kind        bindableKind
	}{
		{"my_catalog", "team_alpha", kindCatalog},
		{"my_schema", "team_alpha.bronze", kindSchema},
		{"my_sc", "sc1", kindStorageCredential},
		{"my_loc", "loc1", kindExternalLocation},
		{"my_vol", "team_alpha.bronze.vol1", kindVolume},
		{"my_conn", "conn1", kindConnection},
	}
	for _, c := range cases {
		require.NoError(t, bindResourceDirect(t.Context(), u, client, c.kind, c.key, c.ucName), c.key)
	}

	state, err := direct.LoadState(direct.StatePath(u))
	require.NoError(t, err)
	assert.NotNil(t, state.Catalogs["my_catalog"])
	assert.NotNil(t, state.Schemas["my_schema"])
	assert.NotNil(t, state.StorageCredentials["my_sc"])
	assert.NotNil(t, state.ExternalLocations["my_loc"])
	assert.NotNil(t, state.Volumes["my_vol"])
	assert.NotNil(t, state.Connections["my_conn"])
	assert.Equal(t, "arn:aws:iam::1:role/x", state.StorageCredentials["my_sc"].AwsIamRole.RoleArn)
	assert.Equal(t, "MANAGED", state.Volumes["my_vol"].VolumeType)
}

func TestBindResourceDirect_PropagatesFetchError(t *testing.T) {
	u := setupUcmFixture(t)
	client := newFakeDirectClient()

	err := bindResourceDirect(t.Context(), u, client, kindCatalog, "my_catalog", "missing")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "fetch catalog")
	assert.Contains(t, err.Error(), "missing")
}

func TestResolveBindable(t *testing.T) {
	u := setupUcmFixture(t)

	cases := []struct {
		key     string
		want    bindableKind
		wantErr string
	}{
		{"my_catalog", kindCatalog, ""},
		{"my_schema", kindSchema, ""},
		{"my_sc", kindStorageCredential, ""},
		{"my_loc", kindExternalLocation, ""},
		{"my_vol", kindVolume, ""},
		{"my_conn", kindConnection, ""},
		{"grant_a", "", "grants are not bindable"},
		{"does_not_exist", "", "no bindable resource"},
	}

	for _, c := range cases {
		got, err := resolveBindable(u, c.key)
		if c.wantErr != "" {
			require.Error(t, err, c.key)
			assert.Contains(t, err.Error(), c.wantErr, c.key)
			continue
		}
		require.NoError(t, err, c.key)
		assert.Equal(t, c.want, got, c.key)
	}
}
