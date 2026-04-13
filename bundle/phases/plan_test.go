package phases

import (
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/libs/dyn"
	"github.com/stretchr/testify/require"
)

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
