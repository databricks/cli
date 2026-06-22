package aircmd

import (
	"io"
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOrNA(t *testing.T) {
	assert.Equal(t, "x", orNA("x"))
	assert.Equal(t, "N/A", orNA(""))
}

func TestSubmittedDisplay(t *testing.T) {
	assert.Equal(t, "N/A", submittedDisplay(&jobs.Run{}))
	// 1700000000000 ms == 2023-11-14 22:13:20 UTC.
	assert.Equal(t, "2023-11-14 22:13 UTC", submittedDisplay(&jobs.Run{StartTime: 1700000000000}))
}

func TestOSC8Link(t *testing.T) {
	assert.Equal(t, "\x1b]8;;https://h.test/x\x1b\\label\x1b]8;;\x1b\\", osc8Link("label", "https://h.test/x"))
}

func TestHyperlink(t *testing.T) {
	// On a non-terminal (no color), the URL is dropped and only the label shows.
	ctx := cmdio.MockDiscard(t.Context())
	assert.Equal(t, "label", hyperlink(ctx, io.Discard, "label", "https://h.test/x"))
	// An empty URL is always rendered as the bare label.
	assert.Equal(t, "label", hyperlink(ctx, io.Discard, "label", ""))
}

func TestFormatDuration(t *testing.T) {
	cases := []struct {
		seconds int64
		want    string
	}{
		{0, "0s"},
		{45, "45s"},
		{60, "1m"},
		{63, "1m 3s"},
		{3600, "1h"},
		{3723, "1h 2m 3s"},
		{7260, "2h 1m"},
	}
	for _, c := range cases {
		assert.Equal(t, c.want, formatDuration(c.seconds))
	}
}

func TestStripExperimentUserPrefix(t *testing.T) {
	cases := []struct {
		name string
		want string
	}{
		{"/Users/me@example.com/my-experiment", "my-experiment"},
		{"/Users/me@example.com/nested/path", "nested/path"},
		{"my-experiment", "my-experiment"},
		{"/Shared/team-experiment", "/Shared/team-experiment"},
		{"/Users/me@example.com", "/Users/me@example.com"},
	}
	for _, c := range cases {
		assert.Equal(t, c.want, stripExperimentUserPrefix(c.name))
	}
}

func TestGpuDisplayName(t *testing.T) {
	assert.Equal(t, "H100", gpuDisplayName("h100_80gb"))
	assert.Equal(t, "A10", gpuDisplayName("GPU_1xA10"))
	assert.Equal(t, "A10", gpuDisplayName("a10"))
	assert.Equal(t, "H100", gpuDisplayName("GPU_8xH100"))
	assert.Equal(t, "H100", gpuDisplayName("GPU_1xH100"))
	// Unknown identifiers pass through unchanged.
	assert.Equal(t, "b200", gpuDisplayName("b200"))
	assert.Empty(t, gpuDisplayName(""))
}

func TestRunStatusPrefersResultState(t *testing.T) {
	// Result state wins once the run has finished.
	assert.Equal(t, "SUCCESS", runStatus(&jobs.RunState{
		LifeCycleState: jobs.RunLifeCycleStateTerminated,
		ResultState:    jobs.RunResultStateSuccess,
	}))
	// Before completion only the lifecycle state is set.
	assert.Equal(t, "RUNNING", runStatus(&jobs.RunState{
		LifeCycleState: jobs.RunLifeCycleStateRunning,
	}))
	// Non-nil state with neither field set, and nil state.
	assert.Equal(t, "UNKNOWN", runStatus(&jobs.RunState{}))
	assert.Equal(t, "UNKNOWN", runStatus(nil))
}

func TestReportedTiming(t *testing.T) {
	// No tasks: run-level times are used.
	start, end := reportedTiming(&jobs.Run{StartTime: 100, EndTime: 200})
	assert.Equal(t, int64(100), start)
	assert.Equal(t, int64(200), end)

	// The last task's window is preferred over the run-level window, so a
	// retried run reports its most recent attempt.
	start, end = reportedTiming(&jobs.Run{
		StartTime: 100, EndTime: 200,
		Tasks: []jobs.RunTask{
			{StartTime: 100, EndTime: 150},
			{StartTime: 300, EndTime: 450},
		},
	})
	assert.Equal(t, int64(300), start)
	assert.Equal(t, int64(450), end)

	// A task missing a field falls back to the run-level value for that field.
	start, end = reportedTiming(&jobs.Run{
		StartTime: 100, EndTime: 200,
		Tasks: []jobs.RunTask{{StartTime: 300}},
	})
	assert.Equal(t, int64(300), start)
	assert.Equal(t, int64(200), end)
}

func TestStartedAt(t *testing.T) {
	// Not started yet.
	assert.Nil(t, startedAt(&jobs.Run{}))
	// 1700000000000 ms == 2023-11-14T22:13:20+00:00 (Python isoformat, not "Z").
	got := startedAt(&jobs.Run{StartTime: 1700000000000})
	require.NotNil(t, got)
	assert.Equal(t, "2023-11-14T22:13:20+00:00", *got)
	// A sub-second start time carries microsecond precision.
	got = startedAt(&jobs.Run{StartTime: 1700000000500})
	require.NotNil(t, got)
	assert.Equal(t, "2023-11-14T22:13:20.500000+00:00", *got)
	// The last attempt's start time is reported. 1700000060000 ms == 22:14:20.
	got = startedAt(&jobs.Run{
		StartTime: 1700000000000,
		Tasks:     []jobs.RunTask{{StartTime: 1700000060000}},
	})
	require.NotNil(t, got)
	assert.Equal(t, "2023-11-14T22:14:20+00:00", *got)
}

func TestDurationSeconds(t *testing.T) {
	// Not started yet.
	assert.Nil(t, durationSeconds(&jobs.Run{}))

	// Finished run: reported end - start.
	d := durationSeconds(&jobs.Run{StartTime: 1700000000000, EndTime: 1700000012000})
	require.NotNil(t, d)
	assert.Equal(t, int64(12), *d)

	// Sub-second remainders round to the nearest second, matching Python: an
	// 11,500 ms run reports 12s, not 11s.
	d = durationSeconds(&jobs.Run{StartTime: 1700000000000, EndTime: 1700000011500})
	require.NotNil(t, d)
	assert.Equal(t, int64(12), *d)

	// The last attempt's window drives the duration for a retried run.
	d = durationSeconds(&jobs.Run{
		StartTime: 1700000000000, EndTime: 1700000012000,
		Tasks: []jobs.RunTask{{StartTime: 1700000000000, EndTime: 1700000005000}},
	})
	require.NotNil(t, d)
	assert.Equal(t, int64(5), *d)

	// Still running: measured against the current time, so positive.
	d = durationSeconds(&jobs.Run{StartTime: 1700000000000})
	require.NotNil(t, d)
	assert.Positive(t, *d)
}

func TestDashboardURL(t *testing.T) {
	// The ?o= workspace id and a trailing-slash-trimmed host, matching Python.
	assert.Equal(t, "https://example.test/jobs/runs/123?o=42", dashboardURL("https://example.test/", 123, 42))
}

func TestLatestAttemptNumber(t *testing.T) {
	assert.Equal(t, 0, latestAttemptNumber(&jobs.Run{}))
	run := &jobs.Run{Tasks: []jobs.RunTask{{AttemptNumber: 0}, {AttemptNumber: 2}}}
	assert.Equal(t, 2, latestAttemptNumber(run))
}

func TestExperimentName(t *testing.T) {
	assert.Nil(t, experimentName(&jobs.Run{}))
	assert.Nil(t, experimentName(&jobs.Run{Tasks: []jobs.RunTask{{}}}))
	assert.Nil(t, experimentName(&jobs.Run{Tasks: []jobs.RunTask{{
		GenAiComputeTask: &jobs.GenAiComputeTask{MlflowExperimentName: ""},
	}}}))
	got := experimentName(&jobs.Run{Tasks: []jobs.RunTask{{
		GenAiComputeTask: &jobs.GenAiComputeTask{MlflowExperimentName: "/Users/me@example.com/exp"},
	}}})
	require.NotNil(t, got)
	assert.Equal(t, "exp", *got)
}

func TestReformatYAMLForDisplay(t *testing.T) {
	// A multi-line command stored as a quoted "\n"-escaped string is re-rendered
	// as a `|` block literal, while key order is preserved.
	in := "experiment_name: foo\ncommand: \"set -e\\npython train.py --epochs 3\\n\"\nmax_retries: 2\n"
	want := "experiment_name: foo\ncommand: |\n  set -e\n  python train.py --epochs 3\nmax_retries: 2\n"
	assert.Equal(t, want, reformatYAMLForDisplay([]byte(in)))

	// A single-line command is left as a plain scalar.
	assert.Equal(t, "command: bash train.sh\n", reformatYAMLForDisplay([]byte("command: bash train.sh\n")))

	// A multi-line value with trailing whitespace can't be a block literal, so the
	// encoder falls back to a quoted style rather than emitting invalid YAML.
	got := reformatYAMLForDisplay([]byte("command: \"trailing space \\nsecond\"\n"))
	assert.Equal(t, "command: \"trailing space \\nsecond\"\n", got)

	// Unparseable content is returned unchanged.
	assert.Equal(t, "\tnot: valid: yaml:", reformatYAMLForDisplay([]byte("\tnot: valid: yaml:")))
}

func TestAccelerators(t *testing.T) {
	assert.Empty(t, accelerators(&jobs.Run{}))
	assert.Empty(t, accelerators(&jobs.Run{Tasks: []jobs.RunTask{{}}}))
	assert.Empty(t, accelerators(&jobs.Run{Tasks: []jobs.RunTask{{
		GenAiComputeTask: &jobs.GenAiComputeTask{},
	}}}))
	assert.Empty(t, accelerators(&jobs.Run{Tasks: []jobs.RunTask{{
		GenAiComputeTask: &jobs.GenAiComputeTask{Compute: &jobs.ComputeConfig{NumGpus: 0}},
	}}}))
	assert.Equal(t, "8x H100", accelerators(&jobs.Run{Tasks: []jobs.RunTask{{
		GenAiComputeTask: &jobs.GenAiComputeTask{Compute: &jobs.ComputeConfig{NumGpus: 8, GpuType: "GPU_8xH100"}},
	}}}))
}

func TestAcceleratorLabel(t *testing.T) {
	assert.Empty(t, acceleratorLabel("GPU_8xH100", 0))
	assert.Equal(t, "8x H100", acceleratorLabel("GPU_8xH100", 8))
	assert.Equal(t, "1x A10", acceleratorLabel("GPU_1xA10", 1))
	// The RPC may report a count without a recognized type.
	assert.Equal(t, "8x", acceleratorLabel("", 8))
}

func TestTrainingWorkflowStatus(t *testing.T) {
	cases := map[string]string{
		"TRAINING_WORKFLOW_STATE_PENDING":               "PENDING",
		"TRAINING_WORKFLOW_STATE_PENDING_SENT":          "PENDING",
		"TRAINING_WORKFLOW_STATE_RUNNING":               "RUNNING",
		"TRAINING_WORKFLOW_STATE_TERMINATION_REQUESTED": "TERMINATING",
		"TRAINING_WORKFLOW_STATE_TERMINATION_SENT":      "TERMINATING",
		"TRAINING_WORKFLOW_STATE_TERMINATED_COMPLETED":  "SUCCESS",
		"TRAINING_WORKFLOW_STATE_TERMINATED_FAILED":     "FAILED",
		"TRAINING_WORKFLOW_STATE_TERMINATED_STOPPED":    "CANCELED",
		"TRAINING_WORKFLOW_STATE_UNSPECIFIED":           "UNKNOWN",
		"":                                              "UNKNOWN",
	}
	for state, want := range cases {
		assert.Equal(t, want, trainingWorkflowStatus(state), state)
	}
}

func TestParseRPCTime(t *testing.T) {
	assert.True(t, parseRPCTime("").IsZero())
	assert.True(t, parseRPCTime("not-a-time").IsZero())
	got := parseRPCTime("2026-06-05T18:46:55.876Z")
	require.False(t, got.IsZero())
	assert.Equal(t, "2026-06-05T18:46:55.876000+00:00", isoFormat(got))
}
