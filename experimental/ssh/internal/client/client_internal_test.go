package client

import (
	"strings"
	"testing"
	"time"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// terminatedRun builds a job run whose SSH server task has terminated, for the failure-surfacing tests.
func terminatedRun(runID, taskRunID int64, message, pageURL string) *jobs.Run {
	return &jobs.Run{
		RunId:      runID,
		RunPageUrl: pageURL,
		Tasks: []jobs.RunTask{{
			TaskKey: sshServerTaskKey,
			RunId:   taskRunID,
			Status: &jobs.RunStatus{
				State:              jobs.RunLifecycleStateV2StateTerminated,
				TerminationDetails: &jobs.TerminationDetails{Message: message},
			},
		}},
	}
}

func TestDescribeRunFailureIncludesMessageTraceAndURL(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	m := mocks.NewMockWorkspaceClient(t)
	api := m.GetMockJobsAPI()
	api.EXPECT().GetRun(mock.Anything, jobs.GetRunRequest{RunId: 1}).Return(
		terminatedRun(1, 99, "Could not reach driver of cluster 0605-x.", "https://example.test/run/1"), nil)
	api.EXPECT().GetRunOutput(mock.Anything, jobs.GetRunOutputRequest{RunId: 99}).Return(
		&jobs.RunOutput{Error: "Run failed with error message", ErrorTrace: "Traceback (most recent call last): boom"}, nil)

	out := describeRunFailure(ctx, m.WorkspaceClient, 1)
	assert.Contains(t, out, "Could not reach driver of cluster 0605-x.")
	assert.Contains(t, out, "Run failed with error message")
	assert.Contains(t, out, "Traceback (most recent call last): boom")
	assert.Contains(t, out, "https://example.test/run/1")
}

func TestDescribeRunFailureTruncatesLongTrace(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	m := mocks.NewMockWorkspaceClient(t)
	api := m.GetMockJobsAPI()
	longTrace := strings.Repeat("x", maxRunFailureTraceBytes+500) + "TAIL_MARKER"
	api.EXPECT().GetRun(mock.Anything, jobs.GetRunRequest{RunId: 1}).Return(
		terminatedRun(1, 99, "", "https://example.test/run/1"), nil)
	api.EXPECT().GetRunOutput(mock.Anything, jobs.GetRunOutputRequest{RunId: 99}).Return(
		&jobs.RunOutput{ErrorTrace: longTrace}, nil)

	out := describeRunFailure(ctx, m.WorkspaceClient, 1)
	assert.Contains(t, out, "...")
	assert.Contains(t, out, "TAIL_MARKER")
	// The leading run of 'x' is dropped by truncation.
	assert.NotContains(t, out, strings.Repeat("x", maxRunFailureTraceBytes+1))
}

func TestDescribeRunFailureDeduplicatesErrorInTrace(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	m := mocks.NewMockWorkspaceClient(t)
	api := m.GetMockJobsAPI()
	errMsg := "SSH server exited with code 1. Last server logs:\nLOG_MARKER"
	api.EXPECT().GetRun(mock.Anything, jobs.GetRunRequest{RunId: 1}).Return(
		terminatedRun(1, 99, "", "https://example.test/run/1"), nil)
	api.EXPECT().GetRunOutput(mock.Anything, jobs.GetRunOutputRequest{RunId: 99}).Return(
		&jobs.RunOutput{Error: errMsg, ErrorTrace: "Traceback (most recent call last):\n  boom\nRuntimeError: " + errMsg}, nil)

	out := describeRunFailure(ctx, m.WorkspaceClient, 1)
	assert.Contains(t, out, "Traceback (most recent call last):")
	assert.Equal(t, 1, strings.Count(out, "LOG_MARKER"))
}

func TestDescribeRunFailureTruncatesLongError(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	m := mocks.NewMockWorkspaceClient(t)
	api := m.GetMockJobsAPI()
	longError := strings.Repeat("x", maxRunFailureTraceBytes+500) + "TAIL_MARKER"
	api.EXPECT().GetRun(mock.Anything, jobs.GetRunRequest{RunId: 1}).Return(
		terminatedRun(1, 99, "", "https://example.test/run/1"), nil)
	api.EXPECT().GetRunOutput(mock.Anything, jobs.GetRunOutputRequest{RunId: 99}).Return(
		&jobs.RunOutput{Error: longError}, nil)

	out := describeRunFailure(ctx, m.WorkspaceClient, 1)
	assert.Contains(t, out, "...")
	assert.Contains(t, out, "TAIL_MARKER")
	assert.NotContains(t, out, strings.Repeat("x", maxRunFailureTraceBytes+1))
}

func TestDescribeRunFailureNoRunID(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	m := mocks.NewMockWorkspaceClient(t)
	out := describeRunFailure(ctx, m.WorkspaceClient, 0)
	assert.Contains(t, out, "no job run ID")
}

func TestRunFailureIfTerminated(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())

	t.Run("terminated", func(t *testing.T) {
		m := mocks.NewMockWorkspaceClient(t)
		api := m.GetMockJobsAPI()
		api.EXPECT().GetRun(mock.Anything, jobs.GetRunRequest{RunId: 1}).Return(
			terminatedRun(1, 99, "boom", "https://example.test/run/1"), nil)
		api.EXPECT().GetRunOutput(mock.Anything, jobs.GetRunOutputRequest{RunId: 99}).Return(
			&jobs.RunOutput{}, nil)

		desc, terminated := runFailureIfTerminated(ctx, m.WorkspaceClient, 1)
		assert.True(t, terminated)
		assert.Contains(t, desc, "boom")
	})

	t.Run("still running", func(t *testing.T) {
		m := mocks.NewMockWorkspaceClient(t)
		api := m.GetMockJobsAPI()
		api.EXPECT().GetRun(mock.Anything, jobs.GetRunRequest{RunId: 1}).Return(&jobs.Run{
			RunId: 1,
			Tasks: []jobs.RunTask{{
				TaskKey: sshServerTaskKey,
				Status:  &jobs.RunStatus{State: jobs.RunLifecycleStateV2StateRunning},
			}},
		}, nil)

		_, terminated := runFailureIfTerminated(ctx, m.WorkspaceClient, 1)
		assert.False(t, terminated)
	})
}

func TestWaitForJobToStartSurfacesFailure(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	m := mocks.NewMockWorkspaceClient(t)
	api := m.GetMockJobsAPI()
	api.EXPECT().GetRun(mock.Anything, jobs.GetRunRequest{RunId: 1}).Return(
		terminatedRun(1, 99, "Could not reach driver of cluster 0605-x.", "https://example.test/run/1"), nil)
	api.EXPECT().GetRunOutput(mock.Anything, jobs.GetRunOutputRequest{RunId: 99}).Return(
		&jobs.RunOutput{}, nil)

	err := waitForJobToStart(ctx, m.WorkspaceClient, 1, 30*time.Second)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ssh server bootstrap job failed")
	assert.Contains(t, err.Error(), "Could not reach driver of cluster 0605-x.")
}
