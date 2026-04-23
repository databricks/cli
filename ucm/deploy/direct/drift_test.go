package direct_test

import (
	"context"
	"errors"
	"testing"

	"github.com/databricks/cli/ucm/deploy/direct"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeReadClient is a read-side stand-in for direct.Client. Write methods
// panic so tests can't accidentally exercise them through drift code paths.
// Each Get* method returns a pre-seeded live value or a pre-seeded error
// keyed by the resource identifier the drift comparator passes in.
type fakeReadClient struct {
	catalogs            map[string]*catalog.CatalogInfo
	catalogErrs         map[string]error
	schemas             map[string]*catalog.SchemaInfo
	schemaErrs          map[string]error
	storageCreds        map[string]*catalog.StorageCredentialInfo
	storageCredErrs     map[string]error
	externalLocations   map[string]*catalog.ExternalLocationInfo
	externalLocationErr map[string]error
	volumes             map[string]*catalog.VolumeInfo
	volumeErrs          map[string]error
	connections         map[string]*catalog.ConnectionInfo
	connectionErrs      map[string]error
}

func (f *fakeReadClient) GetCatalog(_ context.Context, name string) (*catalog.CatalogInfo, error) {
	if err := f.catalogErrs[name]; err != nil {
		return nil, err
	}
	return f.catalogs[name], nil
}

func (f *fakeReadClient) GetSchema(_ context.Context, fullName string) (*catalog.SchemaInfo, error) {
	if err := f.schemaErrs[fullName]; err != nil {
		return nil, err
	}
	return f.schemas[fullName], nil
}

func (f *fakeReadClient) GetStorageCredential(_ context.Context, name string) (*catalog.StorageCredentialInfo, error) {
	if err := f.storageCredErrs[name]; err != nil {
		return nil, err
	}
	return f.storageCreds[name], nil
}

func (f *fakeReadClient) GetExternalLocation(_ context.Context, name string) (*catalog.ExternalLocationInfo, error) {
	if err := f.externalLocationErr[name]; err != nil {
		return nil, err
	}
	return f.externalLocations[name], nil
}

func (f *fakeReadClient) GetVolume(_ context.Context, name string) (*catalog.VolumeInfo, error) {
	if err := f.volumeErrs[name]; err != nil {
		return nil, err
	}
	return f.volumes[name], nil
}

func (f *fakeReadClient) GetConnection(_ context.Context, name string) (*catalog.ConnectionInfo, error) {
	if err := f.connectionErrs[name]; err != nil {
		return nil, err
	}
	return f.connections[name], nil
}

// Write-side methods panic — they are unreachable from drift code but required
// to satisfy the direct.Client interface.
func (*fakeReadClient) CreateCatalog(context.Context, catalog.CreateCatalog) (*catalog.CatalogInfo, error) {
	panic("unexpected write")
}

func (*fakeReadClient) UpdateCatalog(context.Context, catalog.UpdateCatalog) (*catalog.CatalogInfo, error) {
	panic("unexpected write")
}
func (*fakeReadClient) DeleteCatalog(context.Context, string) error { panic("unexpected write") }
func (*fakeReadClient) CreateSchema(context.Context, catalog.CreateSchema) (*catalog.SchemaInfo, error) {
	panic("unexpected write")
}

func (*fakeReadClient) UpdateSchema(context.Context, catalog.UpdateSchema) (*catalog.SchemaInfo, error) {
	panic("unexpected write")
}
func (*fakeReadClient) DeleteSchema(context.Context, string) error { panic("unexpected write") }
func (*fakeReadClient) CreateStorageCredential(context.Context, catalog.CreateStorageCredential) (*catalog.StorageCredentialInfo, error) {
	panic("unexpected write")
}

func (*fakeReadClient) UpdateStorageCredential(context.Context, catalog.UpdateStorageCredential) (*catalog.StorageCredentialInfo, error) {
	panic("unexpected write")
}

func (*fakeReadClient) DeleteStorageCredential(context.Context, string) error {
	panic("unexpected write")
}

func (*fakeReadClient) CreateExternalLocation(context.Context, catalog.CreateExternalLocation) (*catalog.ExternalLocationInfo, error) {
	panic("unexpected write")
}

func (*fakeReadClient) UpdateExternalLocation(context.Context, catalog.UpdateExternalLocation) (*catalog.ExternalLocationInfo, error) {
	panic("unexpected write")
}

func (*fakeReadClient) DeleteExternalLocation(context.Context, string) error {
	panic("unexpected write")
}

func (*fakeReadClient) CreateVolume(context.Context, catalog.CreateVolumeRequestContent) (*catalog.VolumeInfo, error) {
	panic("unexpected write")
}

func (*fakeReadClient) UpdateVolume(context.Context, catalog.UpdateVolumeRequestContent) (*catalog.VolumeInfo, error) {
	panic("unexpected write")
}
func (*fakeReadClient) DeleteVolume(context.Context, string) error { panic("unexpected write") }
func (*fakeReadClient) CreateConnection(context.Context, catalog.CreateConnection) (*catalog.ConnectionInfo, error) {
	panic("unexpected write")
}

func (*fakeReadClient) UpdateConnection(context.Context, catalog.UpdateConnection) (*catalog.ConnectionInfo, error) {
	panic("unexpected write")
}

func (*fakeReadClient) DeleteConnection(context.Context, string) error {
	panic("unexpected write")
}

func (*fakeReadClient) UpdatePermissions(context.Context, catalog.UpdatePermissions) error {
	panic("unexpected write")
}

func (*fakeReadClient) ListCatalogs(context.Context) ([]catalog.CatalogInfo, error) {
	return nil, nil
}
func (*fakeReadClient) ListSchemas(context.Context, string) ([]catalog.SchemaInfo, error) {
	return nil, nil
}
func (*fakeReadClient) ListStorageCredentials(context.Context) ([]catalog.StorageCredentialInfo, error) {
	return nil, nil
}
func (*fakeReadClient) ListExternalLocations(context.Context) ([]catalog.ExternalLocationInfo, error) {
	return nil, nil
}
func (*fakeReadClient) ListVolumes(context.Context, string, string) ([]catalog.VolumeInfo, error) {
	return nil, nil
}
func (*fakeReadClient) ListConnections(context.Context) ([]catalog.ConnectionInfo, error) {
	return nil, nil
}

func TestComputeDrift_NoDrift(t *testing.T) {
	state := direct.NewState()
	state.Catalogs["sales"] = &direct.CatalogState{
		Name:    "sales",
		Comment: "sales data",
		Tags:    map[string]string{"owner": "sales"},
	}
	client := &fakeReadClient{
		catalogs: map[string]*catalog.CatalogInfo{
			"sales": {Name: "sales", Comment: "sales data", Properties: map[string]string{"owner": "sales"}},
		},
	}

	r, err := direct.ComputeDrift(t.Context(), client, state)
	require.NoError(t, err)
	assert.False(t, r.HasDrift())
	assert.Empty(t, r.Drift)
}

func TestComputeDrift_CommentDriftReportsStateAndLive(t *testing.T) {
	state := direct.NewState()
	state.Catalogs["sales"] = &direct.CatalogState{Name: "sales", Comment: "sales data"}
	client := &fakeReadClient{
		catalogs: map[string]*catalog.CatalogInfo{
			"sales": {Name: "sales", Comment: "sales domain data"},
		},
	}

	r, err := direct.ComputeDrift(t.Context(), client, state)
	require.NoError(t, err)
	require.True(t, r.HasDrift())
	require.Len(t, r.Drift, 1)
	assert.Equal(t, "resources.catalogs.sales", r.Drift[0].Key)
	require.Len(t, r.Drift[0].Fields, 1)
	assert.Equal(t, "comment", r.Drift[0].Fields[0].Field)
	assert.Equal(t, "sales data", r.Drift[0].Fields[0].State)
	assert.Equal(t, "sales domain data", r.Drift[0].Fields[0].Live)
}

func TestComputeDrift_MissingLiveResourceFlagsExists(t *testing.T) {
	state := direct.NewState()
	state.Catalogs["sales"] = &direct.CatalogState{Name: "sales"}
	client := &fakeReadClient{
		catalogErrs: map[string]error{"sales": errors.New("Catalog 'sales' does not exist.")},
	}

	r, err := direct.ComputeDrift(t.Context(), client, state)
	require.NoError(t, err)
	require.Len(t, r.Drift, 1)
	require.Len(t, r.Drift[0].Fields, 1)
	assert.Equal(t, "_exists", r.Drift[0].Fields[0].Field)
	assert.Equal(t, true, r.Drift[0].Fields[0].State)
	assert.Equal(t, false, r.Drift[0].Fields[0].Live)
}

func TestComputeDrift_PropagatesNonNotFoundError(t *testing.T) {
	state := direct.NewState()
	state.Catalogs["sales"] = &direct.CatalogState{Name: "sales"}
	client := &fakeReadClient{
		catalogErrs: map[string]error{"sales": errors.New("500 internal server error")},
	}

	_, err := direct.ComputeDrift(t.Context(), client, state)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "read catalog sales")
}

