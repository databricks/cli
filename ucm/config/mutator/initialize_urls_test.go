package mutator_test

import (
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
					"cat1": {Name: "cat1"},
				},
				Schemas: map[string]*resources.Schema{
					"sch1": {Name: "sch1", Catalog: "cat1"},
				},
				Volumes: map[string]*resources.Volume{
					"vol1": {Name: "vol1", CatalogName: "cat1", SchemaName: "sch1"},
				},
				StorageCredentials: map[string]*resources.StorageCredential{
					"sc1": {Name: "sc1"},
				},
				ExternalLocations: map[string]*resources.ExternalLocation{
					"el1": {Name: "el1", Url: "s3://bucket/path"},
				},
				Connections: map[string]*resources.Connection{
					"conn1": {Name: "conn1", ConnectionType: "POSTGRESQL"},
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
					"cat1": {Name: "cat1"},
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
					"cat1": {Name: "cat1"},
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
