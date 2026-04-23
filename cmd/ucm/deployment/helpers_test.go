package deployment

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/ucm"
	"github.com/databricks/cli/ucm/deploy/direct"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/stretchr/testify/require"
)

// fakeDirectClient is an in-memory stand-in for direct.Client used by
// bind/unbind tests. Only the Get* methods used by the bind path are
// populated — the mutating methods panic if called unexpectedly.
type fakeDirectClient struct {
	catalogs           map[string]*catalog.CatalogInfo
	schemas            map[string]*catalog.SchemaInfo
	storageCredentials map[string]*catalog.StorageCredentialInfo
	externalLocations  map[string]*catalog.ExternalLocationInfo
	volumes            map[string]*catalog.VolumeInfo
	connections        map[string]*catalog.ConnectionInfo
}

func newFakeDirectClient() *fakeDirectClient {
	return &fakeDirectClient{
		catalogs:           map[string]*catalog.CatalogInfo{},
		schemas:            map[string]*catalog.SchemaInfo{},
		storageCredentials: map[string]*catalog.StorageCredentialInfo{},
		externalLocations:  map[string]*catalog.ExternalLocationInfo{},
		volumes:            map[string]*catalog.VolumeInfo{},
		connections:        map[string]*catalog.ConnectionInfo{},
	}
}

func (f *fakeDirectClient) GetCatalog(_ context.Context, name string) (*catalog.CatalogInfo, error) {
	v, ok := f.catalogs[name]
	if !ok {
		return nil, fmt.Errorf("catalog %q not found", name)
	}
	return v, nil
}

func (f *fakeDirectClient) GetSchema(_ context.Context, fullName string) (*catalog.SchemaInfo, error) {
	v, ok := f.schemas[fullName]
	if !ok {
		return nil, fmt.Errorf("schema %q not found", fullName)
	}
	return v, nil
}

func (f *fakeDirectClient) GetStorageCredential(_ context.Context, name string) (*catalog.StorageCredentialInfo, error) {
	v, ok := f.storageCredentials[name]
	if !ok {
		return nil, fmt.Errorf("storage_credential %q not found", name)
	}
	return v, nil
}

func (f *fakeDirectClient) GetExternalLocation(_ context.Context, name string) (*catalog.ExternalLocationInfo, error) {
	v, ok := f.externalLocations[name]
	if !ok {
		return nil, fmt.Errorf("external_location %q not found", name)
	}
	return v, nil
}

func (f *fakeDirectClient) GetVolume(_ context.Context, name string) (*catalog.VolumeInfo, error) {
	v, ok := f.volumes[name]
	if !ok {
		return nil, fmt.Errorf("volume %q not found", name)
	}
	return v, nil
}

func (f *fakeDirectClient) GetConnection(_ context.Context, name string) (*catalog.ConnectionInfo, error) {
	v, ok := f.connections[name]
	if !ok {
		return nil, fmt.Errorf("connection %q not found", name)
	}
	return v, nil
}

// Mutating methods panic: bind/unbind must only read from UC, never write.
func (f *fakeDirectClient) CreateCatalog(context.Context, catalog.CreateCatalog) (*catalog.CatalogInfo, error) {
	panic("bind/unbind should not call CreateCatalog")
}

func (f *fakeDirectClient) UpdateCatalog(context.Context, catalog.UpdateCatalog) (*catalog.CatalogInfo, error) {
	panic("bind/unbind should not call UpdateCatalog")
}
func (f *fakeDirectClient) DeleteCatalog(context.Context, string) error {
	panic("bind/unbind should not call DeleteCatalog")
}

func (f *fakeDirectClient) CreateSchema(context.Context, catalog.CreateSchema) (*catalog.SchemaInfo, error) {
	panic("bind/unbind should not call CreateSchema")
}

func (f *fakeDirectClient) UpdateSchema(context.Context, catalog.UpdateSchema) (*catalog.SchemaInfo, error) {
	panic("bind/unbind should not call UpdateSchema")
}
func (f *fakeDirectClient) DeleteSchema(context.Context, string) error {
	panic("bind/unbind should not call DeleteSchema")
}

func (f *fakeDirectClient) CreateStorageCredential(context.Context, catalog.CreateStorageCredential) (*catalog.StorageCredentialInfo, error) {
	panic("bind/unbind should not call CreateStorageCredential")
}

