package python

import (
	"math"
	"testing"

	"github.com/databricks/cli/bundle/config/mutator/resourcemutator"

	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn/merge"

	"github.com/databricks/cli/libs/dyn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestMergeOutput_wholeFloatAsInt(t *testing.T) {
	input := mapOf("not_resource", dyn.V(1.0))
	output := mapOf("not_resource", dyn.V(1))

	merged, _, err := applyPythonOutput(input, output)

	require.NoError(t, err)
	assert.Equal(t, input.Get("not_resource"), merged.Get("not_resource"))
}

func TestMergeOutput_wholeFloatAsInt_negativeZero(t *testing.T) {
	input := mapOf("not_resource", dyn.V(math.Copysign(0, -1)))
	output := mapOf("not_resource", dyn.V(0))

	_, _, err := applyPythonOutput(input, output)

	require.NoError(t, err)
}

func TestMergeOutput_wholeFloatAsInt_shortestDecimalAbove2p53(t *testing.T) {
	// json.Marshal writes 1234567890123456789.0 (stored as 1234567890123456768) as 1234567890123456800.
	input := mapOf("not_resource", dyn.V(1234567890123456789.0))
	output := mapOf("not_resource", dyn.V(int64(1234567890123456800)))

	merged, _, err := applyPythonOutput(input, output)

	require.NoError(t, err)
	assert.Equal(t, input.Get("not_resource"), merged.Get("not_resource"))
}

func TestMergeOutput_disallowNearbyIntAbove2p53(t *testing.T) {
	// float64(9007199254740993) rounds to 9007199254740992, but Python did change the value.
	input := mapOf("not_resource", dyn.V(9007199254740992.0))
	output := mapOf("not_resource", dyn.V(int64(9007199254740993)))

	_, _, err := applyPythonOutput(input, output)

	assert.EqualError(t, err, `unexpected change at "not_resource" (update)`)
}

func TestMergeOutput_disallowFloatToDifferentInt(t *testing.T) {
	input := mapOf("not_resource", dyn.V(1.5))
	output := mapOf("not_resource", dyn.V(1))

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

	location := diag.Location{
		File:   "databricks.yml",
		Line:   10,
		Column: 20,
	}

	testCases := []overrideVisitorOmitemptyTestCase{
		{
			name:        "undo delete of empty variables",
			path:        dyn.MustPathFromString("variables"),
			left:        dyn.NewValue([]dyn.Value{}, []diag.Location{location}),
			expectedErr: merge.ErrOverrideUndoDelete,
		},
		{
			name:        "undo delete of empty job clusters",
			path:        dyn.MustPathFromString("resources.jobs.job0.job_clusters"),
			left:        dyn.NewValue([]dyn.Value{}, []diag.Location{location}),
			expectedErr: merge.ErrOverrideUndoDelete,
		},
		{
			name:        "allow delete of non-empty job clusters",
			path:        dyn.MustPathFromString("resources.jobs.job0.job_clusters"),
			left:        dyn.NewValue([]dyn.Value{dyn.NewValue("abc", []diag.Location{location})}, []diag.Location{location}),
			expectedErr: nil,
		},
		{
			name:        "undo delete of empty tags",
			path:        dyn.MustPathFromString("resources.jobs.job0.tags"),
			left:        dyn.NewValue(map[string]dyn.Value{}, []diag.Location{location}),
			expectedErr: merge.ErrOverrideUndoDelete,
		},
		{
			name: "allow delete of non-empty tags",
			path: dyn.MustPathFromString("resources.jobs.job0.tags"),
			left: dyn.NewValue(map[string]dyn.Value{"dev": dyn.NewValue("true", []diag.Location{location})}, []diag.Location{location}),

			expectedErr: nil,
		},
		{
			name:        "undo delete of nil",
			path:        dyn.MustPathFromString("resources.jobs.job0.tags"),
			left:        dyn.NilValue.WithLocations([]diag.Location{location}),
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
