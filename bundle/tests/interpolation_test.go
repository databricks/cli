package config_tests

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/interpolation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInterpolation(t *testing.T) {
	b := load(t, "./interpolation")
	err := bundle.Apply(context.Background(), b, interpolation.Interpolate(
		interpolation.IncludeLookupsInPath("bundle"),
		interpolation.IncludeLookupsInPath("workspace"),
	))
	require.NoError(t, err)
	assert.Equal(t, "foo bar", b.Config.Bundle.Name)
	assert.Equal(t, "foo bar | bar", b.Config.Resources.Jobs["my_job"].Name)
}
