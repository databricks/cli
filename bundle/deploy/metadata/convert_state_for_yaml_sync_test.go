package metadata

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/engine"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/env"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertStateForYamlSyncDisabled(t *testing.T) {
	ctx := context.Background()
	b := &bundle.Bundle{
		BundleRootPath: t.TempDir(),
		Config: config.Root{
			Bundle: config.Bundle{
				Target: "default",
			},
		},
	}

	mutator := ConvertStateForYamlSync(engine.EngineTerraform)
	diags := bundle.Apply(ctx, b, mutator)
	require.NoError(t, diags.Error())

	// Verify no snapshot file was created
	_, snapshotPath := b.StateFilenameConfigSnapshot(ctx)
	_, err := os.Stat(snapshotPath)
	assert.True(t, os.IsNotExist(err), "snapshot file should not exist when env var is disabled")
}

func TestConvertStateForYamlSyncWithDirectEngine(t *testing.T) {
	ctx := context.Background()
	ctx = env.Set(ctx, "DATABRICKS_BUNDLE_ENABLE_EXPERIMENTAL_YAML_SYNC", "true")

	b := &bundle.Bundle{
		BundleRootPath: t.TempDir(),
		Config: config.Root{
			Bundle: config.Bundle{
				Target: "default",
			},
		},
	}

	mutator := ConvertStateForYamlSync(engine.EngineDirect)
	diags := bundle.Apply(ctx, b, mutator)
	require.NoError(t, diags.Error())

	// Verify no snapshot file was created (direct engine already has direct state)
	_, snapshotPath := b.StateFilenameConfigSnapshot(ctx)
	_, err := os.Stat(snapshotPath)
	assert.True(t, os.IsNotExist(err), "snapshot file should not exist when using direct engine")
}

func TestConvertStateForYamlSyncWithTerraformStateError(t *testing.T) {
	ctx := context.Background()
	ctx = env.Set(ctx, "DATABRICKS_BUNDLE_ENABLE_EXPERIMENTAL_YAML_SYNC", "true")

	b := &bundle.Bundle{
		BundleRootPath: t.TempDir(),
		Config: config.Root{
			Bundle: config.Bundle{
				Target: "default",
			},
		},
	}

	// No terraform state exists, so conversion should fail gracefully
	mutator := ConvertStateForYamlSync(engine.EngineTerraform)
	diags := bundle.Apply(ctx, b, mutator)

	// Should not return error - failures are logged as warnings
	require.NoError(t, diags.Error())

	// Verify no snapshot file was created
	_, snapshotPath := b.StateFilenameConfigSnapshot(ctx)
	_, err := os.Stat(snapshotPath)
	assert.True(t, os.IsNotExist(err), "snapshot file should not exist when terraform state is missing")
}

func TestConvertStateForYamlSyncName(t *testing.T) {
	mutator := ConvertStateForYamlSync(engine.EngineTerraform)
	assert.Equal(t, "metadata.ConvertStateForYamlSync", mutator.Name())
}

func TestStateFilenameConfigSnapshot(t *testing.T) {
	tempDir := t.TempDir()
	b := &bundle.Bundle{
		BundleRootPath: tempDir,
		Config: config.Root{
			Bundle: config.Bundle{
				Target: "dev",
			},
		},
	}

	ctx := context.Background()
	remotePath, localPath := b.StateFilenameConfigSnapshot(ctx)

	assert.Equal(t, "resources-config-sync-snapshot.json", remotePath)
	assert.Contains(t, localPath, filepath.Join(".databricks", "bundle", "dev", "resources-config-sync-snapshot.json"))
}

func TestReverseInterpolatePreservesBConfigValue(t *testing.T) {
	// This test verifies that our approach of getting b.Config.Value(),
	// reverse interpolating it, and wrapping in a new config.Root
	// does NOT mutate b.Config

	b := &bundle.Bundle{
		Config: config.Root{
			Bundle: config.Bundle{
				Name: "test",
			},
		},
	}

	// Create a config with terraform-style references
	err := b.Config.Mutate(func(_ dyn.Value) (dyn.Value, error) {
		return dyn.V(map[string]dyn.Value{
			"bundle": dyn.V(map[string]dyn.Value{
				"name": dyn.V("test"),
			}),
			"resources": dyn.V(map[string]dyn.Value{
				"jobs": dyn.V(map[string]dyn.Value{
					"my_job": dyn.V(map[string]dyn.Value{
						"name":       dyn.V("My Job"),
						"depends_on": dyn.V("${databricks_pipeline.my_pipeline.id}"),
					}),
				}),
			}),
		}), nil
	})
	require.NoError(t, err)

	// Store the original value as JSON for comparison
	originalValue := b.Config.Value()
	originalJSON, err := json.Marshal(originalValue.AsAny())
	require.NoError(t, err)

	// Now do what convert_state_for_yaml_sync.go does:
	// 1. Get the value from b.Config
	interpolatedRoot := b.Config.Value()

	// 2. Reverse interpolate it (creates NEW dyn.Value)
	uninterpolatedRoot, err := reverseInterpolate(interpolatedRoot)
	require.NoError(t, err)

	// Verify reverse interpolation worked
	dependsOn, err := dyn.GetByPath(uninterpolatedRoot, dyn.MustPathFromString("resources.jobs.my_job.depends_on"))
	require.NoError(t, err)
	dependsOnStr, ok := dependsOn.AsString()
	require.True(t, ok)
	assert.Equal(t, "${resources.pipelines.my_pipeline.id}", dependsOnStr, "should be bundle-style after reverse interpolation")

	// 3. Create a temporary config.Root wrapping the uninterpolated value
	var uninterpolatedConfig config.Root
	err = uninterpolatedConfig.Mutate(func(_ dyn.Value) (dyn.Value, error) {
		return uninterpolatedRoot, nil
	})
	require.NoError(t, err)

	// CRITICAL TEST: Verify b.Config.Value() is unchanged
	afterValue := b.Config.Value()
	afterJSON, err := json.Marshal(afterValue.AsAny())
	require.NoError(t, err)

	assert.Equal(t, string(originalJSON), string(afterJSON), "b.Config.Value() should not change")

	// Verify terraform-style reference is still present in b.Config
	originalDependsOn, err := dyn.GetByPath(afterValue, dyn.MustPathFromString("resources.jobs.my_job.depends_on"))
	require.NoError(t, err)
	originalDependsOnStr, ok := originalDependsOn.AsString()
	require.True(t, ok)
	assert.Equal(t, "${databricks_pipeline.my_pipeline.id}", originalDependsOnStr, "terraform-style reference should be preserved in b.Config")
}
