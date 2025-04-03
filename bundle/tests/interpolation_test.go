package config_tests

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInterpolation(t *testing.T) {
	b := load(t, "./interpolation")
	diags := bundle.ApplySeq(
		context.Background(),
		b,
		mutator.ResolveVariableReferencesOnlyResources(
			"bundle",
			"workspace",
		),
		mutator.ResolveVariableReferencesWithoutResources(
			"bundle",
			"workspace",
		),
	)
	require.NoError(t, diags.Error())
	assert.Equal(t, "foo bar", b.Config.Bundle.Name)
	assert.Equal(t, "foo bar | bar", b.Config.Resources.Jobs["my_job"].Name)
}

func TestInterpolationWithTarget_withoutResources(t *testing.T) {
	b := loadTarget(t, "./interpolation_target", "development")
	diags := bundle.Apply(context.Background(), b, mutator.ResolveVariableReferencesOnlyResources(
		"bundle",
		"workspace",
	))
	require.NoError(t, diags.Error())
	assert.Equal(t, "foo ${workspace.profile}", b.Config.Bundle.Name)
	assert.Equal(t, "foo bar | bar | development | development", b.Config.Resources.Jobs["my_job"].Name)
}

func TestInterpolationWithTarget_onlyResources(t *testing.T) {
	b := loadTarget(t, "./interpolation_target", "development")
	diags := bundle.Apply(context.Background(), b, mutator.ResolveVariableReferencesWithoutResources(
		"bundle",
		"workspace",
	))
	require.NoError(t, diags.Error())
	assert.Equal(t, "foo bar", b.Config.Bundle.Name)
	assert.Equal(t, "${bundle.name} | ${workspace.profile} | ${bundle.environment} | ${bundle.target}", b.Config.Resources.Jobs["my_job"].Name)
}
