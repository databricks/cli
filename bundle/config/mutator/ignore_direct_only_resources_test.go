package mutator_test

import (
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIgnoreDirectOnlyResourcesNoDirectOnlyReturnsNil(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"my_job": {},
				},
			},
		},
	}

	diags := bundle.Apply(t.Context(), b, mutator.IgnoreDirectOnlyResources())
	assert.Empty(t, diags)
	assert.Len(t, b.Config.Resources.Jobs, 1)
}

func TestIgnoreDirectOnlyResourcesRemovesThem(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"my_job": {},
				},
				Catalogs: map[string]*resources.Catalog{
					"my_catalog": {CreateCatalog: catalog.CreateCatalog{Name: "my_catalog"}},
				},
				InstancePools: map[string]*resources.InstancePool{
					"my_pool": {CreateInstancePool: compute.CreateInstancePool{InstancePoolName: "my_pool"}},
				},
			},
		},
	}

	diags := bundle.Apply(t.Context(), b, mutator.IgnoreDirectOnlyResources())
	require.Len(t, diags, 2)
	assert.Equal(t, "ignoring catalog resources in this deploy", diags[0].Summary)
	assert.Equal(t, "ignoring instance_pool resources in this deploy", diags[1].Summary)

	assert.Empty(t, b.Config.Resources.Catalogs)
	assert.Empty(t, b.Config.Resources.InstancePools)
	assert.Len(t, b.Config.Resources.Jobs, 1)
}
