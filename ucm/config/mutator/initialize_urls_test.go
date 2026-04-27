package mutator_test

import (
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"testing"

	"github.com/databricks/cli/ucm"
	"github.com/databricks/cli/ucm/config"
	"github.com/databricks/cli/ucm/config/mutator"
	"github.com/databricks/cli/ucm/config/resources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitializeURLs(t *testing.T) {
	u := &ucm.Ucm{
		Config: config.Root{
			Workspace: config.Workspace{
				Host: "https://mycompany.databricks.com/",
			},
			Resources: config.Resources{
				Catalogs: map[string]*resources.Catalog{
					"cat1": {CreateCatalog: catalog.CreateCatalog{Name: "cat1"}, ID: "cat1"},
				},
				Schemas: map[string]*resources.Schema{
					"sch1": {CreateSchema: catalog.CreateSchema{Name: "sch1", CatalogName: "cat1"}, ID: "cat1.sch1"},
				},
				Volumes: map[string]*resources.Volume{
					"vol1": {CreateVolumeRequestContent: catalog.CreateVolumeRequestContent{Name: "vol1", CatalogName: "cat1", SchemaName: "sch1"}, ID: "cat1.sch1.vol1"},
				},
				StorageCredentials: map[string]*resources.StorageCredential{
					"sc1": {CreateStorageCredential: catalog.CreateStorageCredential{Name: "sc1"}, ID: "sc1"},
				},
				ExternalLocations: map[string]*resources.ExternalLocation{
					"el1": {CreateExternalLocation: catalog.CreateExternalLocation{Name: "el1", Url: "s3://bucket/path"}, ID: "el1"},
				},
				Connections: map[string]*resources.Connection{
					"conn1": {CreateConnection: catalog.CreateConnection{Name: "conn1", ConnectionType: catalog.ConnectionType("POSTGRESQL")}, ID: "conn1"},
				},
			},
		},
	}

	diags := ucm.Apply(t.Context(), u, mutator.InitializeURLs())
	require.Empty(t, diags)

	assert.Equal(t, "https://mycompany.databricks.com/explore/data/cat1", u.Config.Resources.Catalogs["cat1"].URL)
	assert.Equal(t, "https://mycompany.databricks.com/explore/data/cat1/sch1", u.Config.Resources.Schemas["sch1"].URL)
	assert.Equal(t, "https://mycompany.databricks.com/explore/data/cat1/sch1/vol1", u.Config.Resources.Volumes["vol1"].URL)
	assert.Equal(t, "https://mycompany.databricks.com/explore/storage-credentials/sc1", u.Config.Resources.StorageCredentials["sc1"].URL)
	assert.Equal(t, "https://mycompany.databricks.com/explore/external-locations/el1", u.Config.Resources.ExternalLocations["el1"].URL)
	assert.Equal(t, "https://mycompany.databricks.com/explore/connections/conn1", u.Config.Resources.Connections["conn1"].URL)
}

func TestInitializeURLsWarnsWhenHostEmpty(t *testing.T) {
	u := &ucm.Ucm{
		Config: config.Root{
			Resources: config.Resources{
				Catalogs: map[string]*resources.Catalog{
					"cat1": {CreateCatalog: catalog.CreateCatalog{Name: "cat1"}},
				},
			},
		},
	}

	diags := ucm.Apply(t.Context(), u, mutator.InitializeURLs())
	require.NotEmpty(t, diags)
	assert.False(t, diags.HasError(), "host-unset should be a warning, not an error")
	assert.Empty(t, u.Config.Resources.Catalogs["cat1"].URL)
}

func TestInitializeURLsStripsTrailingSlash(t *testing.T) {
	u := &ucm.Ucm{
		Config: config.Root{
			Workspace: config.Workspace{
				Host: "https://mycompany.databricks.com/",
			},
			Resources: config.Resources{
				Catalogs: map[string]*resources.Catalog{
					"cat1": {CreateCatalog: catalog.CreateCatalog{Name: "cat1"}, ID: "cat1"},
				},
			},
		},
	}

	diags := ucm.Apply(t.Context(), u, mutator.InitializeURLs())
	require.Empty(t, diags)

	// Exactly one slash between host and path — no double slash regardless of
	// whether Workspace.Host had a trailing slash.
	assert.Equal(t, "https://mycompany.databricks.com/explore/data/cat1", u.Config.Resources.Catalogs["cat1"].URL)
}

// TestInitializeURLsSkipsNonDeployedResources verifies the DAB-parity gate:
// resources without a tfstate ID are declared-but-not-yet-deployed and must
// leave URL empty so `ucm summary` renders "(not deployed)" instead of a URL
// that 404s.
func TestInitializeURLsSkipsNonDeployedResources(t *testing.T) {
	u := &ucm.Ucm{
		Config: config.Root{
			Workspace: config.Workspace{
				Host: "https://mycompany.databricks.com",
			},
			Resources: config.Resources{
				Catalogs: map[string]*resources.Catalog{
					"cat1": {CreateCatalog: catalog.CreateCatalog{Name: "cat1"}}, // no ID
				},
				Schemas: map[string]*resources.Schema{
					"sch1": {CreateSchema: catalog.CreateSchema{Name: "sch1", CatalogName: "cat1"}}, // no ID
				},
			},
		},
	}

	diags := ucm.Apply(t.Context(), u, mutator.InitializeURLs())
	require.Empty(t, diags)

	assert.Empty(t, u.Config.Resources.Catalogs["cat1"].URL)
	assert.Empty(t, u.Config.Resources.Schemas["sch1"].URL)
}
