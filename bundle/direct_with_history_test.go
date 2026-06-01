package bundle_test

import (
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/engine"
	"github.com/databricks/cli/internal/testutil"
	"github.com/stretchr/testify/assert"
)

func TestIsDirectWithHistoryEnvVar(t *testing.T) {
	testutil.CleanupEnvironment(t)
	t.Setenv("DATABRICKS_BUNDLE_ENGINE", "direct_with_history")
	b := &bundle.Bundle{}
	assert.True(t, bundle.IsDirectWithHistory(t.Context(), b))
}

func TestIsDirectWithHistoryOtherEngines(t *testing.T) {
	for _, v := range []string{"direct", "terraform", ""} {
		t.Run(v, func(t *testing.T) {
			testutil.CleanupEnvironment(t)
			t.Setenv("DATABRICKS_BUNDLE_ENGINE", v)
			b := &bundle.Bundle{}
			assert.False(t, bundle.IsDirectWithHistory(t.Context(), b))
		})
	}
}

func TestIsDirectWithHistoryConfig(t *testing.T) {
	testutil.CleanupEnvironment(t)
	b := &bundle.Bundle{
		Config: config.Root{
			Bundle: config.Bundle{
				Engine: engine.EngineDirectWithHistory,
			},
		},
	}
	assert.True(t, bundle.IsDirectWithHistory(t.Context(), b))
}

func TestIsDirectWithHistoryConfigTakesPriority(t *testing.T) {
	testutil.CleanupEnvironment(t)
	t.Setenv("DATABRICKS_BUNDLE_ENGINE", "direct_with_history")
	b := &bundle.Bundle{
		Config: config.Root{
			Bundle: config.Bundle{
				Engine: engine.EngineDirect,
			},
		},
	}
	assert.False(t, bundle.IsDirectWithHistory(t.Context(), b))
}

func TestIsDirectWithHistoryUnset(t *testing.T) {
	testutil.CleanupEnvironment(t)
	b := &bundle.Bundle{}
	assert.False(t, bundle.IsDirectWithHistory(t.Context(), b))
}
