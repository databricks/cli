package mutator

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/assert"
)

func TestSortJobClusters(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"foo": {
						JobSettings: &jobs.JobSettings{
							Tasks: []jobs.Task{
								{
									TaskKey:        "c",
									EnvironmentKey: "3",
								},
								{
									TaskKey:        "a",
									EnvironmentKey: "1",
								},
								{
									TaskKey:        "b",
									EnvironmentKey: "2",
								},
							},
						},
					},
					"bar": {
						JobSettings: &jobs.JobSettings{
							Tasks: []jobs.Task{
								{
									TaskKey: "d",
								},
								{
									TaskKey: "e",
								},
							},
						},
					},
				},
			},
		},
	}

	diags := bundle.Apply(context.Background(), b, SortJobTasks())
	assert.NoError(t, diags.Error())

	assert.Equal(t, []jobs.Task{
		{
			TaskKey:        "a",
			EnvironmentKey: "1",
		},
		{
			TaskKey:        "b",
			EnvironmentKey: "2",
		},
		{
			TaskKey:        "c",
			EnvironmentKey: "3",
		},
	}, b.Config.Resources.Jobs["foo"].Tasks)

	assert.Equal(t, []jobs.Task{
		{
			TaskKey: "d",
		},
		{
			TaskKey: "e",
		},
	}, b.Config.Resources.Jobs["bar"].Tasks)
}
