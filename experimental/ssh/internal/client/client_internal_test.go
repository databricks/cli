package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/databricks/cli/experimental/ssh/internal/proxy"
	"github.com/databricks/cli/experimental/ssh/internal/sshconfig"
	"github.com/databricks/cli/experimental/ssh/internal/vscode"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/telemetry/protos"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/databricks/databricks-sdk-go/service/environments"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestValidateClusterAccessSingleUser(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	m := mocks.NewMockWorkspaceClient(t)
	m.GetMockClustersAPI().EXPECT().Get(ctx, compute.GetClusterRequest{ClusterId: "cluster-123"}).Return(&compute.ClusterDetails{
		DataSecurityMode: compute.DataSecurityModeSingleUser,
		SingleUserName:   "me@example.com",
	}, nil)

	err := ValidateClusterAccess(ctx, m.WorkspaceClient, "cluster-123")
	assert.NoError(t, err)
}

// A dedicated cluster reporting the newer DATA_SECURITY_MODE_DEDICATED enum (rather than the
// legacy SINGLE_USER alias) must still pass validation.
func TestValidateClusterAccessDedicatedEnum(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	m := mocks.NewMockWorkspaceClient(t)
	m.GetMockClustersAPI().EXPECT().Get(ctx, compute.GetClusterRequest{ClusterId: "cluster-123"}).Return(&compute.ClusterDetails{
		DataSecurityMode: compute.DataSecurityModeDataSecurityModeDedicated,
		SingleUserName:   "me@example.com",
	}, nil)

	err := ValidateClusterAccess(ctx, m.WorkspaceClient, "cluster-123")
	assert.NoError(t, err)
}

func TestValidateClusterAccessInvalidAccessMode(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	m := mocks.NewMockWorkspaceClient(t)
	m.GetMockClustersAPI().EXPECT().Get(ctx, compute.GetClusterRequest{ClusterId: "cluster-123"}).Return(&compute.ClusterDetails{
		DataSecurityMode: compute.DataSecurityModeUserIsolation,
	}, nil)

	err := ValidateClusterAccess(ctx, m.WorkspaceClient, "cluster-123")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must be a dedicated single-user cluster")
	// The error surfaces the UI label ("Standard"), not the raw API enum (USER_ISOLATION).
	assert.Contains(t, err.Error(), "Current access mode: Standard")
	assert.NotContains(t, err.Error(), "USER_ISOLATION")
}

// A Dedicated cluster assigned to a group (no single_user_name) is rejected, and the error
// reports the mode specifically as "Dedicated (group)".
func TestValidateClusterAccessDedicatedGroup(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	m := mocks.NewMockWorkspaceClient(t)
	m.GetMockClustersAPI().EXPECT().Get(ctx, compute.GetClusterRequest{ClusterId: "cluster-123"}).Return(&compute.ClusterDetails{
		DataSecurityMode: compute.DataSecurityModeDataSecurityModeDedicated,
	}, nil)

	err := ValidateClusterAccess(ctx, m.WorkspaceClient, "cluster-123")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must be a dedicated single-user cluster")
	assert.Contains(t, err.Error(), "Current access mode: Dedicated (group)")
}

func TestAccessModeUILabel(t *testing.T) {
	tests := []struct {
		mode           compute.DataSecurityMode
		singleUserName string
		want           string
	}{
		{compute.DataSecurityModeSingleUser, "me@example.com", "Dedicated (single user)"},
		{compute.DataSecurityModeDataSecurityModeDedicated, "me@example.com", "Dedicated (single user)"},
		{compute.DataSecurityModeDataSecurityModeDedicated, "", "Dedicated (group)"},
		{compute.DataSecurityModeUserIsolation, "", "Standard"},
		{compute.DataSecurityModeDataSecurityModeStandard, "", "Standard"},
		{compute.DataSecurityModeNone, "", "No isolation"},
		// Legacy/unknown modes fall back to the raw API value.
		{compute.DataSecurityModeLegacyTableAcl, "", "LEGACY_TABLE_ACL"},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, accessModeUILabel(tt.mode, tt.singleUserName), "mode=%s", tt.mode)
	}
}

func TestValidateClusterAccessClusterNotFound(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	m := mocks.NewMockWorkspaceClient(t)
	m.GetMockClustersAPI().EXPECT().Get(ctx, compute.GetClusterRequest{ClusterId: "nonexistent"}).Return(nil, errors.New("cluster not found"))

	err := ValidateClusterAccess(ctx, m.WorkspaceClient, "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get cluster information for cluster ID 'nonexistent'")
}

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

