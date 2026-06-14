package aircmd

import (
	"testing"

	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

func TestStartedAt(t *testing.T) {
	// Not started yet.
	assert.Nil(t, startedAt(&jobs.Run{}))
	// 1700000000000 ms == 2023-11-14T22:13:20Z.
	got := startedAt(&jobs.Run{StartTime: 1700000000000})
	require.NotNil(t, got)
	assert.Equal(t, "2023-11-14T22:13:20Z", *got)
}

func TestDurationSeconds(t *testing.T) {
	// Not started yet.
	assert.Nil(t, durationSeconds(&jobs.Run{}))

	// Backend-provided duration wins (milliseconds → seconds).
	d := durationSeconds(&jobs.Run{StartTime: 1700000000000, RunDuration: 5000})
	require.NotNil(t, d)
	assert.Equal(t, int64(5), *d)

	// Finished run with no RunDuration: end - start.
	d = durationSeconds(&jobs.Run{StartTime: 1700000000000, EndTime: 1700000012000})
	require.NotNil(t, d)
	assert.Equal(t, int64(12), *d)

	// Still running: measured against the current time, so positive.
	d = durationSeconds(&jobs.Run{StartTime: 1700000000000})
	require.NotNil(t, d)
	assert.Positive(t, *d)
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
