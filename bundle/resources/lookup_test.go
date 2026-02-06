package resources

import (
	"sort"
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

func TestLookupByPrefix_NoMatches(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"foo": {},
					"bar": {},
				},
			},
		},
	}

	matches := LookupByPrefix(b, "qux")
	assert.Empty(t, matches)
}

func TestLookupByPrefix_SingleMatch(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"foo_job": {
						JobSettings: jobs.JobSettings{Name: "Foo job"},
					},
					"bar_job": {},
				},
			},
		},
	}

	matches := LookupByPrefix(b, "foo")
	require.Len(t, matches, 1)
	assert.Equal(t, "foo_job", matches[0].Key)
	assert.Equal(t, "Foo job", matches[0].Resource.GetName())
}

func TestLookupByPrefix_MultipleMatches(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"my_job_1": {},
					"my_job_2": {},
					"other":    {},
				},
			},
		},
	}

	matches := LookupByPrefix(b, "my_")
	require.Len(t, matches, 2)

	keys := []string{matches[0].Key, matches[1].Key}
	sort.Strings(keys)
	assert.Equal(t, []string{"my_job_1", "my_job_2"}, keys)
}

func TestLookupByPrefix_WithFilters(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"my_job": {},
				},
				Pipelines: map[string]*resources.Pipeline{
					"my_pipeline": {},
				},
			},
		},
	}

	includeJobs := func(ref Reference) bool {
		_, ok := ref.Resource.(*resources.Job)
		return ok
	}

	matches := LookupByPrefix(b, "my_", includeJobs)
	require.Len(t, matches, 1)
	assert.Equal(t, "my_job", matches[0].Key)
}

func TestLookupByPrefix_ExactPrefixMatchesAll(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"foo":     {},
					"foobar":  {},
					"foobaz":  {},
					"another": {},
				},
			},
		},
	}

	matches := LookupByPrefix(b, "foo")
	require.Len(t, matches, 3)

	keys := make([]string, len(matches))
	for i, m := range matches {
		keys[i] = m.Key
	}
	sort.Strings(keys)
	assert.Equal(t, []string{"foo", "foobar", "foobaz"}, keys)
}