func TestResolveBaseEnvironment(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())

	// Paths and resource IDs are passed through without hitting the API.
	t.Run("env.yaml path passthrough", func(t *testing.T) {
		m := mocks.NewMockWorkspaceClient(t)
		got, err := resolveBaseEnvironment(ctx, m.WorkspaceClient, "/Workspace/path/to/env.yaml")
		require.NoError(t, err)
		assert.Equal(t, "/Workspace/path/to/env.yaml", got)
	})

	t.Run("resource ID passthrough", func(t *testing.T) {
		m := mocks.NewMockWorkspaceClient(t)
		got, err := resolveBaseEnvironment(ctx, m.WorkspaceClient, "workspace-base-environments/dbe_123")
		require.NoError(t, err)
		assert.Equal(t, "workspace-base-environments/dbe_123", got)
	})

	t.Run("display name resolves to resource ID", func(t *testing.T) {
		m := mocks.NewMockWorkspaceClient(t)
		m.GetMockEnvironmentsAPI().EXPECT().
			ListWorkspaceBaseEnvironmentsAll(mock.Anything, environments.ListWorkspaceBaseEnvironmentsRequest{}).
			Return([]environments.WorkspaceBaseEnvironment{
				{DisplayName: "other", Name: "workspace-base-environments/dbe_other"},
				{DisplayName: "my-env", Name: "workspace-base-environments/dbe_mine"},
			}, nil)

		got, err := resolveBaseEnvironment(ctx, m.WorkspaceClient, "my-env")
		require.NoError(t, err)
		assert.Equal(t, "workspace-base-environments/dbe_mine", got)
	})

	t.Run("display name not found", func(t *testing.T) {
		m := mocks.NewMockWorkspaceClient(t)
		m.GetMockEnvironmentsAPI().EXPECT().
			ListWorkspaceBaseEnvironmentsAll(mock.Anything, environments.ListWorkspaceBaseEnvironmentsRequest{}).
			Return([]environments.WorkspaceBaseEnvironment{
				{DisplayName: "other", Name: "workspace-base-environments/dbe_other"},
			}, nil)

		_, err := resolveBaseEnvironment(ctx, m.WorkspaceClient, "my-env")
		require.Error(t, err)
		assert.Contains(t, err.Error(), `no workspace base environment found with display name "my-env"`)
	})

	t.Run("display name ambiguous", func(t *testing.T) {
		m := mocks.NewMockWorkspaceClient(t)
		m.GetMockEnvironmentsAPI().EXPECT().
			ListWorkspaceBaseEnvironmentsAll(mock.Anything, environments.ListWorkspaceBaseEnvironmentsRequest{}).
			Return([]environments.WorkspaceBaseEnvironment{
				{DisplayName: "my-env", Name: "workspace-base-environments/dbe_1"},
				{DisplayName: "my-env", Name: "workspace-base-environments/dbe_2"},
			}, nil)

		_, err := resolveBaseEnvironment(ctx, m.WorkspaceClient, "my-env")
		require.Error(t, err)
		assert.Contains(t, err.Error(), `multiple workspace base environments found with display name "my-env"`)
	})
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
		wantOmits      []string
		wantEmpty      bool
	}{
		{
			name:           "host key failure names the host and the pinned file",
			stderr:         hostKeyFailureStderr,
			hostName:       "databricks-cpu-6e7644d0",
			knownHostsFile: "/tmp/known_hosts",
			wantContains:   []string{"databricks-cpu-6e7644d0", "/tmp/known_hosts"},
		},
		{
			// The stale-entry advice this hint used to give no longer applies: the CLI
			// rewrites the entry from the workspace before every connection.
			name:           "host key failure does not blame a stale local entry",
			stderr:         hostKeyFailureStderr,
			hostName:       "databricks-cpu-6e7644d0",
			knownHostsFile: "/tmp/known_hosts",
			wantOmits:      []string{"ssh-keygen -R"},
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
			for _, unwanted := range tt.wantOmits {
				assert.NotContains(t, got, unwanted)
			}
		})
	}
}

