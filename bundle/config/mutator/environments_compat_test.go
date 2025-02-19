package mutator_test

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnvironmentsToTargetsWithBothDefined(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Environments: map[string]*config.Target{
				"name": {
					Mode: config.Development,
				},
			},
			Targets: map[string]*config.Target{
				"name": {
					Mode: config.Development,
				},
			},
		},
	}

	diags := bundle.Apply(context.Background(), b, mutator.EnvironmentsToTargets())
	assert.ErrorContains(t, diags.Error(), `both 'environments' and 'targets' are specified;`)
}

func TestEnvironmentsToTargetsWithEnvironmentsDefined(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Environments: map[string]*config.Target{
				"name": {
					Mode: config.Development,
				},
			},
		},
	}

	diags := bundle.Apply(context.Background(), b, mutator.EnvironmentsToTargets())
	require.NoError(t, diags.Error())
	assert.Empty(t, b.Config.Environments)
	assert.Len(t, b.Config.Targets, 1)
}

func TestEnvironmentsToTargetsWithTargetsDefined(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Targets: map[string]*config.Target{
				"name": {
					Mode: config.Development,
				},
			},
		},
	}

	diags := bundle.Apply(context.Background(), b, mutator.EnvironmentsToTargets())
	require.NoError(t, diags.Error())
	assert.Empty(t, b.Config.Environments)
	assert.Len(t, b.Config.Targets, 1)
}
