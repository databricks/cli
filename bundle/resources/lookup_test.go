package resources

import (
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLookup_EmptyBundle(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{},
		},
	}

	_, err := Lookup(b, "foo")
	require.Error(t, err)
	assert.ErrorContains(t, err, "resource with key \"foo\" not found")
}

func TestLookup_NotFound(t *testing.T) {
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
			},
		},
	}

	_, err := Lookup(b, "qux")
	require.Error(t, err)
	assert.ErrorContains(t, err, `resource with key "qux" not found`)
}

func TestLookup_MultipleFound(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"foo": {
						JobSettings: jobs.JobSettings{},
					},
				},
				Pipelines: map[string]*resources.Pipeline{
					"foo": {
						CreatePipeline: pipelines.CreatePipeline{},
					},
				},
			},
		},
	}

	_, err := Lookup(b, "foo")
	require.Error(t, err)
	assert.ErrorContains(t, err, `multiple resources with key "foo" found`)
}

func TestLookup_Nominal(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"foo": {
						JobSettings: jobs.JobSettings{
							Name: "Foo job",
						},
					},
				},
			},
		},
	}

	// Lookup by key only.
	out, err := Lookup(b, "foo")
	if assert.NoError(t, err) {
		assert.Equal(t, "Foo job", out.Resource.GetName())
	}

	// Lookup by type and key.
	out, err = Lookup(b, "jobs.foo")
	if assert.NoError(t, err) {
		assert.Equal(t, "Foo job", out.Resource.GetName())
	}
}

func TestLookup_NominalWithFilters(t *testing.T) {
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
						CreatePipeline: pipelines.CreatePipeline{},
					},
				},
			},
		},
	}

	includeJobs := func(ref Reference) bool {
		_, ok := ref.Resource.(*resources.Job)
		return ok
	}

	// This should succeed because the filter includes jobs.
	_, err := Lookup(b, "foo", includeJobs)
	require.NoError(t, err)

	// This should fail because the filter excludes pipelines.
	_, err = Lookup(b, "bar", includeJobs)
	require.Error(t, err)
	assert.ErrorContains(t, err, `resource with key "bar" not found`)
}
