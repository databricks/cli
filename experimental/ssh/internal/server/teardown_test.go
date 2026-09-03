package server

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/telemetry"
	"github.com/databricks/cli/libs/telemetry/protos"
	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testNotebookPid = 100
	testServerPid   = 200
)

// procWithDetachedWork is the teardown shape the warning is about: the notebook anchors the
// server, and one process has been adopted by the notebook with a process group of its own.
func procWithDetachedWork(t *testing.T) string {
	return fakeProc(t, map[int]fakeProcess{
		testNotebookPid: {comm: "python", ppid: 1, pgrp: testNotebookPid, state: "S"},
		testServerPid:   {comm: "databricks", ppid: testNotebookPid, pgrp: testServerPid, state: "S"},
		400:             {comm: "tmux: server", ppid: testNotebookPid, pgrp: 400, state: "S"},
	})
}

func TestReportDetachedDescendantsWarning(t *testing.T) {
	t.Run("warns and names the flag when the work is about to be swept", func(t *testing.T) {
		ctx, logs := captureWarnLogs(t.Context())
		reportDetachedDescendants(ctx, ServerOptions{}, procWithDetachedWork(t), testServerPid)

		assert.Contains(t, logs.String(), "1 detached process(es) still running (pids 400)")
		assert.Contains(t, logs.String(), "--keep-detached-for")
	})

	t.Run("stays quiet when the run is held open for them", func(t *testing.T) {
		ctx, logs := captureWarnLogs(t.Context())
		opts := ServerOptions{KeepDetachedFor: time.Hour}
		reportDetachedDescendants(ctx, opts, procWithDetachedWork(t), testServerPid)

		assert.Empty(t, logs.String())
	})

	// The flag is rejected for serverless, so pointing at it there would be misleading.
	t.Run("stays quiet on serverless", func(t *testing.T) {
		ctx, logs := captureWarnLogs(t.Context())
		reportDetachedDescendants(ctx, ServerOptions{Serverless: true}, procWithDetachedWork(t), testServerPid)

		assert.Empty(t, logs.String())
	})

	t.Run("stays quiet when nothing was left behind", func(t *testing.T) {
		root := fakeProc(t, map[int]fakeProcess{
			testNotebookPid: {comm: "python", ppid: 1, pgrp: testNotebookPid, state: "S"},
			testServerPid:   {comm: "databricks", ppid: testNotebookPid, pgrp: testServerPid, state: "S"},
		})
		ctx, logs := captureWarnLogs(t.Context())
		reportDetachedDescendants(ctx, ServerOptions{}, root, testServerPid)

		assert.Empty(t, logs.String())
	})

	t.Run("does not warn when the process tree cannot be read", func(t *testing.T) {
		ctx, logs := captureWarnLogs(t.Context())
		reportDetachedDescendants(ctx, ServerOptions{}, t.TempDir(), testServerPid)

		assert.Empty(t, logs.String())
	})
}

// The teardown event is the only measurement of how often the tunnel is about to destroy
// detached work, so this pins that it reaches the wire with the fields a query needs.
func TestReportDetachedDescendantsTelemetry(t *testing.T) {
	tests := []struct {
		name string
		opts ServerOptions
		root func(t *testing.T) string
		want protos.SshTunnelTeardownEvent
	}{
		{
			name: "detached work left behind on a dedicated cluster",
			opts: ServerOptions{},
			root: procWithDetachedWork,
			want: protos.SshTunnelTeardownEvent{
				ComputeType:                      protos.SshTunnelComputeTypeDedicated,
				HadDetachedDescendantsAtTeardown: true,
			},
		},
		{
			name: "the run was held open for it",
			opts: ServerOptions{KeepDetachedFor: 2 * time.Hour},
			root: procWithDetachedWork,
			want: protos.SshTunnelTeardownEvent{
				ComputeType:                      protos.SshTunnelComputeTypeDedicated,
				KeepDetachedRequested:            true,
				HadDetachedDescendantsAtTeardown: true,
			},
		},
		{
			name: "nothing detached, on serverless",
			opts: ServerOptions{Serverless: true},
			root: func(t *testing.T) string {
				return fakeProc(t, map[int]fakeProcess{
					testNotebookPid: {comm: "python", ppid: 1, pgrp: testNotebookPid, state: "S"},
					testServerPid:   {comm: "databricks", ppid: testNotebookPid, pgrp: testServerPid, state: "S"},
				})
			},
			want: protos.SshTunnelTeardownEvent{ComputeType: protos.SshTunnelComputeTypeServerless},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := testserver.New(t)
			t.Cleanup(server.Close)

			var body telemetry.RequestBody
			server.Handle("POST", "/telemetry-ext", func(req testserver.Request) any {
				require.NoError(t, json.Unmarshal(req.Body, &body))
				return telemetry.ResponseBody{NumProtoSuccess: 1}
			})

			ctx := telemetry.WithNewLogger(t.Context())
			ctx = cmdctx.SetConfigUsed(ctx, &config.Config{Host: server.URL, Token: "token"})

			reportDetachedDescendants(ctx, tt.opts, tt.root(t), testServerPid)
			require.NoError(t, telemetry.Upload(ctx, protos.ExecutionContext{}))

			require.Len(t, body.ProtoLogs, 1)
			var logged protos.FrontendLog
			require.NoError(t, json.Unmarshal([]byte(body.ProtoLogs[0]), &logged))
			require.NotNil(t, logged.Entry.DatabricksCliLog.SshTunnelTeardownEvent)
			assert.Equal(t, tt.want, *logged.Entry.DatabricksCliLog.SshTunnelTeardownEvent)
			// The connect event stays untouched, so is_success queries keep counting connections.
			assert.Nil(t, logged.Entry.DatabricksCliLog.SshTunnelEvent)
		})
	}
}
