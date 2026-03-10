package mutator_test

import (
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/engine"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func boolPtr(v bool) *bool {
	return &v
}

func TestValidateLifecycleStartedDirectEngine(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Apps: map[string]*resources.App{
					"my_app": {
						Lifecycle: resources.LifecycleWithStarted{
							Started: boolPtr(true),
						},
					},
				},
			},
		},
	}

	diags := bundle.Apply(t.Context(), b, mutator.ValidateLifecycleStarted(engine.EngineDirect))
	require.NoError(t, diags.Error())
}

func TestValidateLifecycleStartedTerraformEngine(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Apps: map[string]*resources.App{
					"my_app": {
						Lifecycle: resources.LifecycleWithStarted{
							Started: boolPtr(true),
						},
					},
				},
			},
		},
	}

	diags := bundle.Apply(t.Context(), b, mutator.ValidateLifecycleStarted(engine.EngineTerraform))
	require.Error(t, diags.Error())
	assert.Contains(t, diags.Error().Error(), "lifecycle.started is only supported in direct deployment mode")
}

func TestValidateLifecycleStartedFalse(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Clusters: map[string]*resources.Cluster{
					"my_cluster": {
						Lifecycle: resources.LifecycleWithStarted{
							Started: boolPtr(false),
						},
					},
				},
			},
		},
	}

	// started=false should not produce an error even with terraform engine
	diags := bundle.Apply(t.Context(), b, mutator.ValidateLifecycleStarted(engine.EngineTerraform))
	require.NoError(t, diags.Error())
}

func TestValidateLifecycleStartedNotSet(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				SqlWarehouses: map[string]*resources.SqlWarehouse{
					"my_warehouse": {
						Lifecycle: resources.LifecycleWithStarted{},
					},
				},
			},
		},
	}

	// started not set should not produce an error
	diags := bundle.Apply(t.Context(), b, mutator.ValidateLifecycleStarted(engine.EngineTerraform))
	require.NoError(t, diags.Error())
}

func TestValidateLifecycleStartedJobNotSupported(t *testing.T) {
	// Jobs don't have LifecycleWithStarted, so lifecycle.started is not accessible.
	// This test verifies that resources without LifecycleWithStarted don't trigger errors.
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"my_job": {
						BaseResource: resources.BaseResource{
							Lifecycle: resources.Lifecycle{PreventDestroy: false},
						},
					},
				},
			},
		},
	}

	diags := bundle.Apply(t.Context(), b, mutator.ValidateLifecycleStarted(engine.EngineTerraform))
	require.NoError(t, diags.Error())
}
