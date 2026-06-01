package bundle_test

import (
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/engine"
	"github.com/databricks/cli/internal/testutil"
	"github.com/stretchr/testify/assert"
)

func TestIsManagedStateEnvVar(t *testing.T) {
	testutil.CleanupEnvironment(t)
	t.Setenv("DATABRICKS_BUNDLE_ENGINE", "direct_with_history")
	b := &bundle.Bundle{}
	assert.True(t, bundle.IsManagedState(t.Context(), b))
}

func TestIsManagedStateOtherEngines(t *testing.T) {
	for _, v := range []string{"direct", "terraform", ""} {
		t.Run(v, func(t *testing.T) {
			testutil.CleanupEnvironment(t)
			t.Setenv("DATABRICKS_BUNDLE_ENGINE", v)
			b := &bundle.Bundle{}
			assert.False(t, bundle.IsManagedState(t.Context(), b))
		})
	}
}

func TestIsManagedStateConfig(t *testing.T) {
	testutil.CleanupEnvironment(t)
	b := &bundle.Bundle{
		Config: config.Root{
			Bundle: config.Bundle{
				Engine: engine.EngineDirectWithHistory,
			},
		},
	}
	assert.True(t, bundle.IsManagedState(t.Context(), b))
}

func TestIsManagedStateConfigTakesPriority(t *testing.T) {
	testutil.CleanupEnvironment(t)
	t.Setenv("DATABRICKS_BUNDLE_ENGINE", "direct_with_history")
	b := &bundle.Bundle{
		Config: config.Root{
			Bundle: config.Bundle{
				Engine: engine.EngineDirect,
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
