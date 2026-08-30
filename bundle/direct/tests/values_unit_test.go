package tests

import (
	"testing"

	"github.com/databricks/cli/libs/structs/structpath"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A field's values form a complete digraph, so one Eulerian circuit covers every ordered
// pair exactly once and no second chain is ever needed. This pins that down: full
// coverage, no repeats, and every step starting where the last one ended.
func TestTransitionsCoverEveryPairInOneChain(t *testing.T) {
	for n := 2; n <= 6; n++ {
		values := make([]any, 0, n-1)
		for i := 1; i < n; i++ {
			values = append(values, i)
		}
		f := field{path: "some.field", values: values} //exhaustruct:ignore

		got := f.transitions()

		// n-1 explicit values plus the implicit absent.
		want := n * (n - 1)
		require.Len(t, got, want, "n=%d", n)

		seen := map[string]bool{}
		for i, tr := range got {
			pair := tr.label()
			assert.False(t, seen[pair], "pair %s repeated at %d (n=%d)", pair, i, n)
			seen[pair] = true
			assert.NotEqual(t, valueLabel(tr.from), valueLabel(tr.to), "self-loop at %d", i)
			if i > 0 {
				assert.Equal(t, valueLabel(got[i-1].to), valueLabel(tr.from),
					"step %d does not start where step %d ended (n=%d)", i, i-1, n)
			}
		}
		assert.Len(t, seen, want, "n=%d", n)
	}
}

// A required field has no absent value, so its chain is one vertex smaller.
func TestTransitionsSkipAbsentForRequiredField(t *testing.T) {
	f := field{path: "name", values: []any{"a", "b"}, required: true} //exhaustruct:ignore

	got := f.transitions()

	require.Len(t, got, 2)
	for _, tr := range got {
		assert.NotEqual(t, "absent", valueLabel(tr.from))
		assert.NotEqual(t, "absent", valueLabel(tr.to))
	}
}

// A pattern from the type walk has to expand against the deployed resource, or the fields
// inside a container are silently never tested. Map wildcards ("tags.*") and element
// wildcards ("tasks[*]") take different paths through splitPattern, so both are pinned here.
func TestExpandPattern(t *testing.T) {
	//exhaustruct:ignore
	job := &resources.Job{JobSettings: jobs.JobSettings{
		Name: "n",
		Tags: map[string]string{"team": "eng"},
		Tasks: []jobs.Task{
			{TaskKey: "a"}, //exhaustruct:ignore
			{TaskKey: "b"}, //exhaustruct:ignore
		},
		EmailNotifications: &jobs.JobEmailNotifications{OnFailure: []string{"a@b.test"}}, //exhaustruct:ignore
	}}

	for _, tc := range []struct {
		pattern string
		want    []string
	}{
		{"name", []string{"name"}},
		{"tags.*", []string{"tags['team']"}},
		{"tasks[*].task_key", []string{"tasks[0].task_key", "tasks[1].task_key"}},
		{"email_notifications.on_failure[*]", []string{"email_notifications.on_failure[0]"}},
		// Nothing behind it in the resource, which is what makes a field "not covered".
		{"job_clusters[*].job_cluster_key", nil},
		{"tasks[0].notebook_task.base_parameters.*", nil},
	} {
		t.Run(tc.pattern, func(t *testing.T) {
			assert.Equal(t, tc.want, expandPattern(job, tc.pattern))
		})
	}
}

// setValue applies one edit the way setField does, without a bundle around it.
func setValue(resource any, path string, value any) error {
	node, err := structpath.ParsePath(path)
	if err != nil {
		return err
	}
	return setNode(resource, node, value)
}

// setField writes into the typed resource, where "absent" is the zero value with the field
// dropped from ForceSendFields -- the distinction the API sees. A map entry and a slice
// element have to be removed outright, which is a different code path.
func TestSetFieldAbsentAndEmpty(t *testing.T) {
	//exhaustruct:ignore
	job := &resources.Job{JobSettings: jobs.JobSettings{
		Name: "n",
		Tags: map[string]string{"team": "eng", "env": "dev"},
		Tasks: []jobs.Task{
			{TaskKey: "a"}, //exhaustruct:ignore
			{TaskKey: "b"}, //exhaustruct:ignore
		},
	}}

	require.NoError(t, setValue(job, "description", ""))
	assert.Contains(t, job.ForceSendFields, "Description")

	require.NoError(t, setValue(job, "description", nil))
	assert.NotContains(t, job.ForceSendFields, "Description")

	// A nested object the config leaves out is allocated on the way in.
	require.NoError(t, setValue(job, "email_notifications.no_alert_for_skipped_runs", true))
	require.NotNil(t, job.EmailNotifications)
	assert.True(t, job.EmailNotifications.NoAlertForSkippedRuns)

	require.NoError(t, setValue(job, "tags['env']", nil))
	assert.Equal(t, map[string]string{"team": "eng"}, job.Tags)

	require.NoError(t, setValue(job, "tasks[0]", nil))
	assert.Equal(t, []jobs.Task{{TaskKey: "b"}}, job.Tasks) //exhaustruct:ignore

	// A list from the value library arrives as []any and has to be decoded into the
	// field's own element type.
	require.NoError(t, setValue(job, "email_notifications.on_failure", []any{"a@b.test"}))
	assert.Equal(t, []string{"a@b.test"}, job.EmailNotifications.OnFailure)
}
