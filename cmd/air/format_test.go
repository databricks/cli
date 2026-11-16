package aircmd

import (
	"encoding/json"
	"io"
	"strconv"
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
	assert.Equal(t, "B300", gpuDisplayName("GPU_8xB300"))
	assert.Equal(t, "RTX6000", gpuDisplayName("GPU_1xRTX6000"))
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
	assert.Equal(t, "https://example.test/jobs/runs/123", dashboardURL("https://example.test/", 123, 0))
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

func TestAcceleratorLabel(t *testing.T) {
	assert.Empty(t, acceleratorLabel("GPU_8xH100", 0))
	assert.Equal(t, "8x H100", acceleratorLabel("GPU_8xH100", 8))
	assert.Equal(t, "1x A10", acceleratorLabel("GPU_1xA10", 1))
	assert.Equal(t, "16x B300", acceleratorLabel("GPU_8xB300", 16))
	assert.Equal(t, "1x RTX6000", acceleratorLabel("GPU_1xRTX6000", 1))
	// The RPC may report a count without a recognized type.
	assert.Equal(t, "8x", acceleratorLabel("", 8))
}

func TestStatusWord(t *testing.T) {
	assert.Equal(t, "SUCCESS", statusWord("TERMINATED", "SUCCESS")) // result wins
	assert.Equal(t, "RUNNING", statusWord("RUNNING", ""))           // falls back to lifecycle
	assert.Equal(t, "UNKNOWN", statusWord("", ""))
}

func TestDisplayRunStatus(t *testing.T) {
	pendingStates := []jobs.RunLifeCycleState{
		jobs.RunLifeCycleStatePending,
		jobs.RunLifeCycleStateQueued,
		jobs.RunLifeCycleStateWaitingForRetry,
		jobs.RunLifeCycleStateBlocked,
	}
	for _, state := range pendingStates {
		run := &jobs.Run{
			State: &jobs.RunState{LifeCycleState: jobs.RunLifeCycleStateRunning},
			Tasks: []jobs.RunTask{{State: &jobs.RunState{LifeCycleState: state}}},
		}
		assert.Equal(t, "PENDING", displayRunStatus(run))
	}

	assert.Equal(t, "RUNNING", displayRunStatus(&jobs.Run{
		State: &jobs.RunState{LifeCycleState: jobs.RunLifeCycleStateRunning},
		Tasks: []jobs.RunTask{{State: &jobs.RunState{LifeCycleState: jobs.RunLifeCycleStateRunning}}},
	}))
	assert.Equal(t, "RUNNING", displayRunStatus(&jobs.Run{State: &jobs.RunState{LifeCycleState: jobs.RunLifeCycleStateRunning}}))
	assert.Equal(t, "FAILED", displayRunStatus(&jobs.Run{
		State: &jobs.RunState{LifeCycleState: jobs.RunLifeCycleStateTerminated, ResultState: jobs.RunResultStateFailed},
		Tasks: []jobs.RunTask{{State: &jobs.RunState{LifeCycleState: jobs.RunLifeCycleStatePending}}},
	}))
}

func TestDetailedDisplayRunStatus(t *testing.T) {
	waiting := &jobs.Run{
		State: &jobs.RunState{LifeCycleState: jobs.RunLifeCycleStateRunning},
		Tasks: []jobs.RunTask{{State: &jobs.RunState{LifeCycleState: jobs.RunLifeCycleStateQueued}}},
	}
	assert.Equal(t, "Waiting for GPU capacity...", detailedDisplayRunStatus(waiting, "Waiting for GPU capacity..."))
	assert.Equal(t, waitingForComputeStatus, detailedDisplayRunStatus(waiting, ""))

	started := &jobs.Run{
		State: &jobs.RunState{LifeCycleState: jobs.RunLifeCycleStateRunning},
		Tasks: []jobs.RunTask{{State: &jobs.RunState{LifeCycleState: jobs.RunLifeCycleStateRunning}}},
	}
	assert.Equal(t, "RUNNING", detailedDisplayRunStatus(started, ""))

	failed := &jobs.Run{
		State: &jobs.RunState{LifeCycleState: jobs.RunLifeCycleStateTerminated, ResultState: jobs.RunResultStateFailed},
		Tasks: []jobs.RunTask{{State: &jobs.RunState{LifeCycleState: jobs.RunLifeCycleStatePending}}},
	}
	assert.Equal(t, "FAILED", detailedDisplayRunStatus(failed, "stale status..."))
}

func TestTerminationReason(t *testing.T) {
	reason := terminationReason(&jobs.Run{
		State:  &jobs.RunState{ResultState: jobs.RunResultStateFailed, StateMessage: "parent reason"},
		Status: &jobs.RunStatus{TerminationDetails: &jobs.TerminationDetails{Message: " detailed reason "}},
	})
	require.NotNil(t, reason)
	assert.Equal(t, "detailed reason", *reason)

	reason = terminationReason(&jobs.Run{
		State: &jobs.RunState{ResultState: jobs.RunResultStateTimedout},
		Tasks: []jobs.RunTask{{State: &jobs.RunState{StateMessage: "task timed out"}}},
	})
	require.NotNil(t, reason)
	assert.Equal(t, "task timed out", *reason)

	assert.Nil(t, terminationReason(&jobs.Run{State: &jobs.RunState{ResultState: jobs.RunResultStateSuccess, StateMessage: "done"}}))
	assert.Nil(t, terminationReason(&jobs.Run{State: &jobs.RunState{ResultState: jobs.RunResultStateCanceled, StateMessage: "cancelled"}}))
}

func TestMaxRetries(t *testing.T) {
	submit := func(task jobs.RunTask) *jobs.Run {
		return &jobs.Run{RunType: jobs.RunTypeSubmitRun, Tasks: []jobs.RunTask{task}}
	}
	aiRuntime := jobs.RunTask{AiRuntimeTask: &jobs.AiRuntimeTask{}}
	reported := func(n int) jobs.RunTask {
		return jobs.RunTask{MaxRetries: n, AiRuntimeTask: &jobs.AiRuntimeTask{}, ForceSendFields: []string{"MaxRetries"}}
	}
	legacy := jobs.RunTask{GenAiComputeTask: &jobs.GenAiComputeTask{}}
	jobRun := &jobs.Run{RunType: jobs.RunTypeJobRun, Tasks: []jobs.RunTask{aiRuntime}}
	tests := []struct {
		name   string
		run    *jobs.Run
		config string
		want   string
	}{
		{"submit, config sets 1", submit(aiRuntime), "max_retries: 1\n", "1"},
		{"submit, config sets 0", submit(aiRuntime), "max_retries: 0\n", "0"},
		{"submit, config sets unlimited", submit(aiRuntime), "max_retries: -1\n", "unlimited"},
		{"submit, config unset uses the CLI default", submit(aiRuntime), "experiment_name: e\n", "3 (CLI default)"},
		{"submit, no config", submit(aiRuntime), "", "N/A"},
		{"submit, unparsable config", submit(aiRuntime), "[not: a map", "N/A"},
		{"legacy submit, config unset is unknown", submit(legacy), "experiment_name: e\n", "N/A"},
		{"legacy submit, config sets 2", submit(legacy), "max_retries: 2\n", "2"},
		{"job run ignores the config", jobRun, "max_retries: 1\n", "N/A"},
		{"job run, config unset", jobRun, "experiment_name: e\n", "N/A"},
		{"run type unknown", &jobs.Run{Tasks: []jobs.RunTask{aiRuntime}}, "max_retries: 1\n", "N/A"},
		{"server reports 0 over the config", submit(reported(0)), "max_retries: 2\n", "0"},
		{"server reports 5 on a job run", &jobs.Run{RunType: jobs.RunTypeJobRun, Tasks: []jobs.RunTask{reported(5)}}, "", "5"},
		{"server reports unlimited", submit(reported(-1)), "", "unlimited"},
		{"no tasks", &jobs.Run{RunType: jobs.RunTypeSubmitRun}, "", "N/A"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var raw []byte
			if tt.config != "" {
				raw = []byte(tt.config)
			}
			assert.Equal(t, tt.want, maxRetriesDisplay(resolveMaxRetries(tt.run, raw)))
		})
	}
}

