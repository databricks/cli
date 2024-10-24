package resources

import (
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/assert"
)

func TestCompletions_SkipDuplicates(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"foo": {},
					"bar": {
						JobSettings: &jobs.JobSettings{
							Name: "Bar job",
						},
					},
				},
				Pipelines: map[string]*resources.Pipeline{
					"foo": {},
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
