package phases

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/engine"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/direct/dstate"
	"github.com/databricks/cli/libs/dyn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPipelineDeletionCascades(t *testing.T) {
	// The value must be read from persisted state (what the delete uses), not config.
	b := &bundle.Bundle{}
	data := dstate.NewDatabase("lineage", 1)
	data.State["resources.pipelines.cascade_true"] = dstate.ResourceEntry{ID: "1", State: json.RawMessage(`{"cascade_on_destroy": true}`)}
	data.State["resources.pipelines.cascade_false"] = dstate.ResourceEntry{ID: "2", State: json.RawMessage(`{"cascade_on_destroy": false}`)}
	data.State["resources.pipelines.cascade_unset"] = dstate.ResourceEntry{ID: "3", State: json.RawMessage(`{}`)}
	data.State["resources.pipelines.cascade_empty"] = dstate.ResourceEntry{ID: "4"}
	b.DeploymentBundle.StateDB.OpenWithData(filepath.Join(t.TempDir(), "resources.json"), data)

	// A bundle whose direct state DB was never opened, as happens under the terraform
	// engine. pipelineDeletionCascades must not touch the state DB there.
	bTerraform := &bundle.Bundle{}

	tests := []struct {
		name        string
		bundle      *bundle.Bundle
		engine      engine.EngineType
		resourceKey string
		want        bool
	}{
		{
			name:        "cascade_on_destroy explicitly true",
			bundle:      b,
			engine:      engine.EngineDirect,
			resourceKey: "resources.pipelines.cascade_true",
			want:        true,
		},
		{
			name:        "cascade_on_destroy explicitly false",
			bundle:      b,
			engine:      engine.EngineDirect,
			resourceKey: "resources.pipelines.cascade_false",
			want:        false,
		},
		{
			name:        "cascade_on_destroy unset defaults to cascade",
			bundle:      b,
			engine:      engine.EngineDirect,
			resourceKey: "resources.pipelines.cascade_unset",
			want:        true,
		},
		{
			name:        "empty persisted state defaults to cascade",
			bundle:      b,
			engine:      engine.EngineDirect,
			resourceKey: "resources.pipelines.cascade_empty",
			want:        true,
		},
		{
			name:        "missing state entry defaults to cascade",
			bundle:      b,
			engine:      engine.EngineDirect,
			resourceKey: "resources.pipelines.does_not_exist",
			want:        true,
		},
		{
			name:        "terraform engine defaults to cascade without reading state",
			bundle:      bTerraform,
			engine:      engine.EngineTerraform,
			resourceKey: "resources.pipelines.cascade_false",
			want:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			action := deployplan.Action{
				ResourceKey: tt.resourceKey,
				ActionType:  deployplan.Delete,
			}
			cascade, err := pipelineDeletionCascades(tt.bundle, action, tt.engine)
			require.NoError(t, err)
			assert.Equal(t, tt.want, cascade)
		})
	}
}

func TestCheckPreventDestroyForAllResources(t *testing.T) {
	for resourceType := range config.SupportedResources() {
		t.Run(resourceType, func(t *testing.T) {
			b := &bundle.Bundle{}

			err := b.Config.Mutate(func(v dyn.Value) (dyn.Value, error) {
				return dyn.Set(v, "resources", dyn.NewValue(map[string]dyn.Value{
					resourceType: dyn.NewValue(map[string]dyn.Value{
						"test_resource": dyn.NewValue(map[string]dyn.Value{
							"lifecycle": dyn.NewValue(map[string]dyn.Value{
								"prevent_destroy": dyn.NewValue(true, nil),
							}, nil),
						}, nil),
					}, nil),
				}, nil))
			})
			require.NoError(t, err)

			actions := []deployplan.Action{
				{
					ResourceKey: "resources." + resourceType + ".test_resource",
					ActionType:  deployplan.Recreate,
				},
			}

			err = checkForPreventDestroy(b, actions)
			require.Error(t, err)
			require.Contains(t, err.Error(), "resources."+resourceType+".test_resource has lifecycle.prevent_destroy set")
			require.Contains(t, err.Error(), "but the plan calls for this resource to be recreated or destroyed")
			require.Contains(t, err.Error(), "disable lifecycle.prevent_destroy for resources."+resourceType+".test_resource")
		})
	}
}