func TestResolveMaxRetriesSource(t *testing.T) {
	run := &jobs.Run{RunType: jobs.RunTypeSubmitRun, Tasks: []jobs.RunTask{{AiRuntimeTask: &jobs.AiRuntimeTask{}}}}
	assert.Equal(t, &runMaxRetries{Limit: 1, Source: maxRetriesConfigured}, resolveMaxRetries(run, []byte("max_retries: 1\n")))
	assert.Equal(t, &runMaxRetries{Limit: 3, Source: maxRetriesCLIDefault}, resolveMaxRetries(run, []byte("command: x\n")))
	run.Tasks[0].MaxRetries = 2
	assert.Equal(t, &runMaxRetries{Limit: 2, Source: maxRetriesReported}, resolveMaxRetries(run, nil))
}

// The runs/get bodies below are trimmed copies of real workspace responses:
// task runs carry no max_retries key at all.
func TestMaxRetriesFromDecodedRunGet(t *testing.T) {
	decode := func(body string) *jobs.Run {
		var run jobs.Run
		require.NoError(t, json.Unmarshal([]byte(body), &run))
		return &run
	}
	display := func(run *jobs.Run, config []byte) string {
		return maxRetriesDisplay(resolveMaxRetries(run, config))
	}
	// Run 111111111111111: submitted with max_retries 1, failed, retried once.
	aiRuntime := decode(`{"run_id": 111111111111111, "run_type": "SUBMIT_RUN", "tasks": [
		{"task_key": "t", "run_id": 222222222222222, "attempt_number": 0, "ai_runtime_task": {"experiment": "e"}},
		{"task_key": "t", "run_id": 333333333333333, "attempt_number": 1, "ai_runtime_task": {"experiment": "e"}}]}`)
	assert.Equal(t, "1", display(aiRuntime, []byte("experiment_name: e\nmax_retries: 1\n")))
	assert.Equal(t, "3 (CLI default)", display(aiRuntime, []byte("experiment_name: e\n")))

	// A multi-container run has no deployment-level command_path, so the config
	// path is unknown and nothing is downloaded.
	multi := decode(`{"run_id": 1, "run_type": "SUBMIT_RUN", "tasks": [{"task_key": "t", "run_id": 2,
		"ai_runtime_task": {"experiment": "e", "deployments": [{"compute": {"accelerator_type": "GPU_1xA10", "accelerator_count": 1}}]}}]}`)
	var data getData
	enrichFromAiRuntimeTask(multi, &data)
	assert.Empty(t, data.TrainingConfigPath)
	assert.Equal(t, "N/A", display(multi, nil))

	// A legacy run as the Python CLI submitted it: yaml_parameters is the user's
	// raw YAML, and max_retries is sent only when > 0.
	legacyYAML := "experiment_name: my-exp\ncompute:\n  accelerator_type: a10\n  num_accelerators: 1\ncommand: python train.py\n"
	legacy := decode(`{"run_id": 1, "run_type": "SUBMIT_RUN", "tasks": [{"task_key": "train", "run_id": 2, "attempt_number": 0,
		"gen_ai_compute_task": {"mlflow_experiment_name": "/Users/u/my-exp", "yaml_parameters": ` + strconv.Quote(legacyYAML) + `}}]}`)
	_, raw := configYAML(t.Context(), nil, genAIComputeTask(legacy))
	assert.Equal(t, "N/A", display(legacy, raw))

	explicitZero := decode(`{"run_type": "SUBMIT_RUN", "tasks": [{"max_retries": 0, "ai_runtime_task": {}}]}`)
	assert.Equal(t, "0", display(explicitZero, []byte("experiment_name: e\n")))
}
