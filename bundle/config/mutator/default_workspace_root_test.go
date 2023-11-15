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

func TestDefaultWorkspaceRoot(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Bundle: config.Bundle{
				Name:   "name",
				Target: "environment",
			},
		},
	}
	err := mutator.DefineDefaultWorkspaceRoot().Apply(context.Background(), b)
	require.NoError(t, err)
	assert.Equal(t, "~/.bundle/name/environment", b.Config.Workspace.RootPath)
}
