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

func TestSelectTarget(t *testing.T) {
	bundle := &bundle.Bundle{
		Config: config.Root{
			Workspace: &config.Workspace{
				Host: "foo",
			},
			Targets: map[string]*config.Target{
				"default": {
					Workspace: &config.Workspace{
						Host: "bar",
					},
				},
			},
		},
	}
	err := mutator.SelectTarget("default").Apply(context.Background(), bundle)
	require.NoError(t, err)
	assert.Equal(t, "bar", bundle.Config.Workspace.Host)
}

func TestSelectTargetNotFound(t *testing.T) {
	bundle := &bundle.Bundle{
		Config: config.Root{
			Targets: map[string]*config.Target{
				"default": {},
			},
		},
	}
	err := mutator.SelectTarget("doesnt-exist").Apply(context.Background(), bundle)
	require.Error(t, err, "no targets defined")
}
