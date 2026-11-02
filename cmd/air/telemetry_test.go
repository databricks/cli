package aircmd

import (
	"context"
	"encoding/json"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/telemetry"
	"github.com/databricks/cli/libs/telemetry/protos"
	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func captureAirTelemetry(t *testing.T, server *testserver.Server, w *databricks.WorkspaceClient) (context.Context, *[]protos.FrontendLog) {
	t.Helper()
	var events []protos.FrontendLog
	server.Handle("POST", "/telemetry-ext", func(req testserver.Request) any {
		var body telemetry.RequestBody
		require.NoError(t, json.Unmarshal(req.Body, &body))
		for _, raw := range body.ProtoLogs {
			var event protos.FrontendLog
			require.NoError(t, json.Unmarshal([]byte(raw), &event))
			events = append(events, event)
		}
		return telemetry.ResponseBody{NumProtoSuccess: int64(len(body.ProtoLogs))}
	})
	ctx := telemetry.WithNewLogger(cmdio.MockDiscard(t.Context()))
	return cmdctx.SetConfigUsed(ctx, w.Config), &events
}

func TestAirRunTelemetry(t *testing.T) {
	for _, tc := range []struct {
		name       string
		snapshot   bool
		fail       bool
		failUpload bool
	}{
		{name: "without snapshot"},
		{name: "uploaded snapshot", snapshot: true},
		{name: "failed submit", snapshot: true, fail: true},
		{name: "failed upload", snapshot: true, failUpload: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := testserver.New(t)
			t.Cleanup(server.Close)
			server.Handle("POST", "/api/2.2/jobs/runs/submit", func(req testserver.Request) any {
				if tc.fail {
					return testserver.Response{StatusCode: http.StatusBadRequest, Body: map[string]string{
						"error_code": "INVALID_PARAMETER_VALUE", "message": "submission rejected",
					}}
				}
				return jobs.SubmitRunResponse{RunId: 555}
			})
			var uploadedSize int64
			server.Handle("POST", "/api/2.0/workspace-files/import-file/{path...}", func(req testserver.Request) any {
				p := req.Vars["path"]
				if strings.HasSuffix(p, ".tar.gz") {
					uploadedSize = int64(len(req.Body))
					time.Sleep(20 * time.Millisecond)
					if tc.failUpload {
						return testserver.Response{StatusCode: http.StatusForbidden, Body: map[string]string{
							"error_code": "PERMISSION_DENIED", "message": "upload rejected",
						}}
					}
				}
				return req.Workspace.WorkspaceFilesImportFile(p, req.Body, req.URL.Query().Get("overwrite") == "true")
			})
			stubValidateConfig(server)
			w, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "token"})
			require.NoError(t, err)
			ctx, events := captureAirTelemetry(t, server, w)
			testserver.AddDefaultHandlers(server)

			configYAML := minimalConfig
			if tc.snapshot {
				repo := filepath.Join(t.TempDir(), "src")
				writeRepoFile(t, repo, "train.py", "print('private source')")
				configYAML += "\ncode_source:\n  type: snapshot\n  snapshot:\n    root_path: " + filepath.ToSlash(repo) + "\n"
			}
			cfgPath := writeConfigFile(t, "run.yaml", configYAML)
			cfg, err := loadRunConfig(cfgPath)
			require.NoError(t, err)
			_, _, err = submitWorkload(ctx, w, cfg, cfgPath, "idem", false)
			if tc.fail || tc.failUpload {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			require.NoError(t, telemetry.Upload(ctx, protos.ExecutionContext{Command: "air_run"}))
			require.Len(t, *events, 1)
			event := (*events)[0].Entry.DatabricksCliLog.AirRunEvent
			require.NotNil(t, event)
			assert.Equal(t, !tc.fail && !tc.failUpload, event.SubmittedSuccessfully)
			assert.Equal(t, tc.snapshot, event.HasCodeSnapshot)
			assert.Equal(t, string(gpuType1xH100), event.GPUType)
			assert.Equal(t, 1, event.NumGPUs)
			assert.Equal(t, 1, event.NumNodes)
			assert.GreaterOrEqual(t, event.SubmitLatencyMs, int64(0))
			if tc.fail || tc.failUpload {
				assert.Empty(t, event.JobRunID)
			} else {
				assert.Equal(t, "555", event.JobRunID)
			}
			if tc.snapshot {
				require.NotNil(t, event.CodeSourceSizeBytes)
				assert.Positive(t, uploadedSize)
				assert.Equal(t, uploadedSize, *event.CodeSourceSizeBytes)
				require.NotNil(t, event.CodeSourceUsesGit)
				assert.False(t, *event.CodeSourceUsesGit)
				assert.Equal(t, protos.AirPackagingModePlainTar, event.CodeSourcePackagingMode)
				require.NotNil(t, event.CodeSourcePackagingDurationMs)
				assert.GreaterOrEqual(t, *event.CodeSourcePackagingDurationMs, int64(0))
				require.NotNil(t, event.CodeSourceUploadDurationMs)
				assert.GreaterOrEqual(t, *event.CodeSourceUploadDurationMs, int64(20))
			} else {
				assert.Nil(t, event.CodeSourceSizeBytes)
				assert.Nil(t, event.CodeSourceUsesGit)
				assert.Empty(t, event.CodeSourcePackagingMode)
				assert.Nil(t, event.CodeSourcePackagingDurationMs)
				assert.Nil(t, event.CodeSourceUploadDurationMs)
			}
			raw, err := json.Marshal(event)
			require.NoError(t, err)
			assert.NotContains(t, string(raw), cfg.ExperimentName)
			assert.NotContains(t, string(raw), "private source")
			assert.NotContains(t, string(raw), "root_path")
		})
	}
}

