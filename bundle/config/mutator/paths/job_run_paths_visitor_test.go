package paths

import (
	"testing"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/dyn"
	"github.com/stretchr/testify/assert"
)

func TestVisitJobRunPaths(t *testing.T) {
	watched := "watched.txt"
	root := config.Root{
		Resources: config.Resources{
			JobRuns: map[string]*resources.JobRun{
				"run0": {
					Lifecycle: &resources.JobRunLifecycle{
						Triggers: []resources.JobRunTrigger{
							{OnFileChange: &watched},
						},
					},
				},
			},
		},
	}

	actual := collectVisitedPaths(t, root, VisitJobRunPaths)
	expected := []dyn.Path{
		dyn.MustPathFromString("resources.job_runs.run0.lifecycle.triggers[0].on_file_change"),
	}

	assert.ElementsMatch(t, expected, actual)
}
