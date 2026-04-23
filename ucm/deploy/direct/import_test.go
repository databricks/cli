package direct_test

import (
	"context"
	"errors"
	"testing"

	"github.com/databricks/cli/ucm"
	"github.com/databricks/cli/ucm/config"
	"github.com/databricks/cli/ucm/config/resources"
	"github.com/databricks/cli/ucm/deploy/direct"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// importFakeClient is a narrow fake that returns pre-seeded SDK responses
// for the Get* calls used by ImportResource. Create/Update/Delete are
// implemented as no-ops since import never issues them.
type importFakeClient struct {
	Catalog           *catalog.CatalogInfo
	Schema            *catalog.SchemaInfo
	StorageCredential *catalog.StorageCredentialInfo
	ExternalLocation  *catalog.ExternalLocationInfo
	Volume            *catalog.VolumeInfo
	Connection        *catalog.ConnectionInfo
	Err               error

	LastGetName string
}

func (c *importFakeClient) GetCatalog(_ context.Context, name string) (*catalog.CatalogInfo, error) {
	c.LastGetName = name
	return c.Catalog, c.Err
}

func (c *importFakeClient) GetSchema(_ context.Context, fullName string) (*catalog.SchemaInfo, error) {
	c.LastGetName = fullName
	return c.Schema, c.Err
}

func (c *importFakeClient) GetStorageCredential(_ context.Context, name string) (*catalog.StorageCredentialInfo, error) {
	c.LastGetName = name
	return c.StorageCredential, c.Err
}

func (c *importFakeClient) GetExternalLocation(_ context.Context, name string) (*catalog.ExternalLocationInfo, error) {
	c.LastGetName = name
	return c.ExternalLocation, c.Err
}

func (c *importFakeClient) GetVolume(_ context.Context, fullName string) (*catalog.VolumeInfo, error) {
	c.LastGetName = fullName
	return c.Volume, c.Err
}

func (c *importFakeClient) GetConnection(_ context.Context, name string) (*catalog.ConnectionInfo, error) {
	c.LastGetName = name
	return c.Connection, c.Err
}

func (*importFakeClient) CreateCatalog(_ context.Context, _ catalog.CreateCatalog) (*catalog.CatalogInfo, error) {
	return nil, nil
}
func (*importFakeClient) UpdateCatalog(_ context.Context, _ catalog.UpdateCatalog) (*catalog.CatalogInfo, error) {
	return nil, nil
}
func (*importFakeClient) DeleteCatalog(_ context.Context, _ string) error { return nil }
func (*importFakeClient) CreateSchema(_ context.Context, _ catalog.CreateSchema) (*catalog.SchemaInfo, error) {
	return nil, nil
}
func (*importFakeClient) UpdateSchema(_ context.Context, _ catalog.UpdateSchema) (*catalog.SchemaInfo, error) {
	return nil, nil
}
func (*importFakeClient) DeleteSchema(_ context.Context, _ string) error { return nil }
func (*importFakeClient) CreateStorageCredential(_ context.Context, _ catalog.CreateStorageCredential) (*catalog.StorageCredentialInfo, error) {
	return nil, nil
}
func (*importFakeClient) UpdateStorageCredential(_ context.Context, _ catalog.UpdateStorageCredential) (*catalog.StorageCredentialInfo, error) {
	return nil, nil
}
func (*importFakeClient) DeleteStorageCredential(_ context.Context, _ string) error { return nil }
func (*importFakeClient) CreateExternalLocation(_ context.Context, _ catalog.CreateExternalLocation) (*catalog.ExternalLocationInfo, error) {
	return nil, nil
}
func (*importFakeClient) UpdateExternalLocation(_ context.Context, _ catalog.UpdateExternalLocation) (*catalog.ExternalLocationInfo, error) {
	return nil, nil
}
func (*importFakeClient) DeleteExternalLocation(_ context.Context, _ string) error { return nil }
func (*importFakeClient) CreateVolume(_ context.Context, _ catalog.CreateVolumeRequestContent) (*catalog.VolumeInfo, error) {
	return nil, nil
}
func (*importFakeClient) UpdateVolume(_ context.Context, _ catalog.UpdateVolumeRequestContent) (*catalog.VolumeInfo, error) {
	return nil, nil
}
func (*importFakeClient) DeleteVolume(_ context.Context, _ string) error { return nil }
func (*importFakeClient) CreateConnection(_ context.Context, _ catalog.CreateConnection) (*catalog.ConnectionInfo, error) {
	return nil, nil
}
func (*importFakeClient) UpdateConnection(_ context.Context, _ catalog.UpdateConnection) (*catalog.ConnectionInfo, error) {
	return nil, nil
}
func (*importFakeClient) DeleteConnection(_ context.Context, _ string) error { return nil }
func (*importFakeClient) UpdatePermissions(_ context.Context, _ catalog.UpdatePermissions) error {
	return nil
}

func (*importFakeClient) ListCatalogs(context.Context) ([]catalog.CatalogInfo, error) {
	return nil, nil
}
func (*importFakeClient) ListSchemas(context.Context, string) ([]catalog.SchemaInfo, error) {
	return nil, nil
}
func (*importFakeClient) ListStorageCredentials(context.Context) ([]catalog.StorageCredentialInfo, error) {
	return nil, nil
}
func (*importFakeClient) ListExternalLocations(context.Context) ([]catalog.ExternalLocationInfo, error) {
	return nil, nil
}
func (*importFakeClient) ListVolumes(context.Context, string, string) ([]catalog.VolumeInfo, error) {
	return nil, nil
}
func (*importFakeClient) ListConnections(context.Context) ([]catalog.ConnectionInfo, error) {
	return nil, nil
}

func TestImportResource_CatalogSeedsStateFromSDK(t *testing.T) {
	u := &ucm.Ucm{Config: config.Root{}}
	u.Config.Resources.Catalogs = map[string]*resources.Catalog{
		"main": {Name: "main", Comment: "declared comment"},
	}
	client := &importFakeClient{Catalog: &catalog.CatalogInfo{
		Name:        "main",
		Comment:     "live comment",
		StorageRoot: "s3://live",
		Properties:  map[string]string{"owner": "team_a"},
	}}
	state := direct.NewState()

	require.NoError(t, direct.ImportResource(t.Context(), u, client, state, "catalog", "main", "main"))
	assert.Equal(t, "main", client.LastGetName)

	got := state.Catalogs["main"]
	require.NotNil(t, got)
	assert.Equal(t, "main", got.Name)
	assert.Equal(t, "live comment", got.Comment)
	assert.Equal(t, "s3://live", got.StorageRoot)
	assert.Equal(t, "team_a", got.Tags["owner"])
}

func TestImportResource_SchemaUsesFullName(t *testing.T) {
	u := &ucm.Ucm{Config: config.Root{}}
	u.Config.Resources.Schemas = map[string]*resources.Schema{
		"raw": {Name: "raw", Catalog: "main"},
	}
	client := &importFakeClient{Schema: &catalog.SchemaInfo{
		Name: "raw", CatalogName: "main", Comment: "bronze",
	}}
	state := direct.NewState()

	require.NoError(t, direct.ImportResource(t.Context(), u, client, state, "schema", "main.raw", "raw"))
	assert.Equal(t, "main.raw", client.LastGetName)

	got := state.Schemas["raw"]
	require.NotNil(t, got)
	assert.Equal(t, "raw", got.Name)
	assert.Equal(t, "main", got.Catalog)
	assert.Equal(t, "bronze", got.Comment)
}

func TestImportResource_StorageCredentialRetainsClientSecret(t *testing.T) {
	u := &ucm.Ucm{Config: config.Root{}}
	u.Config.Resources.StorageCredentials = map[string]*resources.StorageCredential{
		"azure_sp": {
			Name: "azure_sp",
			AzureServicePrincipal: &resources.AzureServicePrincipal{
				DirectoryId:   "tenant",
				ApplicationId: "app",
				ClientSecret:  "local-only-secret",
			},
		},
	}
	client := &importFakeClient{StorageCredential: &catalog.StorageCredentialInfo{
		Name:    "azure_sp",
		Comment: "live",
		AzureServicePrincipal: &catalog.AzureServicePrincipal{
			DirectoryId:   "tenant",
			ApplicationId: "app",
			// ClientSecret deliberately not echoed by UC.
		},
	}}
	state := direct.NewState()

	require.NoError(t, direct.ImportResource(t.Context(), u, client, state, "storage_credential", "azure_sp", "azure_sp"))

	got := state.StorageCredentials["azure_sp"]
	require.NotNil(t, got)
	require.NotNil(t, got.AzureServicePrincipal)
	assert.Equal(t, "local-only-secret", got.AzureServicePrincipal.ClientSecret)
}

func TestImportResource_VolumeUsesFullName(t *testing.T) {
	u := &ucm.Ucm{Config: config.Root{}}
	u.Config.Resources.Volumes = map[string]*resources.Volume{
		"docs": {Name: "docs", CatalogName: "main", SchemaName: "raw", VolumeType: "MANAGED"},
	}
	client := &importFakeClient{Volume: &catalog.VolumeInfo{
		Name: "docs", CatalogName: "main", SchemaName: "raw", VolumeType: catalog.VolumeTypeManaged,
	}}
	state := direct.NewState()

	require.NoError(t, direct.ImportResource(t.Context(), u, client, state, "volume", "main.raw.docs", "docs"))
	assert.Equal(t, "main.raw.docs", client.LastGetName)

	got := state.Volumes["docs"]
	require.NotNil(t, got)
	assert.Equal(t, "docs", got.Name)
	assert.Equal(t, "MANAGED", got.VolumeType)
}

func TestImportResource_ConnectionCopiesOptions(t *testing.T) {
	u := &ucm.Ucm{Config: config.Root{}}
	u.Config.Resources.Connections = map[string]*resources.Connection{
		"mysql_prod": {Name: "mysql_prod", ConnectionType: "MYSQL"},
	}
	client := &importFakeClient{Connection: &catalog.ConnectionInfo{
		Name:           "mysql_prod",
		ConnectionType: catalog.ConnectionTypeMysql,
		Options:        map[string]string{"host": "db.example.com"},
	}}
	state := direct.NewState()

	require.NoError(t, direct.ImportResource(t.Context(), u, client, state, "connection", "mysql_prod", "mysql_prod"))

	got := state.Connections["mysql_prod"]
	require.NotNil(t, got)
	assert.Equal(t, "db.example.com", got.Options["host"])
}

func TestImportResource_UnknownKindErrors(t *testing.T) {
	u := &ucm.Ucm{Config: config.Root{}}
	state := direct.NewState()

	err := direct.ImportResource(t.Context(), u, &importFakeClient{}, state, "table", "foo", "foo")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported import kind")
}

func TestImportResource_PropagatesSDKError(t *testing.T) {
	u := &ucm.Ucm{Config: config.Root{}}
	u.Config.Resources.Catalogs = map[string]*resources.Catalog{"main": {Name: "main"}}
	sentinel := errors.New("sdk boom")
	client := &importFakeClient{Err: sentinel}
	state := direct.NewState()

	err := direct.ImportResource(t.Context(), u, client, state, "catalog", "main", "main")
	require.ErrorIs(t, err, sentinel)
}
