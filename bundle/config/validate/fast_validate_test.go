package validate

import (
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/assert"
)

func TestFastValidateRunsRegisteredValidators(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"job1": {
						JobSettings: jobs.JobSettings{
							Tasks: []jobs.Task{
								{
									TaskKey:         "task1",
									JobClusterKey:   "undefined",
									SparkPythonTask: &jobs.SparkPythonTask{PythonFile: "main.py"},
								},
								{
									TaskKey:         "task2",
									SparkPythonTask: &jobs.SparkPythonTask{PythonFile: "main.py"},
								},
							},
						},
					},
				},
			},
		},
	}

	diags := bundle.Apply(t.Context(), b, FastValidate())

	var summaries []string
	for _, d := range diags {
		summaries = append(summaries, d.Summary)
	}
	// The validators run in parallel, so their diagnostics come in any order.
	assert.ElementsMatch(t, []string{
		"job_cluster_key undefined is not defined",
		"Missing required cluster or environment settings",
	}, summaries)
}
