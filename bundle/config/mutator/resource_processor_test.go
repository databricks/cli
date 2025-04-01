package mutator

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	assert "github.com/databricks/cli/libs/dyn/dynassert"
)

func TestResourceProcessor_process(t *testing.T) {
	ctx := context.Background()
	initializeMutator := recordingMutator{}
	normalizeMutator := recordingMutator{}

	rp := NewResourceProcessor(
		[]bundle.Mutator{&initializeMutator},
		[]bundle.Mutator{&normalizeMutator},
	)

	b := bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"job_1": {},
					"job_2": {},
					"job_3": {},
					"job_4": {},
					"job_5": {},
				},
			},
		},
	}

	opts := ResourceProcessorOpts{
		AddedResources:   NewResourceKeySet(),
		UpdatedResources: NewResourceKeySet(),
	}

	opts.AddedResources.AddResourceKey(ResourceKey{Type: "jobs", Name: "job_1"})
	opts.UpdatedResources.AddResourceKey(ResourceKey{Type: "jobs", Name: "job_2"})
	opts.UpdatedResources.AddResourceKey(ResourceKey{Type: "jobs", Name: "job_3"})

	diags := rp.Process(ctx, &b, opts)

	assert.NoError(t, diags.Error())
	assert.ElementsMatch(t, initializeMutator.jobNames, []string{"job_1"})
	assert.ElementsMatch(t, normalizeMutator.jobNames, []string{"job_1", "job_2", "job_3"})

	assert.Equal(t, 5, len(b.Config.Resources.Jobs))
}

type mergeResourcesTestCase struct {
	name     string
	src      dyn.Value
	dst      dyn.Value
	expected dyn.Value
}

func TestMergeResources(t *testing.T) {
	job1 := dyn.V("job_1")
	job2 := dyn.V("job_2")
	job3 := dyn.V("job_3")

	testCases := []mergeResourcesTestCase{
		{
			name:     "add resources to empty bundle (1)",
			src:      mapOf("resources", mapOf("jobs", mapOf("job_1", job1))),
			dst:      mapOf("resources", emptyMap()),
			expected: mapOf("resources", mapOf("jobs", mapOf("job_1", job1))),
		},
		{
			name:     "add resources to empty bundle (2)",
			src:      mapOf("resources", mapOf("jobs", mapOf("job_1", job1))),
			dst:      emptyMap(),
			expected: mapOf("resources", mapOf("jobs", mapOf("job_1", job1))),
		},
		{
			name:     "add new resource",
			src:      mapOf("resources", mapOf("jobs", mapOf("job_1", job1))),
			dst:      mapOf("resources", mapOf("jobs", mapOf("job_2", job2))),
			expected: mapOf("resources", mapOf("jobs", mapOf2("job_1", job1, "job_2", job2))),
		},
		{
			name:     "override resource",
			src:      mapOf("resources", mapOf("jobs", mapOf("job_1", job3))),
			dst:      mapOf("resources", mapOf("jobs", mapOf2("job_1", job1, "job_2", job2))),
			expected: mapOf("resources", mapOf("jobs", mapOf2("job_1", job3, "job_2", job2))),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual, err := mergeResources(tc.src, tc.dst)

			assert.NoError(t, err)
			assert.Equal(t, tc.expected, actual)
		})
	}
}

type selectResourcesTestCase struct {
	name          string
	config        dyn.Value
	resourcePaths []ResourceKey
	expected      dyn.Value
}

func TestSelectResources(t *testing.T) {
	job1 := dyn.V("job_1")
	job2 := dyn.V("job_2")

	testCases := []selectResourcesTestCase{
		{
			name:   "extract resources",
			config: mapOf("resources", mapOf("jobs", mapOf2("job_1", job1, "job_2", job2))),
			resourcePaths: []ResourceKey{
				{
					Type: "jobs",
					Name: "job_1",
				},
			},
			expected: mapOf("resources", mapOf("jobs", mapOf("job_1", job1))),
		},
		{
			name:          "extract no resources",
			config:        mapOf("resources", mapOf("jobs", mapOf2("job_1", job1, "job_2", job2))),
			resourcePaths: []ResourceKey{},
			expected:      mapOf("resources", emptyMap()),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resourceKeySet := NewResourceKeySet()
			for _, key := range tc.resourcePaths {
				resourceKeySet.AddResourceKey(key)
			}

			actual, err := selectResources(tc.config, resourceKeySet)

			assert.NoError(t, err)
			assert.Equal(t, tc.expected, actual)
		})
	}
}

func mapOf(key string, value dyn.Value) dyn.Value {
	return dyn.V(map[string]dyn.Value{key: value})
}

func mapOf2(k0 string, v0 dyn.Value, k1 string, v1 dyn.Value) dyn.Value {
	return dyn.V(map[string]dyn.Value{k0: v0, k1: v1})
}

func emptyMap() dyn.Value {
	return dyn.V(map[string]dyn.Value{})
}

// recordingMutator is a mutator that records the names of jobs it has seen.
type recordingMutator struct {
	jobNames []string
}

func (c *recordingMutator) Name() string {
	return "recording"
}

func (c *recordingMutator) Apply(_ context.Context, b *bundle.Bundle) diag.Diagnostics {
	for name := range b.Config.Resources.Jobs {
		c.jobNames = append(c.jobNames, name)
	}

	return diag.Diagnostics{}
}
