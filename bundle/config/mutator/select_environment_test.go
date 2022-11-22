package mutator_test

import (
	"testing"

	"github.com/databricks/bricks/bundle/config"
	"github.com/databricks/bricks/bundle/config/mutator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSelectEnvironment(t *testing.T) {
	root := &config.Root{
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
	}
	_, err := mutator.SelectEnvironment("default").Apply(root)
	require.NoError(t, err)
	assert.Equal(t, "bar", root.Workspace.Host)
}

func TestSelectEnvironmentNotFound(t *testing.T) {
	root := &config.Root{
		Environments: map[string]*config.Environment{
			"default": {},
		},
	}
	_, err := mutator.SelectEnvironment("doesnt-exist").Apply(root)
	require.Error(t, err, "no environments defined")
}
