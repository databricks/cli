package tests

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/databricks/cli/libs/structs/structpath"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/databricks/databricks-sdk-go/service/sql"
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

	// Only the last element can be made absent: removing an earlier one would shift its
	// successor into the same path, so the field would still be there holding another value.
	require.ErrorContains(t, setValue(job, "tasks[0]", nil), "would shift into it")
	require.NoError(t, setValue(job, "tasks[1]", nil))
	assert.Equal(t, []jobs.Task{{TaskKey: "a"}}, job.Tasks) //exhaustruct:ignore
	require.NoError(t, setValue(job, "tasks[0]", nil))
	assert.Empty(t, job.Tasks)

	// A list from the value library arrives as []any and has to be decoded into the
	// field's own element type.
	require.NoError(t, setValue(job, "email_notifications.on_failure", []any{"a@b.test"}))
	assert.Equal(t, []string{"a@b.test"}, job.EmailNotifications.OnFailure)
}

// A skip key may be a pattern, since a field inside a slice has no fixed index and the
// value library cannot name one. Written as a table because the three key shapes -- exact,
// element wildcard, subtree -- take different paths through the matcher.
func TestSkipReasonMatchesPatterns(t *testing.T) {
	fv := &fieldValues{skip: map[string]string{
		"storage_root":       "needs an external location",
		"aliases[*].id":      "assigned by the backend",
		"telemetry_config.*": "only accepted as a whole",
	}} //exhaustruct:ignore

	for _, tc := range []struct {
		path   string
		reason string
	}{
		{"storage_root", "needs an external location"},
		{"aliases[0].id", "assigned by the backend"},
		{"aliases[3].id", "assigned by the backend"},
		{"telemetry_config.enabled", "only accepted as a whole"},
		// A pattern of its own, which is what a field under an absent container is.
		{"telemetry_config.sinks[*].name", "only accepted as a whole"},
		{"aliases[0].alias_name", ""},
		{"comment", ""},
		// A key naming a whole subtree, written with the trailing wildcard, still covers a
		// pattern beneath it -- which is the form a field under an absent container takes.
		{"telemetry_config.sinks[*].endpoint", "only accepted as a whole"},
	} {
		t.Run(tc.path, func(t *testing.T) {
			reason, skipped := fv.skipReason(tc.path)
			assert.Equal(t, tc.reason != "", skipped)
			assert.Equal(t, tc.reason, reason)
		})
	}
}

// The generated required-field data is keyed by pattern, so a concrete index has to be
// turned back into a wildcard before the lookup -- otherwise a required field inside a
// slice looks optional and gets an "absent" transition a user cannot deploy.
func TestIsRequiredInsideSlice(t *testing.T) {
	assert.True(t, isRequired("jobs", "tasks[0].task_key"))
	assert.True(t, isRequired("jobs", "tasks[3].task_key"))
	assert.False(t, isRequired("jobs", "tasks[0].description"))

	assert.True(t, isRequired("alerts", "display_name"))
	assert.False(t, isRequired("alerts", "custom_summary"))
}

// Every id in a message has to redact to the same placeholder on both sides, or a report from
// a real workspace could never match one from the fake server: the fake hands out UUIDs where
// a workspace hands out hex.
func TestRedactIDs(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{"updating id=79f5e1f2-de43-03a0-79f5-e1f2de4303a1: nope", "updating id=[ID]: nope"},
		{"updating id=00145E75FBD6DCBB: nope", "updating id=[ID]: nope"},
		{"Catalog 'test-schema-f0ad2bd52de6e45119d42' missing", "Catalog 'test-schema-[UNIQUE_NAME]' missing"},
		{"principal 79f5e1f2-de43-03a0-79f5-e1f2de4303a1 denied", "principal [UUID] denied"},
	} {
		assert.Equal(t, tc.want, redactIDs(tc.in))
	}
}

