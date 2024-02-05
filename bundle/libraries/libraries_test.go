package libraries

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/assert"
)

func TestMapFilesToTaskLibrariesNoGlob(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Path: "testdata",
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"job1": {
						JobSettings: &jobs.JobSettings{
							Tasks: []jobs.Task{
								{
									Libraries: []compute.Library{
										{
											Whl: "library1",
										},
										{
											Whl: "library2",
										},
										{
											Whl: "/absolute/path/in/workspace/library3",
										},
									},
								},
								{
									Libraries: []compute.Library{
										{
											Whl: "library1",
										},
										{
											Whl: "library2",
										},
									},
								},
							},
						},
					},
					"job2": {
						JobSettings: &jobs.JobSettings{
							Tasks: []jobs.Task{
								{
									Libraries: []compute.Library{
										{
											Whl: "library1",
										},
										{
											Whl: "library2",
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	out := MapFilesToTaskLibraries(context.Background(), b)
	assert.Len(t, out, 2)

	// Pointer equality for "library1"
	assert.Equal(t, []*compute.Library{
		&b.Config.Resources.Jobs["job1"].JobSettings.Tasks[0].Libraries[0],
		&b.Config.Resources.Jobs["job1"].JobSettings.Tasks[1].Libraries[0],
		&b.Config.Resources.Jobs["job2"].JobSettings.Tasks[0].Libraries[0],
	}, out["testdata/library1"])

	// Pointer equality for "library2"
	assert.Equal(t, []*compute.Library{
		&b.Config.Resources.Jobs["job1"].JobSettings.Tasks[0].Libraries[1],
		&b.Config.Resources.Jobs["job1"].JobSettings.Tasks[1].Libraries[1],
		&b.Config.Resources.Jobs["job2"].JobSettings.Tasks[0].Libraries[1],
	}, out["testdata/library2"])
}
