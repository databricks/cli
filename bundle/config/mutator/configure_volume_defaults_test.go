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
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigureVolumeDefaultsVolumeType(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Volumes: map[string]*resources.Volume{
					"v1": {
						// Empty string is skipped.
						// See below for how it is set.
						CreateVolumeRequestContent: &catalog.CreateVolumeRequestContent{
							VolumeType: "",
						},
					},
					"v2": {
						// Non-empty string is skipped.
						CreateVolumeRequestContent: &catalog.CreateVolumeRequestContent{
							VolumeType: "already-set",
						},
					},
					"v3": {
						// No volume type set.
					},
					"v4": nil,
				},
			},
		},
	}

	// We can't set an empty string in the typed configuration.
	// Do it on the dyn.Value directly.
	bundletest.Mutate(t, b, func(v dyn.Value) (dyn.Value, error) {
		return dyn.Set(v, "resources.volumes.v1.volume_type", dyn.V(""))
	})

	diags := bundle.Apply(context.Background(), b, mutator.ConfigureVolumeDefaults())
	require.NoError(t, diags.Error())

	var v dyn.Value
	var err error

	// Set to empty string; unchanged.
	v, err = dyn.Get(b.Config.Value(), "resources.volumes.v1.volume_type")
	require.NoError(t, err)
	assert.Equal(t, "", v.MustString())

	// Set to non-empty string; unchanged.
	v, err = dyn.Get(b.Config.Value(), "resources.volumes.v2.volume_type")
	require.NoError(t, err)
	assert.Equal(t, "already-set", v.MustString())

	// Not set; set to default.
	v, err = dyn.Get(b.Config.Value(), "resources.volumes.v3.volume_type")
	require.NoError(t, err)
	assert.Equal(t, "MANAGED", v.MustString())

	// No valid volume; No change.
	_, err = dyn.Get(b.Config.Value(), "resources.volumes.v4.volume_type")
	assert.True(t, dyn.IsCannotTraverseNilError(err))
}