func TestCheckPreventDestroyForJob(t *testing.T) {
	b := &bundle.Bundle{}
	err := b.Config.Mutate(func(v dyn.Value) (dyn.Value, error) {
		return dyn.Set(v, "resources", dyn.NewValue(map[string]dyn.Value{
			"jobs": dyn.NewValue(map[string]dyn.Value{
				"test_resource": dyn.NewValue(map[string]dyn.Value{
					"lifecycle": dyn.NewValue(map[string]dyn.Value{
						"prevent_destroy": dyn.NewValue(true, nil),
					}, nil),
				}, nil),
			}, nil),
		}, nil))
	})
	require.NoError(t, err)

	actions := []deployplan.Action{
		{
			ResourceKey: "resources.jobs.test_resource",
			ActionType:  deployplan.Recreate,
		},
	}

	err = checkForPreventDestroy(b, actions)
	require.Error(t, err)
	require.Contains(t, err.Error(), "resources.jobs.test_resource has lifecycle.prevent_destroy set")
	require.Contains(t, err.Error(), "but the plan calls for this resource to be recreated or destroyed")
	require.Contains(t, err.Error(), "disable lifecycle.prevent_destroy for resources.jobs.test_resource")
}

func TestCheckPreventDestroyForApp(t *testing.T) {
	b := &bundle.Bundle{}
	err := b.Config.Mutate(func(v dyn.Value) (dyn.Value, error) {
		return dyn.Set(v, "resources", dyn.NewValue(map[string]dyn.Value{
			"apps": dyn.NewValue(map[string]dyn.Value{
				"test_resource": dyn.NewValue(map[string]dyn.Value{
					"lifecycle": dyn.NewValue(map[string]dyn.Value{
						"prevent_destroy": dyn.NewValue(true, nil),
					}, nil),
				}, nil),
			}, nil),
		}, nil))
	})
	require.NoError(t, err)

	actions := []deployplan.Action{
		{
			ResourceKey: "resources.apps.test_resource",
			ActionType:  deployplan.Delete,
		},
	}

	err = checkForPreventDestroy(b, actions)
	require.Error(t, err)
	require.Contains(t, err.Error(), "resources.apps.test_resource has lifecycle.prevent_destroy set")
}

func TestCheckPreventDestroyNoError(t *testing.T) {
	b := &bundle.Bundle{}
	err := b.Config.Mutate(func(v dyn.Value) (dyn.Value, error) {
		return dyn.Set(v, "resources", dyn.NewValue(map[string]dyn.Value{
			"jobs": dyn.NewValue(map[string]dyn.Value{
				"test_resource": dyn.NewValue(map[string]dyn.Value{}, nil),
			}, nil),
		}, nil))
	})
	require.NoError(t, err)

	actions := []deployplan.Action{
		{
			ResourceKey: "resources.jobs.test_resource",
			ActionType:  deployplan.Recreate,
		},
	}

	err = checkForPreventDestroy(b, actions)
	require.NoError(t, err)
}

func TestCheckForPreventDestroyWhenFirstHasNoPreventDestroy(t *testing.T) {
	b := &bundle.Bundle{}
	err := b.Config.Mutate(func(v dyn.Value) (dyn.Value, error) {
		return dyn.Set(v, "resources", dyn.NewValue(map[string]dyn.Value{
			"jobs": dyn.NewValue(map[string]dyn.Value{
				"test_job": dyn.NewValue(map[string]dyn.Value{}, nil),
			}, nil),
			"apps": dyn.NewValue(map[string]dyn.Value{
				"test_app": dyn.NewValue(map[string]dyn.Value{
					"lifecycle": dyn.NewValue(map[string]dyn.Value{
						"prevent_destroy": dyn.NewValue(true, nil),
					}, nil),
				}, nil),
			}, nil),
		}, nil))
	})
	require.NoError(t, err)

	actions := []deployplan.Action{
		{
			ResourceKey: "resources.jobs.test_job",
			ActionType:  deployplan.Recreate,
		},
		{
			ResourceKey: "resources.apps.test_app",
			ActionType:  deployplan.Recreate,
		},
	}

	err = checkForPreventDestroy(b, actions)
	require.Error(t, err)
	require.Contains(t, err.Error(), "resources.apps.test_app has lifecycle.prevent_destroy set")
}
