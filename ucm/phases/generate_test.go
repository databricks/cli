package phases_test

import (
	"context"
	"errors"
	"testing"

	"github.com/databricks/cli/ucm/deploy/direct"
	"github.com/databricks/cli/ucm/phases"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// scanClient is the test stand-in for direct.Client used by the generate
// phase tests. Only the List* methods are exercised; the rest return zero
// values so Plan/Apply test code paths can't accidentally reach this client.
type scanClient struct {
	catalogs           []catalog.CatalogInfo
	schemas            map[string][]catalog.SchemaInfo
	volumes            map[string][]catalog.VolumeInfo
	storageCredentials []catalog.StorageCredentialInfo
	externalLocations  []catalog.ExternalLocationInfo
	connections        []catalog.ConnectionInfo

	listCatalogsErr error
	listSchemasErr  map[string]error
}

func (c *scanClient) ListCatalogs(_ context.Context) ([]catalog.CatalogInfo, error) {
	return c.catalogs, c.listCatalogsErr
}

func (c *scanClient) ListSchemas(_ context.Context, name string) ([]catalog.SchemaInfo, error) {
	if err := c.listSchemasErr[name]; err != nil {
		return nil, err
	}
	return c.schemas[name], nil
}

func (c *scanClient) ListVolumes(_ context.Context, cat, sch string) ([]catalog.VolumeInfo, error) {
	return c.volumes[cat+"."+sch], nil
}

func (c *scanClient) ListStorageCredentials(_ context.Context) ([]catalog.StorageCredentialInfo, error) {
	return c.storageCredentials, nil
}

func (c *scanClient) ListExternalLocations(_ context.Context) ([]catalog.ExternalLocationInfo, error) {
	return c.externalLocations, nil
}

func (c *scanClient) ListConnections(_ context.Context) ([]catalog.ConnectionInfo, error) {
	return c.connections, nil
}

// The remaining direct.Client methods are unreachable from the generate path
// but must exist to satisfy the interface.
func (*scanClient) GetCatalog(context.Context, string) (*catalog.CatalogInfo, error) {
	return nil, nil
}

func (*scanClient) CreateCatalog(context.Context, catalog.CreateCatalog) (*catalog.CatalogInfo, error) {
	return nil, nil
}

func (*scanClient) UpdateCatalog(context.Context, catalog.UpdateCatalog) (*catalog.CatalogInfo, error) {
	return nil, nil
}
func (*scanClient) DeleteCatalog(context.Context, string) error                    { return nil }
func (*scanClient) GetSchema(context.Context, string) (*catalog.SchemaInfo, error) { return nil, nil }
func (*scanClient) CreateSchema(context.Context, catalog.CreateSchema) (*catalog.SchemaInfo, error) {
	return nil, nil
}

func (*scanClient) UpdateSchema(context.Context, catalog.UpdateSchema) (*catalog.SchemaInfo, error) {
	return nil, nil
}
func (*scanClient) DeleteSchema(context.Context, string) error { return nil }
func (*scanClient) GetStorageCredential(context.Context, string) (*catalog.StorageCredentialInfo, error) {
	return nil, nil
}

func (*scanClient) CreateStorageCredential(context.Context, catalog.CreateStorageCredential) (*catalog.StorageCredentialInfo, error) {
	return nil, nil
}

func (*scanClient) UpdateStorageCredential(context.Context, catalog.UpdateStorageCredential) (*catalog.StorageCredentialInfo, error) {
	return nil, nil
}
func (*scanClient) DeleteStorageCredential(context.Context, string) error { return nil }
func (*scanClient) GetExternalLocation(context.Context, string) (*catalog.ExternalLocationInfo, error) {
	return nil, nil
}

func (*scanClient) CreateExternalLocation(context.Context, catalog.CreateExternalLocation) (*catalog.ExternalLocationInfo, error) {
	return nil, nil
}

func (*scanClient) UpdateExternalLocation(context.Context, catalog.UpdateExternalLocation) (*catalog.ExternalLocationInfo, error) {
	return nil, nil
}
func (*scanClient) DeleteExternalLocation(context.Context, string) error { return nil }
func (*scanClient) GetVolume(context.Context, string) (*catalog.VolumeInfo, error) {
	return nil, nil
}

func (*scanClient) CreateVolume(context.Context, catalog.CreateVolumeRequestContent) (*catalog.VolumeInfo, error) {
	return nil, nil
}

func (*scanClient) UpdateVolume(context.Context, catalog.UpdateVolumeRequestContent) (*catalog.VolumeInfo, error) {
	return nil, nil
}
func (*scanClient) DeleteVolume(context.Context, string) error { return nil }
func (*scanClient) GetConnection(context.Context, string) (*catalog.ConnectionInfo, error) {
	return nil, nil
}

func (*scanClient) CreateConnection(context.Context, catalog.CreateConnection) (*catalog.ConnectionInfo, error) {
	return nil, nil
}

func (*scanClient) UpdateConnection(context.Context, catalog.UpdateConnection) (*catalog.ConnectionInfo, error) {
	return nil, nil
}
func (*scanClient) DeleteConnection(context.Context, string) error                     { return nil }
func (*scanClient) UpdatePermissions(context.Context, catalog.UpdatePermissions) error { return nil }

func TestGenerate_SkipsSystemAndMainCatalogs(t *testing.T) {
	c := &scanClient{
		catalogs: []catalog.CatalogInfo{
			{Name: "system", CatalogType: catalog.CatalogTypeSystemCatalog},
			{Name: "hive_metastore"},
			{Name: "main"},
			{Name: "team_alpha", Comment: "alpha", Properties: map[string]string{"owner": "alpha"}},
		},
	}

	r, err := phases.Generate(t.Context(), c, phases.GenerateOptions{
		Name:  "prod",
		Host:  "https://x.example.com",
		Kinds: []string{phases.KindCatalog},
	})
	require.NoError(t, err)
	require.NotNil(t, r.Root)

	require.Len(t, r.Root.Resources.Catalogs, 1)
	got := r.Root.Resources.Catalogs["team_alpha"]
	require.NotNil(t, got)
	assert.Equal(t, "team_alpha", got.Name)
	assert.Equal(t, "alpha", got.Comment)
	assert.Equal(t, map[string]string{"owner": "alpha"}, got.Tags)

	require.Len(t, r.State.Catalogs, 1)
	require.NotNil(t, r.State.Catalogs["team_alpha"])
}

func TestGenerate_SchemasSkipInformationSchema(t *testing.T) {
	c := &scanClient{
		catalogs: []catalog.CatalogInfo{{Name: "team_alpha"}},
		schemas: map[string][]catalog.SchemaInfo{
			"team_alpha": {
				{Name: "bronze", CatalogName: "team_alpha", Comment: "raw"},
				{Name: "information_schema", CatalogName: "team_alpha"},
			},
		},
	}

	r, err := phases.Generate(t.Context(), c, phases.GenerateOptions{
		Name:  "prod",
		Kinds: []string{phases.KindSchema},
	})
	require.NoError(t, err)

	require.Len(t, r.Root.Resources.Schemas, 1)
	s := r.Root.Resources.Schemas["team_alpha_bronze"]
	require.NotNil(t, s)
	assert.Equal(t, "bronze", s.Name)
	assert.Equal(t, "team_alpha", s.Catalog)

	require.NotNil(t, r.State.Schemas["team_alpha_bronze"])
}

func TestGenerate_Volumes(t *testing.T) {
	c := &scanClient{
		catalogs: []catalog.CatalogInfo{{Name: "team_alpha"}},
		schemas: map[string][]catalog.SchemaInfo{
			"team_alpha": {{Name: "bronze", CatalogName: "team_alpha"}},
		},
		volumes: map[string][]catalog.VolumeInfo{
			"team_alpha.bronze": {
				{Name: "landing", CatalogName: "team_alpha", SchemaName: "bronze", VolumeType: catalog.VolumeTypeManaged, StorageLocation: "s3://managed/echo", Comment: "managed"},
				{Name: "raw", CatalogName: "team_alpha", SchemaName: "bronze", VolumeType: catalog.VolumeTypeExternal, StorageLocation: "s3://bucket/raw"},
			},
		},
	}

	r, err := phases.Generate(t.Context(), c, phases.GenerateOptions{
		Kinds: []string{phases.KindVolume},
	})
	require.NoError(t, err)
	require.Len(t, r.Root.Resources.Volumes, 2)

	managed := r.Root.Resources.Volumes["team_alpha_bronze_landing"]
	require.NotNil(t, managed)
	assert.Equal(t, "MANAGED", managed.VolumeType)
	assert.Empty(t, managed.StorageLocation, "managed volume must not retain echoed storage_location")

	external := r.Root.Resources.Volumes["team_alpha_bronze_raw"]
	require.NotNil(t, external)
	assert.Equal(t, "EXTERNAL", external.VolumeType)
	assert.Equal(t, "s3://bucket/raw", external.StorageLocation)
}

func TestGenerate_StorageCredentialAzureSPEmitsWarningAndPlaceholder(t *testing.T) {
	c := &scanClient{
		storageCredentials: []catalog.StorageCredentialInfo{{
			Name: "az-sp",
			AzureServicePrincipal: &catalog.AzureServicePrincipal{
				DirectoryId:   "dir",
				ApplicationId: "app",
			},
		}},
	}

	r, err := phases.Generate(t.Context(), c, phases.GenerateOptions{
		Kinds: []string{phases.KindStorageCredential},
	})
	require.NoError(t, err)
	require.Len(t, r.Warnings, 1)
	assert.Contains(t, r.Warnings[0], "client_secret")

	res := r.Root.Resources.StorageCredentials["az-sp"]
	require.NotNil(t, res)
	require.NotNil(t, res.AzureServicePrincipal)
	assert.Empty(t, res.AzureServicePrincipal.ClientSecret)
}

func TestGenerate_AwsIamRoleRoundTrips(t *testing.T) {
	c := &scanClient{
		storageCredentials: []catalog.StorageCredentialInfo{{
			Name:       "aws-role",
			AwsIamRole: &catalog.AwsIamRoleResponse{RoleArn: "arn:aws:iam::123:role/uc"},
		}},
	}

	r, err := phases.Generate(t.Context(), c, phases.GenerateOptions{
		Kinds: []string{phases.KindStorageCredential},
	})
	require.NoError(t, err)
	assert.Empty(t, r.Warnings)
	res := r.Root.Resources.StorageCredentials["aws-role"]
	require.NotNil(t, res)
	require.NotNil(t, res.AwsIamRole)
	assert.Equal(t, "arn:aws:iam::123:role/uc", res.AwsIamRole.RoleArn)
}

func TestGenerate_ExternalLocations(t *testing.T) {
	c := &scanClient{
		externalLocations: []catalog.ExternalLocationInfo{{
			Name:           "shared",
			Url:            "s3://bucket/data",
			CredentialName: "aws-role",
			Comment:        "shared landing",
			ReadOnly:       true,
		}},
	}

	r, err := phases.Generate(t.Context(), c, phases.GenerateOptions{
		Kinds: []string{phases.KindExternalLocation},
	})
	require.NoError(t, err)
	res := r.Root.Resources.ExternalLocations["shared"]
	require.NotNil(t, res)
	assert.Equal(t, "s3://bucket/data", res.Url)
	assert.Equal(t, "aws-role", res.CredentialName)
	assert.True(t, res.ReadOnly)

	st := r.State.ExternalLocations["shared"]
	require.NotNil(t, st)
	assert.True(t, st.ReadOnly)
}

func TestGenerate_Connections(t *testing.T) {
	c := &scanClient{
		connections: []catalog.ConnectionInfo{{
			Name:           "pg-main",
			ConnectionType: catalog.ConnectionTypePostgresql,
			Options:        map[string]string{"host": "pg.example.com"},
			Comment:        "postgres",
			ReadOnly:       true,
		}},
	}

	r, err := phases.Generate(t.Context(), c, phases.GenerateOptions{
		Kinds: []string{phases.KindConnection},
	})
	require.NoError(t, err)
	res := r.Root.Resources.Connections["pg-main"]
	require.NotNil(t, res)
	assert.Equal(t, "POSTGRESQL", res.ConnectionType)
	assert.Equal(t, "pg.example.com", res.Options["host"])
	assert.True(t, res.ReadOnly)
}

func TestGenerate_DefaultKindsCoversAllSupported(t *testing.T) {
	c := &scanClient{
		catalogs:          []catalog.CatalogInfo{{Name: "team_alpha"}},
		schemas:           map[string][]catalog.SchemaInfo{"team_alpha": {{Name: "bronze", CatalogName: "team_alpha"}}},
		volumes:           map[string][]catalog.VolumeInfo{"team_alpha.bronze": {{Name: "landing", CatalogName: "team_alpha", SchemaName: "bronze", VolumeType: catalog.VolumeTypeManaged}}},
		externalLocations: []catalog.ExternalLocationInfo{{Name: "shared", Url: "s3://bucket/x", CredentialName: "c"}},
		storageCredentials: []catalog.StorageCredentialInfo{{
			Name: "aws-role", AwsIamRole: &catalog.AwsIamRoleResponse{RoleArn: "arn"},
		}},
		connections: []catalog.ConnectionInfo{{Name: "pg", ConnectionType: catalog.ConnectionTypePostgresql, Options: map[string]string{"host": "h"}}},
	}

	r, err := phases.Generate(t.Context(), c, phases.GenerateOptions{Name: "auto"})
	require.NoError(t, err)

	assert.Len(t, r.Root.Resources.Catalogs, 1)
	assert.Len(t, r.Root.Resources.Schemas, 1)
	assert.Len(t, r.Root.Resources.Volumes, 1)
	assert.Len(t, r.Root.Resources.ExternalLocations, 1)
	assert.Len(t, r.Root.Resources.StorageCredentials, 1)
	assert.Len(t, r.Root.Resources.Connections, 1)
}

func TestGenerate_UnknownKindErrors(t *testing.T) {
	_, err := phases.Generate(t.Context(), &scanClient{}, phases.GenerateOptions{
		Kinds: []string{"bogus"},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown kind")
}

func TestGenerate_ListCatalogsErrorPropagates(t *testing.T) {
	c := &scanClient{listCatalogsErr: errors.New("boom")}
	_, err := phases.Generate(t.Context(), c, phases.GenerateOptions{Kinds: []string{phases.KindCatalog}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "list catalogs")
}

func TestGenerate_EmitsUcmNameAndWorkspaceHost(t *testing.T) {
	r, err := phases.Generate(t.Context(), &scanClient{}, phases.GenerateOptions{
		Name:  "scanned-prod",
		Host:  "https://example.cloud.databricks.com",
		Kinds: []string{phases.KindConnection},
	})
	require.NoError(t, err)
	assert.Equal(t, "scanned-prod", r.Root.Ucm.Name)
	assert.Equal(t, "https://example.cloud.databricks.com", r.Root.Workspace.Host)
	assert.Equal(t, direct.StateVersion, r.State.Version)
}