func (f *fakeDirectClient) UpdateStorageCredential(context.Context, catalog.UpdateStorageCredential) (*catalog.StorageCredentialInfo, error) {
	panic("bind/unbind should not call UpdateStorageCredential")
}
func (f *fakeDirectClient) DeleteStorageCredential(context.Context, string) error {
	panic("bind/unbind should not call DeleteStorageCredential")
}

func (f *fakeDirectClient) CreateExternalLocation(context.Context, catalog.CreateExternalLocation) (*catalog.ExternalLocationInfo, error) {
	panic("bind/unbind should not call CreateExternalLocation")
}

func (f *fakeDirectClient) UpdateExternalLocation(context.Context, catalog.UpdateExternalLocation) (*catalog.ExternalLocationInfo, error) {
	panic("bind/unbind should not call UpdateExternalLocation")
}
func (f *fakeDirectClient) DeleteExternalLocation(context.Context, string) error {
	panic("bind/unbind should not call DeleteExternalLocation")
}

func (f *fakeDirectClient) CreateVolume(context.Context, catalog.CreateVolumeRequestContent) (*catalog.VolumeInfo, error) {
	panic("bind/unbind should not call CreateVolume")
}

func (f *fakeDirectClient) UpdateVolume(context.Context, catalog.UpdateVolumeRequestContent) (*catalog.VolumeInfo, error) {
	panic("bind/unbind should not call UpdateVolume")
}
func (f *fakeDirectClient) DeleteVolume(context.Context, string) error {
	panic("bind/unbind should not call DeleteVolume")
}

func (f *fakeDirectClient) CreateConnection(context.Context, catalog.CreateConnection) (*catalog.ConnectionInfo, error) {
	panic("bind/unbind should not call CreateConnection")
}

func (f *fakeDirectClient) UpdateConnection(context.Context, catalog.UpdateConnection) (*catalog.ConnectionInfo, error) {
	panic("bind/unbind should not call UpdateConnection")
}
func (f *fakeDirectClient) DeleteConnection(context.Context, string) error {
	panic("bind/unbind should not call DeleteConnection")
}

func (f *fakeDirectClient) UpdatePermissions(context.Context, catalog.UpdatePermissions) error {
	panic("bind/unbind should not call UpdatePermissions")
}

func (*fakeDirectClient) ListCatalogs(context.Context) ([]catalog.CatalogInfo, error) {
	return nil, nil
}
func (*fakeDirectClient) ListSchemas(context.Context, string) ([]catalog.SchemaInfo, error) {
	return nil, nil
}
func (*fakeDirectClient) ListStorageCredentials(context.Context) ([]catalog.StorageCredentialInfo, error) {
	return nil, nil
}
func (*fakeDirectClient) ListExternalLocations(context.Context) ([]catalog.ExternalLocationInfo, error) {
	return nil, nil
}
func (*fakeDirectClient) ListVolumes(context.Context, string, string) ([]catalog.VolumeInfo, error) {
	return nil, nil
}
func (*fakeDirectClient) ListConnections(context.Context) ([]catalog.ConnectionInfo, error) {
	return nil, nil
}

var _ direct.Client = (*fakeDirectClient)(nil)

// setupUcmFixture writes a ucm.yml with every supported resource kind into a
// fresh temp dir, loads it via ucm.Load, and selects the default target. The
// returned *ucm.Ucm is ready for a direct-engine bind/unbind call against.
func setupUcmFixture(t *testing.T) *ucm.Ucm {
	t.Helper()
	dir := t.TempDir()
	yml := `ucm:
  name: test-bind
  engine: direct

workspace:
  host: https://example.cloud.databricks.com

resources:
  catalogs:
    my_catalog:
      name: team_alpha
  schemas:
    my_schema:
      catalog: team_alpha
      name: bronze
  storage_credentials:
    my_sc:
      name: sc1
      aws_iam_role:
        role_arn: arn:aws:iam::1:role/x
  external_locations:
    my_loc:
      name: loc1
      url: s3://b/x
      credential_name: sc1
  volumes:
    my_vol:
      name: vol1
      catalog_name: team_alpha
      schema_name: bronze
      volume_type: MANAGED
  connections:
    my_conn:
      name: conn1
      connection_type: POSTGRESQL
      options: { host: db }
  grants:
    grant_a:
      securable: { type: catalog, name: team_alpha }
      principal: g
      privileges: [USE_CATALOG]
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "ucm.yml"), []byte(yml), 0o644))

	u, err := ucm.Load(t.Context(), dir)
	require.NoError(t, err)
	// Direct-engine state paths depend on Config.Ucm.Target being set.
	u.Config.Ucm.Target = "default"
	return u
}
