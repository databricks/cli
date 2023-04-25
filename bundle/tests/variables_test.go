package config_tests

import (
	"context"
	"testing"

	"github.com/databricks/bricks/bundle"
	"github.com/databricks/bricks/bundle/config/interpolation"
	"github.com/databricks/bricks/bundle/config/mutator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVariables(t *testing.T) {
	t.Setenv("BUNDLE_VAR_B", "bar_from_env")
	t.Setenv("BUNDLE_VAR_C", "world")
	b := load(t, "./variables")
	err := bundle.Apply(context.Background(), b, []bundle.Mutator{
		mutator.LoadVariablesFromEnvionment(),
		interpolation.Interpolate(
			interpolation.IncludeLookupsInPath("bundle"),
			interpolation.IncludeLookupsInPath("workspace"),
			interpolation.IncludeLookupsInPath("variables"),
		)})
	require.NoError(t, err)
	assert.Equal(t, "name: foo bar_from_env world", b.Config.Bundle.Name)
	assert.Equal(t, "foo", b.Config.Variables["a"])
	assert.Equal(t, "bar_from_env", b.Config.Variables["b"])
	assert.Equal(t, "world", b.Config.Variables["c"])
}

func TestVariablesWhenUndefinedVarIsUsed(t *testing.T) {
	b := load(t, "./variables")
	err := bundle.Apply(context.Background(), b, []bundle.Mutator{
		mutator.LoadVariablesFromEnvionment(),
		interpolation.Interpolate(
			interpolation.IncludeLookupsInPath("bundle"),
			interpolation.IncludeLookupsInPath("workspace"),
			interpolation.IncludeLookupsInPath("variables"),
		)})
	assert.ErrorContains(t, err, "could not resolve reference variables.c")
}
