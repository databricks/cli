package mutator_test

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/internal/bundletest"
	"github.com/databricks/cli/libs/dyn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigureDefaultParentPath(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Workspace: config.Workspace{
				ResourcePath: "/foo/bar",
			},
			Resources: config.Resources{
				Dashboards: map[string]*resources.Dashboard{
					"d1": {
						// Empty string is skipped.
						// See below for how it is set.
						ParentPath: "",
					},
					"d2": {
						// Non-empty string is skipped.
						ParentPath: "already-set",
					},
					"d3": {
						// No parent path set.
					},
					"d4": nil,
				},
			},
		},
	}

	// We can't set an empty string in the typed configuration.
	// Do it on the dyn.Value directly.
	bundletest.Mutate(t, b, func(v dyn.Value) (dyn.Value, error) {
		return dyn.Set(v, "resources.dashboards.d1.parent_path", dyn.V(""))
	})

	diags := bundle.Apply(context.Background(), b, mutator.ConfigureDefaultParentPath())
	require.NoError(t, diags.Error())

	assert.Equal(t, "", b.Config.Resources.Dashboards["d1"].ParentPath)
	assert.Equal(t, "already-set", b.Config.Resources.Dashboards["d2"].ParentPath)
	assert.Equal(t, "/foo/bar", b.Config.Resources.Dashboards["d3"].ParentPath)
	assert.Nil(t, b.Config.Resources.Dashboards["d4"])
}
