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
	diags := bundle.Apply(context.Background(), b, mutator.ResolveVariableReferences(
		"bundle",
		"workspace",
	))
	require.Empty(t, diags)
	assert.Equal(t, "foo bar", b.Config.Bundle.Name)
	assert.Equal(t, "foo bar | bar", b.Config.Resources.Jobs["my_job"].Name)
}

func TestInterpolationWithTarget(t *testing.T) {
	b := loadTarget(t, "./interpolation_target", "development")
	diags := bundle.Apply(context.Background(), b, mutator.ResolveVariableReferences(
		"bundle",
		"workspace",
	))
	require.Empty(t, diags)
	assert.Equal(t, "foo bar", b.Config.Bundle.Name)
	assert.Equal(t, "foo bar | bar | development | development", b.Config.Resources.Jobs["my_job"].Name)
}
