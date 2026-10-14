package validate

import (
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/sdktypes/jobs/v2"
	"github.com/stretchr/testify/assert"
)

func TestValidateJobRunIdempotencyTokenAcceptsRunsWithoutOne(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				JobRuns: map[string]*resources.JobRun{
					"my_run": {RunNowRequest: jobs.RunNowRequest{JobId: new(int64(456))}},
					// An empty `job_runs.<name>:` entry loads as a present key with a nil value.
					"empty": nil,
				},
			},
		},
	}

	assert.Empty(t, ValidateJobRunIdempotencyToken().Apply(t.Context(), b))
}
