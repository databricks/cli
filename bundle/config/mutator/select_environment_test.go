package mutator_test

import (
	"context"
	"testing"

	"github.com/databricks/bricks/bundle"
	"github.com/databricks/bricks/bundle/config"
	"github.com/databricks/bricks/bundle/config/mutator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSelectEnvironment(t *testing.T) {
	bundle := &bundle.Bundle{
		Config: config.Root{
			Workspace: config.Workspace{
				Host: "foo",
			},
			Environments: map[string]*config.Environment{
				"default": {
					Workspace: &config.Workspace{
						Host: "bar",
					},
				},
			},
		},
	}
	_, err := mutator.SelectEnvironment("default").Apply(context.Background(), bundle)
	require.NoError(t, err)
	assert.Equal(t, "bar", bundle.Config.Workspace.Host)
}

func TestSelectEnvironmentNotFound(t *testing.T) {
	bundle := &bundle.Bundle{
		Config: config.Root{
			Environments: map[string]*config.Environment{
				"default": {},
			},
		},
	}
	_, err := mutator.SelectEnvironment("doesnt-exist").Apply(context.Background(), bundle)
	require.Error(t, err, "no environments defined")
}
