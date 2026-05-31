package mutator_test

import (
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/databricks/cli/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplyManagedStateEnvTrue(t *testing.T) {
	testutil.CleanupEnvironment(t)
	t.Setenv("DATABRICKS_BUNDLE_MANAGED_STATE", "true")
	b := &bundle.Bundle{}
	diags := bundle.Apply(t.Context(), b, mutator.ApplyManagedStateEnv())
	require.NoError(t, diags.Error())
	require.NotNil(t, b.Config.Bundle.Deployment.ManagedState)
	assert.True(t, *b.Config.Bundle.Deployment.ManagedState)
}

func TestApplyManagedStateEnvOtherValues(t *testing.T) {
	for _, v := range []string{"false", "1", "yes", "on", "TRUE", "garbage", ""} {
		t.Run(v, func(t *testing.T) {
			testutil.CleanupEnvironment(t)
			t.Setenv("DATABRICKS_BUNDLE_MANAGED_STATE", v)
			b := &bundle.Bundle{}
			diags := bundle.Apply(t.Context(), b, mutator.ApplyManagedStateEnv())
			require.NoError(t, diags.Error())
			assert.Nil(t, b.Config.Bundle.Deployment.ManagedState)
		})
	}
}

func TestApplyManagedStateEnvUnset(t *testing.T) {
	testutil.CleanupEnvironment(t)
	b := &bundle.Bundle{}
	diags := bundle.Apply(t.Context(), b, mutator.ApplyManagedStateEnv())
	require.NoError(t, diags.Error())
	assert.Nil(t, b.Config.Bundle.Deployment.ManagedState)
}

func TestApplyManagedStateEnvConfigTakesPriority(t *testing.T) {
	testutil.CleanupEnvironment(t)
	t.Setenv("DATABRICKS_BUNDLE_MANAGED_STATE", "true")
	disabled := false
	b := &bundle.Bundle{}
	b.Config.Bundle.Deployment.ManagedState = &disabled
	diags := bundle.Apply(t.Context(), b, mutator.ApplyManagedStateEnv())
	require.NoError(t, diags.Error())
	assert.False(t, *b.Config.Bundle.Deployment.ManagedState)
}
