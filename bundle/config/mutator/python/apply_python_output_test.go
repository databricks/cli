package python

import (
	"testing"

	"github.com/databricks/cli/bundle/config/mutator/resourcemutator"

	"github.com/databricks/cli/libs/dyn/merge"

	"github.com/databricks/cli/libs/dyn"
	assert "github.com/databricks/cli/libs/dyn/dynassert"
)

type applyPythonOutputTestCase struct {
	name string

	input  dyn.Value
	output dyn.Value

	added   []resourcemutator.ResourceKey
	updated []resourcemutator.ResourceKey
	deleted []resourcemutator.ResourceKey
}

func TestApplyPythonOutput(t *testing.T) {
	job1 := mapOf("name", dyn.V("job 1"))
	job2 := mapOf("name", dyn.V("job 2"))

	testCases := []applyPythonOutputTestCase{
		{
			name:   "add job (0)",
			input:  emptyMap(),
			output: mapOf("resources", mapOf("jobs", mapOf("job_1", job1))),
			added: []resourcemutator.ResourceKey{
				{
					Type: "jobs",
					Name: "job_1",
				},
			},
		},
		{
			name:   "add job (1)",
			input:  mapOf("resources", emptyMap()),
			output: mapOf("resources", mapOf("jobs", mapOf("job_1", job1))),
			added: []resourcemutator.ResourceKey{
				{
					Type: "jobs",
					Name: "job_1",
				},
			},
		},
		{
			name:   "add job (2)",
			input:  mapOf("resources", mapOf("jobs", emptyMap())),
			output: mapOf("resources", mapOf("jobs", mapOf("job_1", job1))),
			added: []resourcemutator.ResourceKey{
				{
					Type: "jobs",
					Name: "job_1",
				},
			},
		},
		{
			name:   "add job (3)",
			input:  mapOf("resources", mapOf("jobs", mapOf("job_1", job1))),
			output: mapOf("resources", mapOf("jobs", mapOf2("job_1", job1, "job_2", job2))),
			added: []resourcemutator.ResourceKey{
				{
					Type: "jobs",
					Name: "job_2",
				},
			},
		},
		{
			name:   "delete job (0)",
			input:  mapOf("resources", mapOf("jobs", mapOf("job_1", job1))),
			output: emptyMap(),
			deleted: []resourcemutator.ResourceKey{
				{
					Type: "jobs",
					Name: "job_1",
				},
			},
		},
		{
			name:   "delete job (1)",
			input:  mapOf("resources", mapOf("jobs", mapOf("job_1", job1))),
			output: mapOf("resources", emptyMap()),
			deleted: []resourcemutator.ResourceKey{
				{
					Type: "jobs",
					Name: "job_1",
				},
			},
		},
		{
			name:   "delete job (2)",
			input:  mapOf("resources", mapOf("jobs", mapOf("job_1", job1))),
			output: mapOf("resources", mapOf("jobs", emptyMap())),
			deleted: []resourcemutator.ResourceKey{
				{
					Type: "jobs",
					Name: "job_1",
				},
			},
		},
		{
			name:   "update job",
			input:  mapOf("resources", mapOf("jobs", mapOf("job_1", job1))),
			output: mapOf("resources", mapOf("jobs", mapOf("job_1", job2))),
			updated: []resourcemutator.ResourceKey{
				{
					Type: "jobs",
					Name: "job_1",
				},
			},
		},
		{
			name:   "update job through 'name' delete",
			input:  mapOf("resources", mapOf("jobs", mapOf("job_1", job1))),
			output: mapOf("resources", mapOf("jobs", mapOf("job_1", emptyMap()))),
			updated: []resourcemutator.ResourceKey{
				{
					Type: "jobs",
					Name: "job_1",
				},
			},
		},
		{
			name: "update job through 'description' insert",
			input: mapOf("resources", mapOf("jobs",
				mapOf("job_1", mapOf("name", dyn.V("name"))),
			)),
			output: mapOf("resources", mapOf("jobs",
				mapOf("job_1", mapOf2("name", dyn.V("name"), "description", dyn.V("description"))),
			)),
			updated: []resourcemutator.ResourceKey{
				{
					Type: "jobs",
					Name: "job_1",
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			merged, state, err := applyPythonOutput(tc.input, tc.output)

			assert.NoError(t, err)
			assert.Equal(t, tc.output, merged)

			assert.ElementsMatch(t, tc.added, state.AddedResources.ToArray())
			assert.ElementsMatch(t, tc.updated, state.UpdatedResources.ToArray())
			assert.ElementsMatch(t, tc.deleted, state.DeletedResources.ToArray())
		})
	}
}

func TestMergeOutput_disallowDelete(t *testing.T) {
	input := mapOf("not_resource", dyn.V("value"))
	output := emptyMap()

	_, _, err := applyPythonOutput(input, output)

	assert.EqualError(t, err, `unexpected change at "not_resource" (delete)`)
}

func TestMergeOutput_disallowInsert(t *testing.T) {
	output := mapOf("not_resource", dyn.V("value"))
	input := emptyMap()

	_, _, err := applyPythonOutput(input, output)

	assert.EqualError(t, err, `unexpected change at "not_resource" (insert)`)
}

func TestMergeOutput_disallowUpdate(t *testing.T) {
	output := mapOf("not_resource", dyn.V("value"))
	input := mapOf("not_resource", dyn.V("new value"))

	_, _, err := applyPythonOutput(input, output)

	assert.EqualError(t, err, `unexpected change at "not_resource" (update)`)
}

type overrideVisitorOmitemptyTestCase struct {
	name        string
	path        dyn.Path
	left        dyn.Value
	expectedErr error
}

func TestCreateOverrideVisitor_omitempty(t *testing.T) {
	// Python output can omit empty sequences/mappings in output, because we don't track them as optional,
	// there is no semantic difference between empty and missing, so we keep them as they were before
	// Python code deleted them.

	location := dyn.Location{
		File:   "databricks.yml",
		Line:   10,
		Column: 20,
	}

	testCases := []overrideVisitorOmitemptyTestCase{
		{
			name:        "undo delete of empty variables",
			path:        dyn.MustPathFromString("variables"),
			left:        dyn.NewValue([]dyn.Value{}, []dyn.Location{location}),
			expectedErr: merge.ErrOverrideUndoDelete,
		},
		{
			name:        "undo delete of empty job clusters",
			path:        dyn.MustPathFromString("resources.jobs.job0.job_clusters"),
			left:        dyn.NewValue([]dyn.Value{}, []dyn.Location{location}),
			expectedErr: merge.ErrOverrideUndoDelete,
		},
		{
			name:        "allow delete of non-empty job clusters",
			path:        dyn.MustPathFromString("resources.jobs.job0.job_clusters"),
			left:        dyn.NewValue([]dyn.Value{dyn.NewValue("abc", []dyn.Location{location})}, []dyn.Location{location}),
			expectedErr: nil,
		},
		{
			name:        "undo delete of empty tags",
			path:        dyn.MustPathFromString("resources.jobs.job0.tags"),
			left:        dyn.NewValue(map[string]dyn.Value{}, []dyn.Location{location}),
			expectedErr: merge.ErrOverrideUndoDelete,
		},
		{
			name: "allow delete of non-empty tags",
			path: dyn.MustPathFromString("resources.jobs.job0.tags"),
			left: dyn.NewValue(map[string]dyn.Value{"dev": dyn.NewValue("true", []dyn.Location{location})}, []dyn.Location{location}),

			expectedErr: nil,
		},
		{
			name:        "undo delete of nil",
			path:        dyn.MustPathFromString("resources.jobs.job0.tags"),
			left:        dyn.NilValue.WithLocations([]dyn.Location{location}),
			expectedErr: merge.ErrOverrideUndoDelete,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, visitor := createOverrideVisitor(dyn.NilValue, dyn.NilValue)

			err := visitor.VisitDelete(tc.path, tc.left)

			assert.Equal(t, tc.expectedErr, err)
		})
	}
}

func mapOf(key string, value dyn.Value) dyn.Value {
	return dyn.V(map[string]dyn.Value{
		key: value,
	})
}

func mapOf2(key1 string, value1 dyn.Value, key2 string, value2 dyn.Value) dyn.Value {
	return dyn.V(map[string]dyn.Value{
		key1: value1,
		key2: value2,
	})
}

func emptyMap() dyn.Value {
	return dyn.V(map[string]dyn.Value{})
}