func TestBuildRemoteShellArgs(t *testing.T) {
	const bashCmd = `command -v bash >/dev/null 2>&1 && exec bash -i || exec "${SHELL:-/bin/sh}" -i`
	const wsHome = "/Workspace/Users/me@example.com"

	// A serverless session has a connection name and no cluster ID.
	serverless := ClientOptions{ConnectionName: "my-conn"}

	t.Run("interactive cds into the workspace home, then opens bash", func(t *testing.T) {
		args := buildRemoteShellArgs(serverless, wsHome)
		assert.Equal(t, []string{`cd '` + wsHome + `' 2>/dev/null; ` + bashCmd}, args)
	})

	t.Run("interactive without a workspace home just opens bash", func(t *testing.T) {
		args := buildRemoteShellArgs(serverless, "")
		assert.Equal(t, []string{bashCmd}, args)
	})

	t.Run("non-interactive passes additional args verbatim", func(t *testing.T) {
		additional := []string{"ls", "-la"}
		args := buildRemoteShellArgs(ClientOptions{ConnectionName: "my-conn", AdditionalArgs: additional}, wsHome)
		assert.Equal(t, additional, args)
	})
}

func TestBuildSSHArgsSetsServerAliveInterval(t *testing.T) {
	args := buildSSHArgs("user", "/key", "/pins/myhost", "proxy command", "myhost", "", ClientOptions{})

	// ssh stops parsing options at the destination, so an option placed after the host would be
	// treated as part of the remote command rather than as an ssh option.
	optIdx := slices.Index(args, "ServerAliveInterval="+strconv.Itoa(sshconfig.ServerAliveIntervalSeconds))
	require.NotEqual(t, -1, optIdx, "ssh must be asked to send keepalives")
	require.Equal(t, "-o", args[optIdx-1])
	assert.Less(t, optIdx, slices.Index(args, "myhost"), "the option must precede the destination host")
}

