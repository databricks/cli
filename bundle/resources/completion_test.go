package resources

import (
	"testing"

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
					"foo": {},
					"bar": {},
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

func TestCompletions_Filter(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"foo": {},
				},
				Pipelines: map[string]*resources.Pipeline{
					"bar": {},
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
