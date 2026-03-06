package mutator

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/engine"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/databricks/databricks-sdk-go/service/sql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func boolPtr(b bool) *bool { return &b }

func TestValidateLifecycleStarted_UnsupportedResource(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"my_job": {
						BaseResource: resources.BaseResource{
							Lifecycle: resources.Lifecycle{
								Started: boolPtr(true),
							},
						},
						JobSettings: jobs.JobSettings{Name: "my_job"},
					},
				},
			},
		},
	}

	m := bundle.Apply(context.Background(), b, ValidateLifecycleStarted(engine.EngineDirect))
	require.Error(t, m.Error())
	assert.Contains(t, m.Error().Error(), "lifecycle.started is not supported for resources.jobs.my_job")
}

func TestValidateLifecycleStarted_SupportedResources(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				SqlWarehouses: map[string]*resources.SqlWarehouse{
					"my_warehouse": {
						BaseResource: resources.BaseResource{
							Lifecycle: resources.Lifecycle{
								Started: boolPtr(true),
							},
						},
						CreateWarehouseRequest: sql.CreateWarehouseRequest{
							Name: "my_warehouse",
						},
					},
				},
			},
		},
	}

	m := bundle.Apply(context.Background(), b, ValidateLifecycleStarted(engine.EngineDirect))
	assert.NoError(t, m.Error())
}

func TestValidateLifecycleStarted_StartedFalse(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"my_job": {
						BaseResource: resources.BaseResource{
							Lifecycle: resources.Lifecycle{
								Started: boolPtr(false),
							},
						},
						JobSettings: jobs.JobSettings{Name: "my_job"},
					},
				},
			},
		},
	}

	m := bundle.Apply(context.Background(), b, ValidateLifecycleStarted(engine.EngineDirect))
	assert.NoError(t, m.Error())
}

func TestValidateLifecycleStarted_TerraformModeIgnored(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"my_job": {
						BaseResource: resources.BaseResource{
							Lifecycle: resources.Lifecycle{
								Started: boolPtr(true),
							},
						},
						JobSettings: jobs.JobSettings{Name: "my_job"},
					},
				},
			},
		},
	}

	// In TF mode, lifecycle.started is ignored — no error even for unsupported resource types.
	m := bundle.Apply(context.Background(), b, ValidateLifecycleStarted(engine.EngineTerraform))
	assert.NoError(t, m.Error())
}
