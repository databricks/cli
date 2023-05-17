package config_tests

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/interpolation"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/databricks/cli/bundle/config/variable"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVariables(t *testing.T) {
	t.Setenv("BUNDLE_VAR_b", "def")
	b := load(t, "./variables/vanilla")
	err := bundle.Apply(context.Background(), b, []bundle.Mutator{
		mutator.SetVariables(),
		interpolation.Interpolate(
			interpolation.IncludeLookupsInPath(variable.VariableReferencePrefix),
		)})
	require.NoError(t, err)
	assert.Equal(t, "abc def", b.Config.Bundle.Name)
}

func TestVariablesLoadingFailsWhenRequiredVariableIsNotSpecified(t *testing.T) {
	b := load(t, "./variables/vanilla")
	err := bundle.Apply(context.Background(), b, []bundle.Mutator{
		mutator.SetVariables(),
		interpolation.Interpolate(
			interpolation.IncludeLookupsInPath(variable.VariableReferencePrefix),
		)})
	assert.ErrorContains(t, err, "no value assigned to required variable b. Assignment can be done through the \"--var\" flag or by setting the BUNDLE_VAR_b environment variable")
}

func TestVariablesEnvironmentsBlockOverride(t *testing.T) {
	b := load(t, "./variables/env_overrides")
	err := bundle.Apply(context.Background(), b, []bundle.Mutator{
		mutator.SelectEnvironment("env-with-single-variable-override"),
		mutator.SetVariables(),
		interpolation.Interpolate(
			interpolation.IncludeLookupsInPath(variable.VariableReferencePrefix),
		)})
	require.NoError(t, err)
	assert.Equal(t, "default-a dev-b", b.Config.Workspace.Profile)
}

func TestVariablesEnvironmentsBlockOverrideForMultipleVariables(t *testing.T) {
	b := load(t, "./variables/env_overrides")
	err := bundle.Apply(context.Background(), b, []bundle.Mutator{
		mutator.SelectEnvironment("env-with-two-variable-overrides"),
		mutator.SetVariables(),
		interpolation.Interpolate(
			interpolation.IncludeLookupsInPath(variable.VariableReferencePrefix),
		)})
	require.NoError(t, err)
	assert.Equal(t, "prod-a prod-b", b.Config.Workspace.Profile)
}

func TestVariablesEnvironmentsBlockOverrideWithProcessEnvVars(t *testing.T) {
	t.Setenv("BUNDLE_VAR_b", "env-var-b")
	b := load(t, "./variables/env_overrides")
	err := bundle.Apply(context.Background(), b, []bundle.Mutator{
		mutator.SelectEnvironment("env-with-two-variable-overrides"),
		mutator.SetVariables(),
		interpolation.Interpolate(
			interpolation.IncludeLookupsInPath(variable.VariableReferencePrefix),
		)})
	require.NoError(t, err)
	assert.Equal(t, "prod-a env-var-b", b.Config.Workspace.Profile)
}

func TestVariablesEnvironmentsBlockOverrideWithMissingVariables(t *testing.T) {
	b := load(t, "./variables/env_overrides")
	err := bundle.Apply(context.Background(), b, []bundle.Mutator{
		mutator.SelectEnvironment("env-missing-a-required-variable-assignment"),
		mutator.SetVariables(),
		interpolation.Interpolate(
			interpolation.IncludeLookupsInPath(variable.VariableReferencePrefix),
		)})
	assert.ErrorContains(t, err, "no value assigned to required variable b. Assignment can be done through the \"--var\" flag or by setting the BUNDLE_VAR_b environment variable")
}

func TestVariablesEnvironmentsBlockOverrideWithUndefinedVariables(t *testing.T) {
	b := load(t, "./variables/env_overrides")
	err := bundle.Apply(context.Background(), b, []bundle.Mutator{
		mutator.SelectEnvironment("env-using-an-undefined-variable"),
		mutator.SetVariables(),
		interpolation.Interpolate(
			interpolation.IncludeLookupsInPath(variable.VariableReferencePrefix),
		)})
	assert.ErrorContains(t, err, "variable c is not defined but is assigned a value")
}
