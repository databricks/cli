package resources

import (
	"testing"

	"github.com/databricks/cli/libs/sdktypes/jobs/v2"
	"github.com/databricks/cli/libs/sdktypes/pipelines/v2"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/stretchr/testify/assert"
)

func TestCompletions_SkipDuplicates(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"foo": {
						JobSettings: jobs.JobSettings{},
					},
					"bar": {
						JobSettings: jobs.JobSettings{},
					},
				},
				Pipelines: map[string]*resources.Pipeline{
					"foo": {
						CreatePipelineRequest: pipelines.CreatePipelineRequest{},
					},
				},
			},
		},
	}

	// Test that this skips duplicates and only includes unambiguous completions.
	out := Completions(b)
	if assert.Len(t, out, 1) {
		assert.Contains(t, out, "bar")
	}
}

func TestCompletions_Filter(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"foo": {
						JobSettings: jobs.JobSettings{},
					},
				},
				Pipelines: map[string]*resources.Pipeline{
					"bar": {
						CreatePipelineRequest: pipelines.CreatePipelineRequest{},
					},
				},
			},
		},
	}

	includeJobs := func(ref Reference) bool {
		_, ok := ref.Resource.(*resources.Job)
		return ok
	}

	// Test that this does not include the pipeline.
	out := Completions(b, includeJobs)
	if assert.Len(t, out, 1) {
		assert.Contains(t, out, "foo")
	}
}
