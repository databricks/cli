package mutator_test

import (
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/databricks/cli/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeBundle(managedState *bool) *bundle.Bundle {
	b := &bundle.Bundle{}
	b.Config.Bundle.Deployment.ManagedState = managedState
	return b
}

//go:fix inline
func boolPtr(v bool) *bool { return new(v) }

func TestApplyManagedStateEnvSetsFromEnvVar(t *testing.T) {
	cases := []struct {
		value string
		want  bool
	}{
		{"true", true},
		{"TRUE", true},
		{"1", true},
		{"yes", true},
		{"on", true},
		{"false", false},
		{"0", false},
		{"no", false},
		{"off", false},
	}
	for _, tc := range cases {
		t.Run(tc.value, func(t *testing.T) {
			testutil.CleanupEnvironment(t)
			t.Setenv("DATABRICKS_BUNDLE_MANAGED_STATE", tc.value)
			b := makeBundle(nil)
			diags := bundle.Apply(t.Context(), b, mutator.ApplyManagedStateEnv())
			require.NoError(t, diags.Error())
			require.NotNil(t, b.Config.Bundle.Deployment.ManagedState)
			assert.Equal(t, tc.want, *b.Config.Bundle.Deployment.ManagedState)
		})
	}
}

func TestApplyManagedStateEnvUnset(t *testing.T) {
	testutil.CleanupEnvironment(t)
	b := makeBundle(nil)
	diags := bundle.Apply(t.Context(), b, mutator.ApplyManagedStateEnv())
	require.NoError(t, diags.Error())
	assert.Nil(t, b.Config.Bundle.Deployment.ManagedState)
}

func TestApplyManagedStateEnvGarbageValueFallsBackToFalse(t *testing.T) {
	testutil.CleanupEnvironment(t)
	t.Setenv("DATABRICKS_BUNDLE_MANAGED_STATE", "garbage")
	b := makeBundle(nil)
	diags := bundle.Apply(t.Context(), b, mutator.ApplyManagedStateEnv())
	require.NoError(t, diags.Error())
	// GetBool treats unrecognised values as false (ok=true), so the field is set.
	require.NotNil(t, b.Config.Bundle.Deployment.ManagedState)
	assert.False(t, *b.Config.Bundle.Deployment.ManagedState)
}

func TestApplyManagedStateEnvConfigTakesPriority(t *testing.T) {
	cases := []struct {
		name        string
		configValue bool
		envValue    string
	}{
		{"config true, env false", true, "false"},
		{"config false, env true", false, "true"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			testutil.CleanupEnvironment(t)
			t.Setenv("DATABRICKS_BUNDLE_MANAGED_STATE", tc.envValue)
			b := makeBundle(new(tc.configValue))
			diags := bundle.Apply(t.Context(), b, mutator.ApplyManagedStateEnv())
			require.NoError(t, diags.Error())
			require.NotNil(t, b.Config.Bundle.Deployment.ManagedState)
			assert.Equal(t, tc.configValue, *b.Config.Bundle.Deployment.ManagedState)
		})
	}
}

// Verify the config field is accessible at the expected path in databricks.yml.
func TestApplyManagedStateEnvConfigPath(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Bundle: config.Bundle{
				Deployment: config.Deployment{
					ManagedState: new(true),
				},
			},
		},
	}
	assert.True(t, *b.Config.Bundle.Deployment.ManagedState)
}