func TestComputeDrift_ExternalLocationReadOnlyDrift(t *testing.T) {
	state := direct.NewState()
	state.ExternalLocations["shared"] = &direct.ExternalLocationState{
		Name:           "shared",
		Url:            "s3://bucket/prefix",
		CredentialName: "cred",
		ReadOnly:       false,
	}
	client := &fakeReadClient{
		externalLocations: map[string]*catalog.ExternalLocationInfo{
			"shared": {
				Name:           "shared",
				Url:            "s3://bucket/prefix",
				CredentialName: "cred",
				ReadOnly:       true,
			},
		},
	}

	r, err := direct.ComputeDrift(t.Context(), client, state)
	require.NoError(t, err)
	require.Len(t, r.Drift, 1)
	require.Len(t, r.Drift[0].Fields, 1)
	assert.Equal(t, "read_only", r.Drift[0].Fields[0].Field)
	assert.Equal(t, false, r.Drift[0].Fields[0].State)
	assert.Equal(t, true, r.Drift[0].Fields[0].Live)
}

func TestComputeDrift_VolumeCommentDrift(t *testing.T) {
	state := direct.NewState()
	state.Volumes["landing"] = &direct.VolumeState{
		Name:        "landing",
		CatalogName: "c",
		SchemaName:  "s",
		VolumeType:  "MANAGED",
		Comment:     "landing zone",
	}
	client := &fakeReadClient{
		volumes: map[string]*catalog.VolumeInfo{
			"c.s.landing": {
				Name:       "landing",
				VolumeType: catalog.VolumeType("MANAGED"),
				Comment:    "landing zone changed",
			},
		},
	}

	r, err := direct.ComputeDrift(t.Context(), client, state)
	require.NoError(t, err)
	require.Len(t, r.Drift, 1)
	require.Len(t, r.Drift[0].Fields, 1)
	assert.Equal(t, "comment", r.Drift[0].Fields[0].Field)
}

