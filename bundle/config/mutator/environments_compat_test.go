package mutator_test

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/stretchr/testify/assert"
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

	err := bundle.Apply(context.Background(), b, mutator.EnvironmentsToTargets())
	assert.ErrorContains(t, err, `both 'environments' and 'targets' are specified;`)
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

	err := bundle.Apply(context.Background(), b, mutator.EnvironmentsToTargets())
	assert.NoError(t, err)
	assert.Len(t, b.Config.Environments, 0)
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

	err := bundle.Apply(context.Background(), b, mutator.EnvironmentsToTargets())
	assert.NoError(t, err)
	assert.Len(t, b.Config.Environments, 0)
	assert.Len(t, b.Config.Targets, 1)
}
