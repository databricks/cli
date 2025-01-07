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
	"github.com/databricks/databricks-sdk-go/service/dashboards"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigureDashboardDefaultsParentPath(t *testing.T) {
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
						Dashboard: &dashboards.Dashboard{
							ParentPath: "",
						},
					},
					"d2": {
						// Non-empty string is skipped.
						Dashboard: &dashboards.Dashboard{
							ParentPath: "already-set",
						},
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

	diags := bundle.Apply(context.Background(), b, mutator.ConfigureDashboardDefaults())
	require.NoError(t, diags.Error())

	var v dyn.Value
	var err error

	// Set to empty string; unchanged.
	v, err = dyn.Get(b.Config.Value(), "resources.dashboards.d1.parent_path")
	if assert.NoError(t, err) {
		assert.Equal(t, "", v.MustString())
	}

	// Set to "already-set"; unchanged.
	v, err = dyn.Get(b.Config.Value(), "resources.dashboards.d2.parent_path")
	if assert.NoError(t, err) {
		assert.Equal(t, "already-set", v.MustString())
	}

	// Not set; now set to the workspace resource path.
	v, err = dyn.Get(b.Config.Value(), "resources.dashboards.d3.parent_path")
	if assert.NoError(t, err) {
		assert.Equal(t, "/foo/bar", v.MustString())
	}

	// No valid dashboard; no change.
	_, err = dyn.Get(b.Config.Value(), "resources.dashboards.d4.parent_path")
	assert.True(t, dyn.IsCannotTraverseNilError(err))
}

func TestConfigureDashboardDefaultsEmbedCredentials(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Dashboards: map[string]*resources.Dashboard{
					"d1": {
						EmbedCredentials: true,
					},
					"d2": {
						EmbedCredentials: false,
					},
					"d3": {
						// No parent path set.
					},
					"d4": nil,
				},
			},
		},
	}

	diags := bundle.Apply(context.Background(), b, mutator.ConfigureDashboardDefaults())
	require.NoError(t, diags.Error())

	var v dyn.Value
	var err error

	// Set to true; still true.
	v, err = dyn.Get(b.Config.Value(), "resources.dashboards.d1.embed_credentials")
	if assert.NoError(t, err) {
		assert.True(t, v.MustBool())
	}

	// Set to false; still false.
	v, err = dyn.Get(b.Config.Value(), "resources.dashboards.d2.embed_credentials")
	if assert.NoError(t, err) {
		assert.False(t, v.MustBool())
	}

	// Not set; now false.
	v, err = dyn.Get(b.Config.Value(), "resources.dashboards.d3.embed_credentials")
	if assert.NoError(t, err) {
		assert.False(t, v.MustBool())
	}

	// No valid dashboard; no change.
	_, err = dyn.Get(b.Config.Value(), "resources.dashboards.d4.embed_credentials")
	assert.True(t, dyn.IsCannotTraverseNilError(err))
}
