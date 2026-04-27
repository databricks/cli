package statemgmt_test

import (
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/ucm"
	"github.com/databricks/cli/ucm/config"
	"github.com/databricks/cli/ucm/config/resources"
	"github.com/databricks/cli/ucm/deploy"
	"github.com/databricks/cli/ucm/statemgmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newUcm returns a Ucm rooted at a fresh temp dir with the target seeded so
// deploy.LocalTfStatePath resolves to <tmp>/.databricks/ucm/<target>/terraform/terraform.tfstate.
func newUcm(t *testing.T, target string, cfg config.Resources) *ucm.Ucm {
	t.Helper()
	root := t.TempDir()
	u := &ucm.Ucm{
		RootPath: root,
		Config: config.Root{
			Ucm:       config.Ucm{Target: target},
			Resources: cfg,
		},
	}
	return u
}

// writeTfstate drops a tfstate file at deploy.LocalTfStatePath(u).
func writeTfstate(t *testing.T, u *ucm.Ucm, body string) {
	t.Helper()
	p := deploy.LocalTfStatePath(u)
	require.NoError(t, os.MkdirAll(filepath.Dir(p), 0o755))
	require.NoError(t, os.WriteFile(p, []byte(body), 0o600))
}

func TestLoad_NilUcmIsNoOp(t *testing.T) {
	diags := statemgmt.Load(t.Context(), nil)
	assert.Empty(t, diags)
}

func TestLoad_MissingFileIsNoOp(t *testing.T) {
	u := newUcm(t, "dev", config.Resources{
		Catalogs: map[string]*resources.Catalog{"c1": {CreateCatalog: catalog.CreateCatalog{Name: "c1"}}},
	})

	diags := statemgmt.Load(t.Context(), u)

	assert.Empty(t, diags)
	assert.Empty(t, u.Config.Resources.Catalogs["c1"].ID)
}

func TestLoad_MalformedJSONWarns(t *testing.T) {
	u := newUcm(t, "dev", config.Resources{
		Catalogs: map[string]*resources.Catalog{"c1": {CreateCatalog: catalog.CreateCatalog{Name: "c1"}}},
	})
	writeTfstate(t, u, `{not valid json`)

	diags := statemgmt.Load(t.Context(), u)

	require.NotEmpty(t, diags)
	assert.False(t, diags.HasError(), "malformed tfstate is a warning, not an error")
	assert.Empty(t, u.Config.Resources.Catalogs["c1"].ID)
}

func TestLoad_WrongVersionWarns(t *testing.T) {
	u := newUcm(t, "dev", config.Resources{
		Catalogs: map[string]*resources.Catalog{"c1": {CreateCatalog: catalog.CreateCatalog{Name: "c1"}}},
	})
	writeTfstate(t, u, `{"version": 3, "resources": []}`)

	diags := statemgmt.Load(t.Context(), u)

	require.NotEmpty(t, diags)
	assert.False(t, diags.HasError())
}

func TestLoad_PopulatesIDsForMappedKinds(t *testing.T) {
	u := newUcm(t, "dev", config.Resources{
		Catalogs: map[string]*resources.Catalog{
			"team_alpha": {CreateCatalog: catalog.CreateCatalog{Name: "team_alpha"}},
		},
		Schemas: map[string]*resources.Schema{
			"bronze": {CreateSchema: catalog.CreateSchema{Name: "bronze", CatalogName: "team_alpha"}},
		},
		Volumes: map[string]*resources.Volume{
			"raw": {Name: "raw", CatalogName: "team_alpha", SchemaName: "bronze"},
		},
		StorageCredentials: map[string]*resources.StorageCredential{
			"creds": {Name: "creds"},
		},
		ExternalLocations: map[string]*resources.ExternalLocation{
			"loc": {Name: "loc"},
		},
		Connections: map[string]*resources.Connection{
			"conn": {Name: "conn", ConnectionType: "POSTGRESQL"},
		},
	})

	writeTfstate(t, u, `{
  "version": 4,
  "resources": [
    {"type": "databricks_catalog",            "name": "team_alpha", "mode": "managed", "instances": [{"attributes": {"id": "team_alpha"}}]},
    {"type": "databricks_schema",             "name": "bronze",     "mode": "managed", "instances": [{"attributes": {"id": "team_alpha.bronze"}}]},
    {"type": "databricks_volume",             "name": "raw",        "mode": "managed", "instances": [{"attributes": {"id": "team_alpha.bronze.raw"}}]},
    {"type": "databricks_storage_credential", "name": "creds",      "mode": "managed", "instances": [{"attributes": {"id": "creds"}}]},
    {"type": "databricks_external_location",  "name": "loc",        "mode": "managed", "instances": [{"attributes": {"id": "loc"}}]},
    {"type": "databricks_connection",         "name": "conn",       "mode": "managed", "instances": [{"attributes": {"id": "conn"}}]}
  ]
}`)

	diags := statemgmt.Load(t.Context(), u)
	require.Empty(t, diags)

	assert.Equal(t, "team_alpha", u.Config.Resources.Catalogs["team_alpha"].ID)
	assert.Equal(t, "team_alpha.bronze", u.Config.Resources.Schemas["bronze"].ID)
	assert.Equal(t, "team_alpha.bronze.raw", u.Config.Resources.Volumes["raw"].ID)
	assert.Equal(t, "creds", u.Config.Resources.StorageCredentials["creds"].ID)
	assert.Equal(t, "loc", u.Config.Resources.ExternalLocations["loc"].ID)
	assert.Equal(t, "conn", u.Config.Resources.Connections["conn"].ID)
}

func TestLoad_UnknownTypeIsSkipped(t *testing.T) {
	u := newUcm(t, "dev", config.Resources{
		Catalogs: map[string]*resources.Catalog{"c1": {CreateCatalog: catalog.CreateCatalog{Name: "c1"}}},
	})
	writeTfstate(t, u, `{
  "version": 4,
  "resources": [
    {"type": "databricks_grant",   "name": "anything", "mode": "managed", "instances": [{"attributes": {"id": "ignored"}}]},
    {"type": "databricks_unknown", "name": "x",        "mode": "managed", "instances": [{"attributes": {"id": "ignored"}}]}
  ]
}`)

	diags := statemgmt.Load(t.Context(), u)

	assert.Empty(t, diags)
	assert.Empty(t, u.Config.Resources.Catalogs["c1"].ID)
}

func TestLoad_MissingConfigKeyIsSkipped(t *testing.T) {
	u := newUcm(t, "dev", config.Resources{
		Catalogs: map[string]*resources.Catalog{
			"c1": {CreateCatalog: catalog.CreateCatalog{Name: "c1"}},
		},
	})
	// state references c2, which has been removed from ucm.yml.
	writeTfstate(t, u, `{
  "version": 4,
  "resources": [
    {"type": "databricks_catalog", "name": "c2", "mode": "managed", "instances": [{"attributes": {"id": "c2"}}]}
  ]
}`)

	diags := statemgmt.Load(t.Context(), u)

	assert.Empty(t, diags)
	assert.Empty(t, u.Config.Resources.Catalogs["c1"].ID)
}

func TestLoad_SkipsNonManagedMode(t *testing.T) {
	u := newUcm(t, "dev", config.Resources{
		Catalogs: map[string]*resources.Catalog{"c1": {CreateCatalog: catalog.CreateCatalog{Name: "c1"}}},
	})
	writeTfstate(t, u, `{
  "version": 4,
  "resources": [
    {"type": "databricks_catalog", "name": "c1", "mode": "data", "instances": [{"attributes": {"id": "c1"}}]}
  ]
}`)

	diags := statemgmt.Load(t.Context(), u)

	assert.Empty(t, diags)
	assert.Empty(t, u.Config.Resources.Catalogs["c1"].ID)
}
