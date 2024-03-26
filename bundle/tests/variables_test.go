package config_tests

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVariables(t *testing.T) {
	t.Setenv("BUNDLE_VAR_b", "def")
	b := load(t, "./variables/vanilla")
	diags := bundle.Apply(context.Background(), b, bundle.Seq(
		mutator.SetVariables(),
		mutator.ResolveVariableReferences(
			"variables",
		),
	))
	require.NoError(t, diags.Error())
	assert.Equal(t, "abc def", b.Config.Bundle.Name)
}

func TestVariablesLoadingFailsWhenRequiredVariableIsNotSpecified(t *testing.T) {
	b := load(t, "./variables/vanilla")
	diags := bundle.Apply(context.Background(), b, bundle.Seq(
		mutator.SetVariables(),
		mutator.ResolveVariableReferences(
			"variables",
		),
	))
	assert.ErrorContains(t, diags.Error(), "no value assigned to required variable b. Assignment can be done through the \"--var\" flag or by setting the BUNDLE_VAR_b environment variable")
}

func TestVariablesTargetsBlockOverride(t *testing.T) {
	b := load(t, "./variables/env_overrides")
	diags := bundle.Apply(context.Background(), b, bundle.Seq(
		mutator.SelectTarget("env-with-single-variable-override"),
		mutator.SetVariables(),
		mutator.ResolveVariableReferences(
			"variables",
		),
	))
	require.NoError(t, diags.Error())
	assert.Equal(t, "default-a dev-b", b.Config.Workspace.Profile)
}

func TestVariablesTargetsBlockOverrideForMultipleVariables(t *testing.T) {
	b := load(t, "./variables/env_overrides")
	diags := bundle.Apply(context.Background(), b, bundle.Seq(
		mutator.SelectTarget("env-with-two-variable-overrides"),
		mutator.SetVariables(),
		mutator.ResolveVariableReferences(
			"variables",
		),
	))
	require.NoError(t, diags.Error())
	assert.Equal(t, "prod-a prod-b", b.Config.Workspace.Profile)
}

func TestVariablesTargetsBlockOverrideWithProcessEnvVars(t *testing.T) {
	t.Setenv("BUNDLE_VAR_b", "env-var-b")
	b := load(t, "./variables/env_overrides")
	diags := bundle.Apply(context.Background(), b, bundle.Seq(
		mutator.SelectTarget("env-with-two-variable-overrides"),
		mutator.SetVariables(),
		mutator.ResolveVariableReferences(
			"variables",
		),
	))
	require.NoError(t, diags.Error())
	assert.Equal(t, "prod-a env-var-b", b.Config.Workspace.Profile)
}

func TestVariablesTargetsBlockOverrideWithMissingVariables(t *testing.T) {
	b := load(t, "./variables/env_overrides")
	diags := bundle.Apply(context.Background(), b, bundle.Seq(
		mutator.SelectTarget("env-missing-a-required-variable-assignment"),
		mutator.SetVariables(),
		mutator.ResolveVariableReferences(
			"variables",
		),
	))
	assert.ErrorContains(t, diags.Error(), "no value assigned to required variable b. Assignment can be done through the \"--var\" flag or by setting the BUNDLE_VAR_b environment variable")
}

func TestVariablesTargetsBlockOverrideWithUndefinedVariables(t *testing.T) {
	b := load(t, "./variables/env_overrides")
	diags := bundle.Apply(context.Background(), b, bundle.Seq(
		mutator.SelectTarget("env-using-an-undefined-variable"),
		mutator.SetVariables(),
		mutator.ResolveVariableReferences(
			"variables",
		),
	))
	assert.ErrorContains(t, diags.Error(), "variable c is not defined but is assigned a value")
}

func TestVariablesWithoutDefinition(t *testing.T) {
	t.Setenv("BUNDLE_VAR_a", "foo")
	t.Setenv("BUNDLE_VAR_b", "bar")
	b := load(t, "./variables/without_definition")
	diags := bundle.Apply(context.Background(), b, mutator.SetVariables())
	require.NoError(t, diags.Error())
	require.True(t, b.Config.Variables["a"].HasValue())
	require.True(t, b.Config.Variables["b"].HasValue())
	assert.Equal(t, "foo", *b.Config.Variables["a"].Value)
	assert.Equal(t, "bar", *b.Config.Variables["b"].Value)
}

func TestVariablesWithTargetLookupOverrides(t *testing.T) {
	b := load(t, "./variables/env_overrides")
	diags := bundle.Apply(context.Background(), b, bundle.Seq(
		mutator.SelectTarget("env-overrides-lookup"),
		mutator.SetVariables(),
	))
	require.NoError(t, diags.Error())
	assert.Equal(t, "cluster: some-test-cluster", b.Config.Variables["d"].Lookup.String())
	assert.Equal(t, "instance-pool: some-test-instance-pool", b.Config.Variables["e"].Lookup.String())
}
