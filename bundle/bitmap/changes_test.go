package bitmap

import (
	"slices"
	"testing"

	"github.com/databricks/cli/libs/dyn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuilderMarkChanges(t *testing.T) {
	schema := EmbeddedSchema()
	job := []string{"resources", "resources.jobs", "resources.jobs.*"}
	for _, tc := range []struct {
		name          string
		before, after string
		want          []string
	}{
		{
			name:   "nothing changed",
			before: `resources: {jobs: {a: {name: x, max_concurrent_runs: 1}}}`,
			after:  `resources: {jobs: {a: {name: x, max_concurrent_runs: 1}}}`,
			want:   nil,
		},
		{
			name:   "a new resource is marked whole",
			before: `resources: {jobs: {a: {name: x}}}`,
			after:  `resources: {jobs: {a: {name: x}, b: {name: y, max_concurrent_runs: 1}}}`,
			want:   append(slices.Clone(job), "resources.jobs.*.name", "resources.jobs.*.max_concurrent_runs"),
		},
		{
			name:   "only the field added to an existing resource",
			before: `resources: {jobs: {a: {name: x, max_concurrent_runs: 1}}}`,
			after:  `resources: {jobs: {a: {name: x, max_concurrent_runs: 1, description: d}}}`,
			want:   append(slices.Clone(job), "resources.jobs.*.description"),
		},
		{
			name:   "a changed bool records its new value",
			before: `resources: {jobs: {a: {queue: {enabled: true}}}}`,
			after:  `resources: {jobs: {a: {queue: {enabled: false}}}}`,
			want:   append(slices.Clone(job), "resources.jobs.*.queue", "resources.jobs.*.queue.enabled", "resources.jobs.*.queue.enabled=false"),
		},
		{
			name:   "an entry added to a map",
			before: `resources: {jobs: {a: {tags: {team: x}}}}`,
			after:  `resources: {jobs: {a: {tags: {team: x, cost: y}}}}`,
			want:   append(slices.Clone(job), "resources.jobs.*.tags", "resources.jobs.*.tags.*"),
		},
		{
			name:   "list elements are paired by key",
			before: `resources: {jobs: {a: {tasks: [{task_key: t1}]}}}`,
			after:  `resources: {jobs: {a: {tasks: [{task_key: t1, description: d}, {task_key: t2}]}}}`,
			want: append(slices.Clone(job), "resources.jobs.*.tasks", "resources.jobs.*.tasks[*]",
				"resources.jobs.*.tasks[*].description", "resources.jobs.*.tasks[*].task_key"),
		},
		// The CLI filled in fields of t1 before Python ran; Python only added t0.
		{
			name:   "an element prepended to a keyed list",
			before: `resources: {jobs: {a: {tasks: [{task_key: t1, notebook_task: {notebook_path: p, source: WORKSPACE}}]}}}`,
			after:  `resources: {jobs: {a: {tasks: [{task_key: t0}, {task_key: t1, notebook_task: {notebook_path: p, source: WORKSPACE}}]}}}`,
			want:   append(slices.Clone(job), "resources.jobs.*.tasks", "resources.jobs.*.tasks[*]", "resources.jobs.*.tasks[*].task_key"),
		},
		{
			name:   "a keyed element that moved and changed",
			before: `resources: {jobs: {a: {tasks: [{task_key: t1}, {task_key: t2, max_retries: 1}]}}}`,
			after:  `resources: {jobs: {a: {tasks: [{task_key: t2, max_retries: 2}, {task_key: t1}]}}}`,
			want:   append(slices.Clone(job), "resources.jobs.*.tasks", "resources.jobs.*.tasks[*]", "resources.jobs.*.tasks[*].max_retries"),
		},
		{
			name:   "a list without keys that Python reorders",
			before: `resources: {jobs: {a: {permissions: [{level: CAN_VIEW, user_name: u}, {level: IS_OWNER, user_name: o}]}}}`,
			after:  `resources: {jobs: {a: {permissions: [{level: IS_OWNER, user_name: o}, {level: CAN_VIEW, user_name: u}]}}}`,
			want:   nil,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			b := NewBuilder(schema)
			b.MarkChanges(loadValue(t, tc.before), loadValue(t, tc.after))
			bits, err := b.Bits(struct{}{})
			require.NoError(t, err)
			want := slices.Clone(tc.want)
			slices.Sort(want)
			assert.Equal(t, want, setPaths(schema, bits))
		})
	}

	// A field written out as null is not set.
	before := loadValue(t, `resources: {jobs: {a: {name: x}}}`)
	after, err := dyn.SetByPath(before, dyn.MustPathFromString("resources.jobs.a.description"), dyn.NilValue)
	require.NoError(t, err)
	b := NewBuilder(schema)
	b.MarkChanges(before, after)
	bits, err := b.Bits(struct{}{})
	require.NoError(t, err)
	assert.Empty(t, setPaths(schema, bits))
}