func TestComputeDrift_StorageCredentialSkipsClientSecret(t *testing.T) {
	state := direct.NewState()
	state.StorageCredentials["az"] = &direct.StorageCredentialState{
		Name: "az",
		AzureServicePrincipal: &direct.AzureServicePrincipalState{
			DirectoryId:   "dir",
			ApplicationId: "app",
			ClientSecret:  "super-secret",
		},
	}
	client := &fakeReadClient{
		storageCreds: map[string]*catalog.StorageCredentialInfo{
			"az": {
				Name: "az",
				AzureServicePrincipal: &catalog.AzureServicePrincipal{
					DirectoryId:   "dir",
					ApplicationId: "app",
				},
			},
		},
	}

	r, err := direct.ComputeDrift(t.Context(), client, state)
	require.NoError(t, err)
	assert.False(t, r.HasDrift(), "client_secret mismatch must not be reported as drift")
}

func TestComputeDrift_ConnectionOptionsMapDrift(t *testing.T) {
	state := direct.NewState()
	state.Connections["snowflake"] = &direct.ConnectionState{
		Name:           "snowflake",
		ConnectionType: "SNOWFLAKE",
		Options:        map[string]string{"host": "old.example.com"},
	}
	client := &fakeReadClient{
		connections: map[string]*catalog.ConnectionInfo{
			"snowflake": {
				Name:           "snowflake",
				ConnectionType: catalog.ConnectionType("SNOWFLAKE"),
				Options:        map[string]string{"host": "new.example.com"},
			},
		},
	}

	r, err := direct.ComputeDrift(t.Context(), client, state)
	require.NoError(t, err)
	require.Len(t, r.Drift, 1)
	require.Len(t, r.Drift[0].Fields, 1)
	assert.Equal(t, "options", r.Drift[0].Fields[0].Field)
}

func TestComputeDrift_OrdersByResourceKindAndKey(t *testing.T) {
	state := direct.NewState()
	state.Catalogs["bb"] = &direct.CatalogState{Name: "bb", Comment: "want"}
	state.Catalogs["aa"] = &direct.CatalogState{Name: "aa", Comment: "want"}
	state.Schemas["aa"] = &direct.SchemaState{Name: "aa", Catalog: "bb", Comment: "want"}

	client := &fakeReadClient{
		catalogs: map[string]*catalog.CatalogInfo{
			"bb": {Name: "bb", Comment: "drift"},
			"aa": {Name: "aa", Comment: "drift"},
		},
		schemas: map[string]*catalog.SchemaInfo{
			"bb.aa": {Name: "aa", CatalogName: "bb", Comment: "drift"},
		},
	}

	r, err := direct.ComputeDrift(t.Context(), client, state)
	require.NoError(t, err)
	require.Len(t, r.Drift, 3)
	assert.Equal(t, "resources.catalogs.aa", r.Drift[0].Key)
	assert.Equal(t, "resources.catalogs.bb", r.Drift[1].Key)
	assert.Equal(t, "resources.schemas.aa", r.Drift[2].Key)
}
