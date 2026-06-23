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

	err := waitForJobToStart(ctx, m.WorkspaceClient, 1, ClientOptions{TaskStartupTimeout: 30 * time.Second})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ssh server bootstrap job failed")
	assert.Contains(t, err.Error(), "Could not reach driver of cluster 0605-x.")
}

// hostKeyFailureStderr is the relevant tail of ssh's stderr when strict checking aborts a
// connection because the remote host key changed.
const hostKeyFailureStderr = `@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@
@    WARNING: REMOTE HOST IDENTIFICATION HAS CHANGED!     @
@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@
Host key for databricks-cpu-6e7644d0 has changed and you have requested strict checking.
Host key verification failed.`

func TestHostKeyChangedHint(t *testing.T) {
	tests := []struct {
		name           string
		stderr         string
		hostName       string
		knownHostsFile string
		wantContains   []string
		wantEmpty      bool
	}{
		{
			name:         "host key failure",
			stderr:       hostKeyFailureStderr,
			hostName:     "databricks-cpu-6e7644d0",
			wantContains: []string{"databricks-cpu-6e7644d0", "ssh-keygen -R databricks-cpu-6e7644d0"},
		},
		{
			name:           "host key failure with custom known_hosts file",
			stderr:         hostKeyFailureStderr,
			hostName:       "databricks-cpu-6e7644d0",
			knownHostsFile: "/tmp/known_hosts",
			wantContains:   []string{"ssh-keygen -R databricks-cpu-6e7644d0 -f /tmp/known_hosts"},
		},
		{
			name:      "unrelated failure",
			stderr:    "kex_exchange_identification: Connection closed by remote host",
			hostName:  "databricks-cpu-6e7644d0",
			wantEmpty: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hostKeyChangedHint(tt.stderr, tt.hostName, tt.knownHostsFile)
			if tt.wantEmpty {
				assert.Empty(t, got)
				return
			}
			for _, want := range tt.wantContains {
				assert.Contains(t, got, want)
			}
		})
	}
}

func TestBuildRemoteShellArgs(t *testing.T) {
	const bashCmd = `command -v bash >/dev/null 2>&1 && exec bash -l || exec "${SHELL:-/bin/sh}" -l`

	t.Run("interactive returns login bash command", func(t *testing.T) {
		args := buildRemoteShellArgs(ClientOptions{})
		require.Len(t, args, 1)
		assert.Equal(t, bashCmd, args[0])
	})

	t.Run("non-interactive passes additional args verbatim", func(t *testing.T) {
		additional := []string{"ls", "-la"}
		args := buildRemoteShellArgs(ClientOptions{AdditionalArgs: additional})
		assert.Equal(t, additional, args)
	})
}

func TestBuildSSHArgsPTYPlacement(t *testing.T) {
	indexOf := func(args []string, want string) int {
		for i, a := range args {
			if a == want {
				return i
			}
		}
		return -1
	}

	t.Run("interactive forces a PTY before the destination", func(t *testing.T) {
		args := buildSSHArgs("user", "/key", "proxy command", "myhost", ClientOptions{})
		ptyIdx := indexOf(args, "-t")
		hostIdx := indexOf(args, "myhost")
		require.NotEqual(t, -1, ptyIdx, "-t must be present for interactive sessions")
		require.NotEqual(t, -1, hostIdx)
		assert.Less(t, ptyIdx, hostIdx, "-t must precede the destination host")
		// The remote command is the final arg, after the host.
		assert.Greater(t, len(args)-1, hostIdx)
		assert.Contains(t, args[len(args)-1], "exec bash -l")
	})

	t.Run("non-interactive does not force a PTY", func(t *testing.T) {
		args := buildSSHArgs("user", "/key", "proxy command", "myhost", ClientOptions{AdditionalArgs: []string{"ls", "-la"}})
		assert.Equal(t, -1, indexOf(args, "-t"), "no PTY for non-interactive passthrough")
		hostIdx := indexOf(args, "myhost")
		require.NotEqual(t, -1, hostIdx)
		assert.Equal(t, []string{"ls", "-la"}, args[hostIdx+1:], "additional args follow the host verbatim")
	})
}

func TestTailWriterRetainsTail(t *testing.T) {
	t.Run("retains only the tail", func(t *testing.T) {
		w := &tailWriter{maxBytes: 4}
		n, err := w.Write([]byte("abcdefgh"))
		require.NoError(t, err)
		assert.Equal(t, 8, n)
		assert.Equal(t, "efgh", w.String())
	})

	t.Run("preserves a short write", func(t *testing.T) {
		w := &tailWriter{maxBytes: 4}
		_, err := w.Write([]byte("ab"))
		require.NoError(t, err)
		assert.Equal(t, "ab", w.String())
	})
}
