package ai

import (
	"testing"

	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/assert"
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
	// Unknown identifiers pass through unchanged.
	assert.Equal(t, "b200", gpuDisplayName("b200"))
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
	assert.Equal(t, "UNKNOWN", runStatus(nil))
}