func TestBuildSSHArgsPinsHostKey(t *testing.T) {
	args := buildSSHArgs("user", "/key", "/pins/myhost", "proxy command", "myhost", "", ClientOptions{})

	// The pinned file is the whole point of strict checking here: without it ssh would
	// fall back to ~/.ssh/known_hosts, where an entry for this name may be left over from
	// other compute (DECO-27882).
	hostIdx := slices.Index(args, "myhost")
	for _, want := range []string{"StrictHostKeyChecking=yes", "UserKnownHostsFile=/pins/myhost"} {
		optIdx := slices.Index(args, want)
		require.NotEqual(t, -1, optIdx, "%s must be passed to ssh", want)
		require.Equal(t, "-o", args[optIdx-1])
		assert.Less(t, optIdx, hostIdx, "the option must precede the destination host")
	}
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
		args := buildSSHArgs("user", "/key", "/pins/myhost", "proxy command", "myhost", "/Workspace/Users/me@example.com", ClientOptions{})
		ptyIdx := indexOf(args, "-t")
		hostIdx := indexOf(args, "myhost")
		require.NotEqual(t, -1, ptyIdx, "-t must be present for interactive sessions")
		require.NotEqual(t, -1, hostIdx)
		assert.Less(t, ptyIdx, hostIdx, "-t must precede the destination host")
		// The remote command is the final arg, after the host.
		assert.Greater(t, len(args)-1, hostIdx)
		assert.Contains(t, args[len(args)-1], "exec bash -i")
	})

	t.Run("non-interactive does not force a PTY", func(t *testing.T) {
		args := buildSSHArgs("user", "/key", "/pins/myhost", "proxy command", "myhost", "", ClientOptions{AdditionalArgs: []string{"ls", "-la"}})
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

func TestBuildSshTunnelEvent(t *testing.T) {
	tests := []struct {
		name string
		opts ClientOptions
		want protos.SshTunnelEvent
	}{
		{
			name: "dedicated cluster via raw SSH client",
			opts: ClientOptions{ClusterID: "abc-123", AutoStartCluster: true},
			want: protos.SshTunnelEvent{
				ComputeType:      protos.SshTunnelComputeTypeDedicated,
				ClientMode:       protos.SshTunnelClientModeSSH,
				AutoStartCluster: true,
			},
		},
		{
			name: "serverless with accelerator",
			opts: ClientOptions{ConnectionName: "my-conn", Accelerator: "GPU_1xA10"},
			want: protos.SshTunnelEvent{
				ComputeType:     protos.SshTunnelComputeTypeServerless,
				AcceleratorType: "GPU_1xA10",
				ClientMode:      protos.SshTunnelClientModeSSH,
			},
		},
		{
			name: "proxy mode takes precedence over IDE",
			opts: ClientOptions{ConnectionName: "my-conn", ProxyMode: true, IDE: "vscode"},
			want: protos.SshTunnelEvent{
				ComputeType: protos.SshTunnelComputeTypeServerless,
				IdeType:     "vscode",
				ClientMode:  protos.SshTunnelClientModeProxy,
			},
		},
		{
			name: "IDE mode",
			opts: ClientOptions{ConnectionName: "my-conn", IDE: "cursor"},
			want: protos.SshTunnelEvent{
				ComputeType: protos.SshTunnelComputeTypeServerless,
				IdeType:     "cursor",
				ClientMode:  protos.SshTunnelClientModeIDE,
			},
		},
		{
			// The raw --base-environment value can carry PII, so only its presence is recorded.
			name: "custom base environment records presence only",
			opts: ClientOptions{ConnectionName: "my-conn", BaseEnvironment: "/Workspace/Users/me@example.com/env.yaml"},
			want: protos.SshTunnelEvent{
				ComputeType:        protos.SshTunnelComputeTypeServerless,
				ClientMode:         protos.SshTunnelClientModeSSH,
				HasBaseEnvironment: true,
			},
		},
		{
			name: "keeping detached processes records the request",
			opts: ClientOptions{ClusterID: "abc-123", KeepDetachedProcesses: true},
			want: protos.SshTunnelEvent{
				ComputeType:           protos.SshTunnelComputeTypeDedicated,
				ClientMode:            protos.SshTunnelClientModeSSH,
				KeepDetachedRequested: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildSshTunnelEvent(tt.opts, connectOutcome{
				isSuccess:         true,
				isReconnect:       true,
				serverStartTimeMs: 1500,
			})
			tt.want.IsSuccess = true
			tt.want.IsReconnect = true
			tt.want.ServerStartTimeMs = 1500
			tt.want.ErrorCategory = protos.SshTunnelErrorCategoryUnspecified
			assert.Equal(t, &tt.want, got)
		})
	}
}

func TestConnectOutcomeCategory(t *testing.T) {
	errFailed := errors.New("failed")

	tests := []struct {
		name    string
		outcome connectOutcome
		want    protos.SshTunnelErrorCategory
	}{
		{
			name:    "success reports no category",
			outcome: connectOutcome{isSuccess: true},
			want:    protos.SshTunnelErrorCategoryUnspecified,
		},
		{
			// A non-zero exit after the tunnel is up belongs to the ssh client, not the connection.
			name:    "error after a successful connection reports no category",
			outcome: connectOutcome{isSuccess: true, err: errFailed},
			want:    protos.SshTunnelErrorCategoryUnspecified,
		},
		{
			// A session end the proxy did attribute must survive isSuccess, or a mid-session
			// drop is indistinguishable from a clean exit.
			name: "attributed session end after a successful connection keeps its category",
			outcome: connectOutcome{
				isSuccess:     true,
				errorCategory: protos.SshTunnelErrorCategoryWebsocketDropped,
				err:           errFailed,
			},
			want: protos.SshTunnelErrorCategoryWebsocketDropped,
		},
		{
			name:    "attributed failure keeps its category",
			outcome: connectOutcome{errorCategory: protos.SshTunnelErrorCategoryIDECommandNotOnPath, err: errFailed},
			want:    protos.SshTunnelErrorCategoryIDECommandNotOnPath,
		},
		{
			name:    "unattributed failure falls back to UNKNOWN",
			outcome: connectOutcome{err: errFailed},
			want:    protos.SshTunnelErrorCategoryUnknown,
		},
		{
			name:    "cancellation reports USER_ABORTED",
			outcome: connectOutcome{err: fmt.Errorf("wrapped: %w", context.Canceled)},
			want:    protos.SshTunnelErrorCategoryUserAborted,
		},
		{
			// Ctrl-C surfaces as a cancellation from whichever step observed it first, so the
			// interruption must win over the category that step recorded.
			name: "cancellation wins over the category set at the failure site",
			outcome: connectOutcome{
				errorCategory: protos.SshTunnelErrorCategoryServerStartTimeout,
				err:           fmt.Errorf("wrapped: %w", context.Canceled),
			},
			want: protos.SshTunnelErrorCategoryUserAborted,
		},
		{
			// A step that shells out reports a killed child as *exec.ExitError, which does not
			// wrap context.Canceled, so the cancelled context is the only evidence left. Without
			// this branch a Ctrl-C during the extension install counts as a rejected install and
			// pollutes the bucket that is supposed to mean "the marketplace or a policy blocked
			// it" -- one of only two reachable under --auto-approve.
			name: "interruption wins over the category set at the failure site",
			outcome: connectOutcome{
				ctxErr:        context.Canceled,
				errorCategory: protos.SshTunnelErrorCategoryIDESSHExtensionInstallFailed,
				err:           errors.New("signal: killed"),
			},
			want: protos.SshTunnelErrorCategoryUserAborted,
		},
		{
			// Only a cancellation is the user giving up. No ancestor of the connect context
			// carries a deadline today, so this is unreachable; matching the cause rather than
			// testing ctxErr for non-nil keeps it that way if one is ever added.
			name: "an expired deadline is not a user abort",
			outcome: connectOutcome{
				ctxErr:        context.DeadlineExceeded,
				errorCategory: protos.SshTunnelErrorCategoryServerStartTimeout,
				err:           errFailed,
			},
			want: protos.SshTunnelErrorCategoryServerStartTimeout,
		},
		{
			// Interrupting an established tunnel is not a connection failure.
			name:    "interruption after a successful connection reports no category",
			outcome: connectOutcome{isSuccess: true, ctxErr: context.Canceled, err: errFailed},
			want:    protos.SshTunnelErrorCategoryUnspecified,
		},
		{
			// ...but a session end the proxy did attribute outranks the interruption, or a drop
			// that happened to coincide with the user giving up would be lost.
			name: "an attributed session end wins over an interruption",
			outcome: connectOutcome{
				isSuccess:     true,
				ctxErr:        context.Canceled,
				errorCategory: protos.SshTunnelErrorCategoryWebsocketDropped,
				err:           errFailed,
			},
			want: protos.SshTunnelErrorCategoryWebsocketDropped,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.outcome.category())
		})
	}
}