func TestAirRunTelemetryConfig(t *testing.T) {
	server := testserver.New(t)
	t.Cleanup(server.Close)
	w, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "token"})
	require.NoError(t, err)
	ctx, events := captureAirTelemetry(t, server, w)
	cfgPath := writeConfigFile(t, "run.yaml", minimalConfig)
	cfg, err := loadRunConfig(cfgPath)
	require.NoError(t, err)
	cfg.Compute.AcceleratorType = string(gpuType8xH100)
	cfg.Compute.NumAccelerators = 16
	cfg.MaxRetries = new(0)
	cfg.TimeoutMinutes = new(5)
	cfg.Parameters = map[string]any{"private-parameter": "private-value"}
	cfg.Environment = &environmentConfig{UnityCatalogImage: "private.schema.image:tag"}
	logRunEvent(ctx, cfg, snapshotResult{
		PackagingMode:       new(modeGitArchive),
		PackagingDurationMs: new(int64(12)),
		UploadDurationMs:    new(int64(23)),
	}, 123, 42*time.Millisecond, nil)
	require.NoError(t, telemetry.Upload(ctx, protos.ExecutionContext{}))
	require.Len(t, *events, 1)
	event := (*events)[0].Entry.DatabricksCliLog.AirRunEvent
	assert.Equal(t, 2, event.NumNodes)
	assert.Equal(t, 0, event.MaxRetries)
	assert.True(t, event.HasDockerImage)
	assert.True(t, event.HasParameters)
	assert.True(t, event.HasTimeout)
	assert.Equal(t, int64(42), event.SubmitLatencyMs)
	assert.Equal(t, protos.AirPackagingModeGitArchive, event.CodeSourcePackagingMode)
	assert.Equal(t, new(int64(12)), event.CodeSourcePackagingDurationMs)
	assert.Equal(t, new(int64(23)), event.CodeSourceUploadDurationMs)
}

func TestSnapshotPackagingFailureMeasurements(t *testing.T) {
	server := testserver.New(t)
	t.Cleanup(server.Close)
	testserver.AddDefaultHandlers(server)
	w, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "token"})
	require.NoError(t, err)
	// The commit is no longer available when git archive attempts to package it.
	result, err := uploadSnapshotTarball(t.Context(), w, newTestRepo(t), snapshotPlan{
		mode: modeGitArchive, commitSHA: "missing-commit", isGitRepo: true,
	}, testSnapshotArtifactPath)
	require.Error(t, err)
	assert.Equal(t, new(modeGitArchive), result.PackagingMode)
	require.NotNil(t, result.PackagingDurationMs)
	assert.GreaterOrEqual(t, *result.PackagingDurationMs, int64(0))
	assert.Nil(t, result.UploadDurationMs)
	assert.Nil(t, result.SizeBytes)
	assert.Empty(t, result.CodeSourcePath)
}

func TestAirRunTelemetryContainersHaveDockerImage(t *testing.T) {
	server := testserver.New(t)
	t.Cleanup(server.Close)
	w, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "token"})
	require.NoError(t, err)
	ctx, events := captureAirTelemetry(t, server, w)
	cfg, err := loadRunConfig(writeConfigFile(t, "run.yaml", minimalConfig))
	require.NoError(t, err)
	cfg.Command = nil
	cfg.Containers = []containerConfig{{Name: "trainer", UnityCatalogImage: "private.schema.image:tag"}}
	logRunEvent(ctx, cfg, snapshotResult{}, 123, time.Millisecond, nil)
	require.NoError(t, telemetry.Upload(ctx, protos.ExecutionContext{}))
	require.Len(t, *events, 1)
	assert.True(t, (*events)[0].Entry.DatabricksCliLog.AirRunEvent.HasDockerImage)
}
