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
	t.Setenv("BUNDLE_VAR_b", "def")
	b := load(t, "./variables")
	err := bundle.Apply(context.Background(), b, []bundle.Mutator{
		mutator.SetVariables(),
		interpolation.Interpolate(
			interpolation.IncludeVariableLookups(),
		)})
	require.NoError(t, err)
	assert.Equal(t, "abc def", b.Config.Bundle.Name)
}

func TestVariablesLoadingFailsWhenRequiredVariableIsNotSpecified(t *testing.T) {
	b := load(t, "./variables")
	err := bundle.Apply(context.Background(), b, []bundle.Mutator{
		mutator.SetVariables(),
		interpolation.Interpolate(
			interpolation.IncludeVariableLookups(),
		)})
	assert.ErrorContains(t, err, "no value assigned to required variable b. Assignment can be done through the \"--var\" flag or by setting the BUNDLE_VAR_b environment variable")
}
