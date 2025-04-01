package python

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/databricks/cli/libs/dyn"
	assert "github.com/databricks/cli/libs/dyn/dynassert"
	"github.com/stretchr/testify/require"
)

type mergeOutputTestCase struct {
	name string

	root   dyn.Value
	output dyn.Value

	added   []mutator.ResourceKey
	updated []mutator.ResourceKey

	phase phase
}

func TestMergeOutput(t *testing.T) {
	testCases := []mergeOutputTestCase{
		{
			name:   "load_resources - empty bundle (0)",
			root:   emptyMap(),
			output: mapOf("resources", mapOf("jobs", mapOf("job_1", emptyMap()))),
			phase:  PythonMutatorPhaseLoadResources,
			added: []mutator.ResourceKey{
				{
					Type: "jobs",
					Name: "job_1",
				},
			},
		},
		{
			name:   "load_resources - empty bundle (1)",
			root:   mapOf("resources", emptyMap()),
			output: mapOf("resources", mapOf("jobs", mapOf("job_1", emptyMap()))),
			phase:  PythonMutatorPhaseLoadResources,
			added: []mutator.ResourceKey{
				{
					Type: "jobs",
					Name: "job_1",
				},
			},
		},
		{
			name:   "load_resources - empty bundle (2)",
			root:   mapOf("resources", mapOf("jobs", emptyMap())),
			output: mapOf("resources", mapOf("jobs", mapOf("job_1", emptyMap()))),
			phase:  PythonMutatorPhaseLoadResources,
			added: []mutator.ResourceKey{
				{
					Type: "jobs",
					Name: "job_1",
				},
			},
		},
		{
			name:   "load_resources - new job",
			root:   mapOf("resources", mapOf("jobs", mapOf("job_1", emptyMap()))),
			output: mapOf("resources", mapOf("jobs", mapOf2("job_1", emptyMap(), "job_2", emptyMap()))),
			phase:  PythonMutatorPhaseLoadResources,
			added: []mutator.ResourceKey{
				{
					Type: "jobs",
					Name: "job_2",
				},
			},
		},
		{
			name: "apply_mutators - update job",
			root: mapOf("resources", mapOf("jobs",
				mapOf("job_1", mapOf("name", dyn.V("name"))),
			)),
			output: mapOf("resources", mapOf("jobs",
				mapOf("job_1", mapOf("name", dyn.V("new name"))),
			)),
			phase: PythonMutatorPhaseApplyMutators,
			updated: []mutator.ResourceKey{
				{
					Type: "jobs",
					Name: "job_1",
				},
			},
		},
		{
			name: "apply_mutators - update job through delete",
			root: mapOf("resources", mapOf("jobs",
				mapOf("job_1", mapOf("name", dyn.V("name"))),
			)),
			output: mapOf("resources", mapOf("jobs",
				mapOf("job_1", emptyMap()),
			)),
			phase: PythonMutatorPhaseApplyMutators,
			updated: []mutator.ResourceKey{
				{
					Type: "jobs",
					Name: "job_1",
				},
			},
		},
		{
			name: "apply_mutators - update job through insert",
			root: mapOf("resources", mapOf("jobs",
				mapOf("job_1", emptyMap()),
			)),
			output: mapOf("resources", mapOf("jobs",
				mapOf("job_1", mapOf("name", dyn.V("name"))),
			)),
			phase: PythonMutatorPhaseApplyMutators,
			updated: []mutator.ResourceKey{
				{
					Type: "jobs",
					Name: "job_1",
				},
			},
		},
	}

	ctx := context.Background()

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			merged, state, err := mergeOutput(ctx, tc.phase, tc.root, tc.output)

			require.NoError(t, err)

			// merged is always the same as output because we don't set locations here
			require.Equal(t, tc.output, merged)

			assert.ElementsMatch(t, tc.added, state.AddedResources.ToArray())
			assert.ElementsMatch(t, tc.updated, state.UpdatedResources.ToArray())
		})
	}
}

func mapOf(key string, value dyn.Value) dyn.Value {
	return dyn.NewValue(map[string]dyn.Value{
		key: value,
	}, []dyn.Location{})
}

func mapOf2(key1 string, value1 dyn.Value, key2 string, value2 dyn.Value) dyn.Value {
	return dyn.V(map[string]dyn.Value{
		key1: value1,
		key2: value2,
	})
}

func emptyMap() dyn.Value {
	return dyn.NewValue(map[string]dyn.Value{}, []dyn.Location{})
}
