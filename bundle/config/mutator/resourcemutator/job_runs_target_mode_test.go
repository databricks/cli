package resourcemutator

import (
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/service/jobs"
)

func init() {
	registerTargetModeExamples("job_runs", func(r *config.Resources) {
		r.JobRuns = map[string]*resources.JobRun{
			"job_run1": {RunNow: jobs.RunNow{JobId: 1234}},
		}
	})
}