// The planner names a slice element by a key-value selector where this suite names it by index,
// so the two forms have to be recognised as the same field -- otherwise a change recorded
// against "tasks[task_key='seeded'].run_if" looks unrelated to "tasks[0].run_if".
func TestSameField(t *testing.T) {
	for _, tc := range []struct {
		drifting, path string
		want           bool
	}{
		{"comment", "comment", true},
		{"tasks", "tasks[0].run_if", true},
		{"tasks[task_key='seeded'].run_if", "tasks[0].run_if", true},
		{"tasks[task_key='seeded']", "tasks[0].run_if", true},
		{"tasks[task_key='seeded'].notebook_task.source", "tasks[0].notebook_task", true},
		{"config", "config.auto_capture_config.enabled", true},
		{"comment", "name", false},
		{"tasks[task_key='seeded'].run_if", "tasks[0].timeout_seconds", false},
	} {
		assert.Equal(t, tc.want, sameField(tc.drifting, tc.path), "%s vs %s", tc.drifting, tc.path)
	}
}

// An SDK enum declares its own values, and the *_UNSPECIFIED member is a protobuf sentinel for
// "unset" rather than a choice: asking for it reads as the write being ignored, since the
// backend normalizes it away.
func TestEnumValues(t *testing.T) {
	// AlertOperator has no UNSPECIFIED member, so the two alphabetically-first values come back.
	got := enumValues(reflect.TypeFor[sql.AlertOperator]())
	require.Len(t, got, 2)
	for _, value := range got {
		assert.IsType(t, sql.AlertOperator(""), value)
		assert.NotContains(t, value, "UNSPECIFIED")
	}

	// SpotInstancePolicy does have one, and it must not be offered.
	got = enumValues(reflect.TypeFor[sql.SpotInstancePolicy]())
	require.NotEmpty(t, got)
	for _, value := range got {
		assert.NotContains(t, fmt.Sprint(value), "UNSPECIFIED")
	}

	// A plain string is not an enum, whatever the backend constrains it to.
	assert.Nil(t, enumValues(reflect.TypeFor[string]()))
}

// A container's values are the resource's own and that value with one entry dropped, both deep
// copies -- a shallow one would share the pointers inside an element with the live resource, so
// a later edit would reach into a value recorded here.
func TestContainerValuesAreIndependent(t *testing.T) {
	//exhaustruct:ignore
	job := &resources.Job{JobSettings: jobs.JobSettings{
		Tasks: []jobs.Task{
			{TaskKey: "a", NotebookTask: &jobs.NotebookTask{NotebookPath: "one"}}, //exhaustruct:ignore
			{TaskKey: "b"}, //exhaustruct:ignore
		},
	}}

	values := containerValues(job, "tasks")
	require.Len(t, values, 2)
	full := values[0].([]jobs.Task)
	trimmed := values[1].([]jobs.Task)
	require.Len(t, full, 2)
	require.Len(t, trimmed, 1)

	// Editing the live resource through the pointer inside an element must not be visible in
	// either recorded value.
	require.NoError(t, setValue(job, "tasks[0].notebook_task.notebook_path", "two"))
	assert.Equal(t, "one", full[0].NotebookTask.NotebookPath)
	assert.Equal(t, "one", trimmed[0].NotebookTask.NotebookPath)
}

// A value long enough or punctuated enough to be unreadable as a subtest name gets a trimmed,
// digest-suffixed label -- stable, and safe to paste back into a test filter.
func TestShortLabel(t *testing.T) {
	assert.Equal(t, "x", shortLabel("x"))
	assert.Equal(t, "dbfs:/FileStore/a.jar", shortLabel("dbfs:/FileStore/a.jar"))

	long := shortLabel(`{"spark_version":{"type":"fixed","value":"13.3"}}`)
	assert.NotContains(t, long, `"`)
	assert.LessOrEqual(t, len(long), 24)
	// Two values sharing a prefix still get different labels.
	assert.NotEqual(t, long, shortLabel(`{"spark_version":{"type":"fixed","value":"14.3"}}`))
	// And the same value always gets the same one.
	assert.Equal(t, long, shortLabel(`{"spark_version":{"type":"fixed","value":"13.3"}}`))
}
