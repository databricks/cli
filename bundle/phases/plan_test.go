package phases

import (
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/databricks-sdk-go/service/apps"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/require"
)

func TestCheckPreventDestroyForJob(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"test_resource": {
						BaseResource: resources.BaseResource{
							Lifecycle: resources.Lifecycle{PreventDestroy: true},
						},
						JobSettings: jobs.JobSettings{},
					},
				},
			},
		},
	}

	actions := []deployplan.Action{
		{
			ResourceKey: "resources.jobs.test_resource",
			ActionType:  deployplan.Recreate,
		},
	}

	err := checkForPreventDestroy(b, actions)
	require.Error(t, err)
	require.Contains(t, err.Error(), "resources.jobs.test_resource has lifecycle.prevent_destroy set")
	require.Contains(t, err.Error(), "but the plan calls for this resource to be recreated or destroyed")
	require.Contains(t, err.Error(), "disable lifecycle.prevent_destroy for resources.jobs.test_resource")
}

func TestCheckPreventDestroyForApp(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Apps: map[string]*resources.App{
					"test_resource": {
						Lifecycle: resources.LifecycleWithStarted{PreventDestroy: true},
					},
				},
			},
		},
	}

	actions := []deployplan.Action{
		{
			ResourceKey: "resources.apps.test_resource",
			ActionType:  deployplan.Delete,
		},
	}

	err := checkForPreventDestroy(b, actions)
	require.Error(t, err)
	require.Contains(t, err.Error(), "resources.apps.test_resource has lifecycle.prevent_destroy set")
}

func TestCheckPreventDestroyNoError(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"test_resource": {
						JobSettings: jobs.JobSettings{},
					},
				},
			},
		},
	}

	actions := []deployplan.Action{
		{
			ResourceKey: "resources.jobs.test_resource",
			ActionType:  deployplan.Recreate,
		},
	}

	err := checkForPreventDestroy(b, actions)
	require.NoError(t, err)
}

func TestCheckForPreventDestroyWhenFirstHasNoPreventDestroy(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Bundle: config.Bundle{
				Name: "test",
			},
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"test_job": {
						JobSettings: jobs.JobSettings{},
					},
				},
				Apps: map[string]*resources.App{
					"test_app": {
						App: apps.App{
							Name: "Test App",
						},
						Lifecycle: resources.LifecycleWithStarted{PreventDestroy: true},
					},
				},
			},
		},
	}

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

	err := checkForPreventDestroy(b, actions)
	require.Error(t, err)
	require.Contains(t, err.Error(), "resources.apps.test_app has lifecycle.prevent_destroy set")
}