// The four Remote SSH extension outcomes were reported as one category until they were split,
// which left the largest IDE-mode failure bucket unattributable. Pin the mapping, including the
// wrapping, since CheckIDESSHExtension returns its sentinels wrapped in a message.
func TestSshExtensionErrorCategory(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want protos.SshTunnelErrorCategory
	}{
		{
			name: "list failure",
			err:  fmt.Errorf("%w in VS Code: %w", vscode.ErrSSHExtensionListFailed, errors.New("exit 4")),
			want: protos.SshTunnelErrorCategoryIDESSHExtensionListFailed,
		},
		{
			name: "install failure",
			err:  fmt.Errorf("%w: %w", vscode.ErrSSHExtensionInstallFailed, errors.New("exit 3")),
			want: protos.SshTunnelErrorCategoryIDESSHExtensionInstallFailed,
		},
		{
			name: "user declined the install",
			err:  fmt.Errorf("%w: install it with ...", vscode.ErrSSHExtensionInstallDeclined),
			want: protos.SshTunnelErrorCategoryIDESSHExtensionInstallDeclined,
		},
		{
			name: "no way to ask for consent",
			err:  fmt.Errorf("%w: install it with ...", vscode.ErrSSHExtensionInstallUnavailable),
			want: protos.SshTunnelErrorCategoryIDESSHExtensionInstallUnavailable,
		},
		{
			// Only reachable if a new failure path forgets its sentinel.
			name: "unsentinelled failure falls back to UNKNOWN",
			err:  errors.New("something else"),
			want: protos.SshTunnelErrorCategoryUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, sshExtensionErrorCategory(tt.err))
		})
	}
}

func TestProxySessionEndCategory(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want protos.SshTunnelErrorCategory
	}{
		{
			name: "clean finish is not attributed",
			err:  nil,
			want: "",
		},
		{
			name: "dropped websocket",
			err:  fmt.Errorf("wrapped: %w", proxy.ErrWebsocketDropped),
			want: protos.SshTunnelErrorCategoryWebsocketDropped,
		},
		{
			name: "handover failure",
			err:  fmt.Errorf("wrapped: %w", proxy.ErrHandoverFailed),
			want: protos.SshTunnelErrorCategoryHandoverFailed,
		},
		{
			name: "connect failure",
			err:  fmt.Errorf("wrapped: %w", proxy.ErrConnectFailed),
			want: protos.SshTunnelErrorCategoryWebsocketConnectFailed,
		},
		{
			// A drop landing during a handover surfaces from either the receiving loop or the
			// handover goroutine, so it must be counted as a drop either way.
			name: "a drop during a handover counts as a drop",
			err:  errors.Join(proxy.ErrHandoverFailed, proxy.ErrWebsocketDropped),
			want: protos.SshTunnelErrorCategoryWebsocketDropped,
		},
		{
			name: "an unrecognised error is left unattributed",
			err:  errors.New("something else"),
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, proxySessionEndCategory(tt.err))
		})
	}
}

