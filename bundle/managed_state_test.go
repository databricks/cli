package bundle_test

import (
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/internal/testutil"
	"github.com/stretchr/testify/assert"
)

func TestIsManagedStateEnvVar(t *testing.T) {
	testutil.CleanupEnvironment(t)
	t.Setenv("DATABRICKS_BUNDLE_MANAGED_STATE", "true")
	b := &bundle.Bundle{}
	assert.True(t, bundle.IsManagedState(t.Context(), b))
}

func TestIsManagedStateEnvVarOtherValues(t *testing.T) {
	for _, v := range []string{"false", "1", "yes", "on", "TRUE", "garbage", ""} {
		t.Run(v, func(t *testing.T) {
			testutil.CleanupEnvironment(t)
			t.Setenv("DATABRICKS_BUNDLE_MANAGED_STATE", v)
			b := &bundle.Bundle{}
			assert.False(t, bundle.IsManagedState(t.Context(), b))
		})
	}
}

func TestIsManagedStateConfigTrue(t *testing.T) {
	testutil.CleanupEnvironment(t)
	enabled := true
	b := &bundle.Bundle{
		Config: config.Root{
			Bundle: config.Bundle{
				Deployment: config.Deployment{
					ManagedState: &enabled,
				},
			},
		},
	}
	assert.True(t, bundle.IsManagedState(t.Context(), b))
}

func TestIsManagedStateConfigTakesPriority(t *testing.T) {
	testutil.CleanupEnvironment(t)
	t.Setenv("DATABRICKS_BUNDLE_MANAGED_STATE", "true")
	disabled := false
	b := &bundle.Bundle{
		Config: config.Root{
			Bundle: config.Bundle{
				Deployment: config.Deployment{
					ManagedState: &disabled,
				},
			},
		},
	}
	assert.False(t, bundle.IsManagedState(t.Context(), b))
}

func TestIsManagedStateUnset(t *testing.T) {
	testutil.CleanupEnvironment(t)
	b := &bundle.Bundle{}
	assert.False(t, bundle.IsManagedState(t.Context(), b))
}
