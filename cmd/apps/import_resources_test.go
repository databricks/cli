package apps

import (
	"testing"

	"github.com/databricks/databricks-sdk-go/service/apps"
	"github.com/stretchr/testify/assert"
)

func TestBuildResourcesMap(t *testing.T) {
	t.Run("with SQL warehouse", func(t *testing.T) {
		app := &apps.App{
			Resources: []apps.AppResource{
				{
					Name: "sql-warehouse",
					SqlWarehouse: &apps.AppResourceSqlWarehouse{
						Id: "abc123",
					},
				},
			},
		}

		resources := buildResourcesMap(app)
		assert.Equal(t, "abc123", resources["sql-warehouse"])
	})

	t.Run("with serving endpoint", func(t *testing.T) {
		app := &apps.App{
			Resources: []apps.AppResource{
				{
					Name: "serving-endpoint",
					ServingEndpoint: &apps.AppResourceServingEndpoint{
						Name: "my-endpoint",
					},
				},
			},
		}

		resources := buildResourcesMap(app)
		assert.Equal(t, "my-endpoint", resources["serving-endpoint"])
	})

	t.Run("with experiment", func(t *testing.T) {
		app := &apps.App{
			Resources: []apps.AppResource{
				{
					Name: "experiment",
					Experiment: &apps.AppResourceExperiment{
						ExperimentId: "exp-456",
					},
				},
			},
		}

		resources := buildResourcesMap(app)
		assert.Equal(t, "exp-456", resources["experiment"])
	})

	t.Run("with database", func(t *testing.T) {
		app := &apps.App{
			Resources: []apps.AppResource{
				{
					Name: "database",
					Database: &apps.AppResourceDatabase{
						DatabaseName: "my-db",
					},
				},
			},
		}

		resources := buildResourcesMap(app)
		assert.Equal(t, "my-db", resources["database"])
	})

	t.Run("with Genie space", func(t *testing.T) {
		app := &apps.App{
			Resources: []apps.AppResource{
				{
					Name: "genie-space",
					GenieSpace: &apps.AppResourceGenieSpace{
						SpaceId: "space-123",
					},
				},
			},
		}

		resources := buildResourcesMap(app)
		assert.Equal(t, "space-123", resources["genie-space"])
	})

	t.Run("with job", func(t *testing.T) {
		app := &apps.App{
			Resources: []apps.AppResource{
				{
					Name: "job",
					Job: &apps.AppResourceJob{
						Id: "job-789",
					},
				},
			},
		}

		resources := buildResourcesMap(app)
		assert.Equal(t, "job-789", resources["job"])
	})

	t.Run("with UC securable", func(t *testing.T) {
		app := &apps.App{
			Resources: []apps.AppResource{
				{
					Name: "uc-securable",
					UcSecurable: &apps.AppResourceUcSecurable{
						SecurableFullName: "catalog.schema.table",
					},
				},
			},
		}

		resources := buildResourcesMap(app)
		assert.Equal(t, "catalog.schema.table", resources["uc-securable"])
	})

	t.Run("with multiple resources", func(t *testing.T) {
		app := &apps.App{
			Resources: []apps.AppResource{
				{
					Name: "sql-warehouse",
					SqlWarehouse: &apps.AppResourceSqlWarehouse{
						Id: "warehouse-1",
					},
				},
				{
					Name: "experiment",
					Experiment: &apps.AppResourceExperiment{
						ExperimentId: "exp-2",
					},
				},
				{
					Name: "serving-endpoint",
					ServingEndpoint: &apps.AppResourceServingEndpoint{
						Name: "endpoint-3",
					},
				},
			},
		}

		resources := buildResourcesMap(app)
		assert.Len(t, resources, 3)
		assert.Equal(t, "warehouse-1", resources["sql-warehouse"])
		assert.Equal(t, "exp-2", resources["experiment"])
		assert.Equal(t, "endpoint-3", resources["serving-endpoint"])
	})

	t.Run("with empty resources", func(t *testing.T) {
		app := &apps.App{
			Resources: []apps.AppResource{},
		}

		resources := buildResourcesMap(app)
		assert.Empty(t, resources)
	})

	t.Run("with nil resources", func(t *testing.T) {
		app := &apps.App{
			Resources: nil,
		}

		resources := buildResourcesMap(app)
		assert.Empty(t, resources)
	})

	t.Run("skips resources with empty name", func(t *testing.T) {
		app := &apps.App{
			Resources: []apps.AppResource{
				{
					Name: "",
					SqlWarehouse: &apps.AppResourceSqlWarehouse{
						Id: "warehouse-1",
					},
				},
				{
					Name: "experiment",
					Experiment: &apps.AppResourceExperiment{
						ExperimentId: "exp-2",
					},
				},
			},
		}

		resources := buildResourcesMap(app)
		assert.Len(t, resources, 1)
		assert.NotContains(t, resources, "")
		assert.Equal(t, "exp-2", resources["experiment"])
	})

	t.Run("skips resources with nil type", func(t *testing.T) {
		app := &apps.App{
			Resources: []apps.AppResource{
				{
					Name: "no-type-resource",
					// All type fields are nil
				},
				{
					Name: "experiment",
					Experiment: &apps.AppResourceExperiment{
						ExperimentId: "exp-2",
					},
				},
			},
		}

		resources := buildResourcesMap(app)
		assert.Len(t, resources, 1)
		assert.NotContains(t, resources, "no-type-resource")
		assert.Equal(t, "exp-2", resources["experiment"])
	})
}