func TestBuildSshTunnelEventReportsErrorCategory(t *testing.T) {
	got := buildSshTunnelEvent(ClientOptions{ConnectionName: "my-conn", IDE: "vscode"}, connectOutcome{
		errorCategory: protos.SshTunnelErrorCategoryIDECommandNotOnPath,
		err:           errors.New("failed"),
	})

	assert.False(t, got.IsSuccess)
	assert.Equal(t, protos.SshTunnelErrorCategoryIDECommandNotOnPath, got.ErrorCategory)
}

// A failed first attempt is the case the telemetry exists to measure, so assert
// the outcome fields reach the payload as an explicit false rather than being
// dropped as zero values.
func TestBuildSshTunnelEventReportsFailure(t *testing.T) {
	got := buildSshTunnelEvent(ClientOptions{ClusterID: "abc-123"}, connectOutcome{})

	assert.False(t, got.IsSuccess)

	b, err := json.Marshal(got)
	require.NoError(t, err)
	assert.Contains(t, string(b), `"is_success":false`)
}

func TestMetadataResponseError(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
		forbidden  bool
		message    string
	}{
		{
			name:       "feature disabled",
			statusCode: 403,
			body:       `"FEATURE_DISABLED: An administrator has disabled port forwarding to serverless compute for this workspace"`,
			forbidden:  true,
			message:    "the workspace refused the SSH tunnel connection: FEATURE_DISABLED: An administrator has disabled port forwarding to serverless compute for this workspace",
		},
		{
			name:       "forbidden with a body that is not a JSON string",
			statusCode: 403,
			body:       "Forbidden\n",
			forbidden:  true,
			message:    "the workspace refused the SSH tunnel connection: Forbidden",
		},
		{
			name:       "forbidden with an empty body",
			statusCode: 403,
			body:       "",
			forbidden:  true,
			message:    "the workspace refused the SSH tunnel connection: status code 403",
		},
		{
			// A stale metadata.json points at a terminated cluster; the caller must start a new server.
			name:       "terminated cluster is retried",
			statusCode: 400,
			body:       `"INVALID_STATE: Cluster 1014-135824-abcdefgh-v2n is in Terminated state but Spark driver is not healthy"`,
			forbidden:  false,
			message:    "server metadata error\nserver is not ok, status code 400",
		},
		{
			name:       "server still starting is retried",
			statusCode: 502,
			body:       "",
			forbidden:  false,
			message:    "server metadata error\nserver is not ok, status code 502",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := metadataResponseError(tc.statusCode, []byte(tc.body))
			assert.Equal(t, tc.forbidden, errors.Is(err, errDriverProxyForbidden))
			assert.Equal(t, !tc.forbidden, errors.Is(err, errServerMetadata))
			assert.EqualError(t, err, tc.message)
		})
	}
}

func TestCheckPortForwardingToServerless(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		body    string
		wantErr error
	}{
		{
			name:    "disabled",
			status:  http.StatusOK,
			body:    `{"effective_boolean_val": {"value": false}}`,
			wantErr: errPortForwardingDisabled,
		},
		{
			name:   "enabled",
			status: http.StatusOK,
			body:   `{"effective_boolean_val": {"value": true}}`,
		},
		{
			name:   "no effective value",
			status: http.StatusOK,
			body:   `{}`,
		},
		{
			// Production does not serve the setting through the public API yet.
			name:   "read fails",
			status: http.StatusBadRequest,
			body:   `{"error_code": "INVALID_PARAMETER_VALUE", "message": "Setting port_fwd_serverless_enabled is not enabled for public API"}`,
		},
		{
			// A retriable status is not retried, so a throttled service cannot hold up the connection.
			name:   "throttled",
			status: http.StatusTooManyRequests,
			body:   `{"error_code": "TOO_MANY_REQUESTS", "message": "slow down"}`,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var reads atomic.Int32
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/api/2.1/settings/"+portFwdServerlessSetting {
					w.WriteHeader(http.StatusNotFound)
					return
				}
				reads.Add(1)
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer srv.Close()

			ctx := cmdio.MockDiscard(t.Context())
			assert.Equal(t, tc.wantErr, checkPortForwardingToServerless(ctx, newProbeClient(t, srv.URL)))
			assert.Equal(t, int32(1), reads.Load())
		})
	}
}
