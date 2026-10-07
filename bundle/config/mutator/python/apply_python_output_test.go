package python

import (
	"encoding/json"
	"testing"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/mutator/resourcemutator"

	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/structs/structvar"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type applyPythonOutputTestCase struct {
	name string

	input  any
	output any

	added   []resourcemutator.ResourceKey
	updated []resourcemutator.ResourceKey
	deleted []resourcemutator.ResourceKey
}

func TestApplyPythonOutput(t *testing.T) {
	job1 := mapOf("name", "job 1")
	job2 := mapOf("name", "job 2")

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
				mapOf("job_1", mapOf("name", "name")),
			)),
			output: mapOf("resources", mapOf("jobs",
				mapOf("job_1", mapOf2("name", "name", "description", "description")),
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
			input := loadRoot(t, tc.input)
			output := loadRoot(t, tc.output)
			plan, state, err := applyPythonOutput(input.View(), output.View())
			require.NoError(t, err)

			require.NoError(t, input.Override(plan))
			assert.Equal(t, output.View().AsAny(), input.View().AsAny())

			assert.ElementsMatch(t, tc.added, state.AddedResources.ToArray())
			assert.ElementsMatch(t, tc.updated, state.UpdatedResources.ToArray())
			assert.ElementsMatch(t, tc.deleted, state.DeletedResources.ToArray())
		})
	}
}

func TestMergeOutput_disallowDelete(t *testing.T) {
	input := loadRoot(t, mapOf("bundle", mapOf("name", "value")))
	output := loadRoot(t, emptyMap())

	_, _, err := applyPythonOutput(input.View(), output.View())

	assert.EqualError(t, err, `unexpected change at "bundle" (delete)`)
}

func TestMergeOutput_disallowInsert(t *testing.T) {
	output := loadRoot(t, mapOf("bundle", mapOf("name", "value")))
	input := loadRoot(t, emptyMap())

	_, _, err := applyPythonOutput(input.View(), output.View())

	assert.EqualError(t, err, `unexpected change at "bundle" (insert)`)
}

func TestMergeOutput_disallowUpdate(t *testing.T) {
	output := loadRoot(t, mapOf("bundle", mapOf("name", "value")))
	input := loadRoot(t, mapOf("bundle", mapOf("name", "new value")))

	_, _, err := applyPythonOutput(input.View(), output.View())

	assert.EqualError(t, err, `unexpected change at "bundle.name" (update)`)
}

// loadRoot returns the configuration for the configuration tree v.
func loadRoot(t *testing.T, v any) *config.Root {
	raw, err := json.Marshal(v)
	require.NoError(t, err)
	r, diags := config.LoadFromBytes("output.json", raw)
	require.NoError(t, diags.Error())
	return r
}

type overrideVisitorOmitemptyTestCase struct {
	name        string
	path        *structpath.PathNode
	left        any
	expectedErr error
}

func TestCreateOverrideVisitor_omitempty(t *testing.T) {
	// Python output can omit empty sequences/mappings in output, because we don't track them as optional,
	// there is no semantic difference between empty and missing, so we keep them as they were before
	// Python code deleted them.

	testCases := []overrideVisitorOmitemptyTestCase{
		{
			name:        "undo delete of empty variables",
			path:        structpath.MustParsePath("variables"),
			left:        &[]string{},
			expectedErr: structvar.ErrOverrideUndoDelete,
		},
		{
			name:        "undo delete of empty job clusters",
			path:        structpath.MustParsePath("resources.jobs.job0.job_clusters"),
			left:        &[]string{},
			expectedErr: structvar.ErrOverrideUndoDelete,
		},
		{
			name:        "allow delete of non-empty job clusters",
			path:        structpath.MustParsePath("resources.jobs.job0.job_clusters"),
			left:        &[]string{"abc"},
			expectedErr: nil,
		},
		{
			name:        "undo delete of empty tags",
			path:        structpath.MustParsePath("resources.jobs.job0.tags"),
			left:        &map[string]string{},
			expectedErr: structvar.ErrOverrideUndoDelete,
		},
		{
			name: "allow delete of non-empty tags",
			path: structpath.MustParsePath("resources.jobs.job0.tags"),
			left: &map[string]string{"dev": "true"},

			expectedErr: nil,
		},
		{
			name:        "undo delete of nil",
			path:        structpath.MustParsePath("resources.jobs.job0.tags"),
			left:        (*map[string]string)(nil),
			expectedErr: structvar.ErrOverrideUndoDelete,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, visitor := createOverrideVisitor(structvar.View{}, structvar.View{})

			err := visitor.VisitDelete(tc.path, structvar.NewView(tc.left, nil, nil))

			assert.Equal(t, tc.expectedErr, err)
		})
	}
}

func mapOf(key string, value any) any {
	return map[string]any{
		key: value,
	}
}

func mapOf2(key1 string, value1 any, key2 string, value2 any) any {
	return map[string]any{
		key1: value1,
		key2: value2,
	}
}

func emptyMap() any {
	return map[string]any{}
}
