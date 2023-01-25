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

func TestDefaultWorkspaceRoot(t *testing.T) {
	bundle := &bundle.Bundle{
		Config: config.Root{
			Bundle: config.Bundle{
				Name:        "name",
				Environment: "environment",
			},
		},
	}
	_, err := mutator.DefineDefaultWorkspaceRoot().Apply(context.Background(), bundle)
	require.NoError(t, err)
	assert.Equal(t, "~/.bundle/name/environment", bundle.Config.Workspace.Root)
}
