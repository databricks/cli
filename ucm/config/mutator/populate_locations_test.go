package mutator_test

import (
	"path/filepath"
	"testing"

	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/ucm"
	"github.com/databricks/cli/ucm/config"
	"github.com/databricks/cli/ucm/config/mutator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPopulateLocationsAddsLocationsField(t *testing.T) {
	root := filepath.Join("testdata", "populate_locations")
	cfgFile := filepath.Join(root, "ucm.yml")

	cfg, diags := config.Load(cfgFile)
	require.False(t, diags.HasError(), "failed to load fixture: %v", diags)

	u := &ucm.Ucm{RootPath: root, Config: *cfg}

	applied := ucm.Apply(t.Context(), u, mutator.PopulateLocations())
	require.Empty(t, applied)

	require.NotNil(t, u.Config.Locations, "PopulateLocations should attach Locations")
	assert.NotEmpty(t, u.Config.Locations.Files, "expected at least the fixture file")
	assert.Contains(t, u.Config.Locations.Locations, "resources.catalogs.team_alpha")
}

func TestPopulateLocationsNoopOnEmptyConfig(t *testing.T) {
	u := &ucm.Ucm{Config: config.Root{}}

	// Seed an empty dynamic tree so Value() is valid.
	require.NoError(t, u.Config.Mutate(func(v dyn.Value) (dyn.Value, error) {
		return v, nil
	}))
	diags := ucm.Apply(t.Context(), u, mutator.PopulateLocations())
	require.Empty(t, diags)

	require.NotNil(t, u.Config.Locations)
	// No YAML-sourced paths — Files stays empty.
	assert.Empty(t, u.Config.Locations.Files)
}
