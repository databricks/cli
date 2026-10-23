package dstate_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"testing/synctest"
	"time"

	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/direct/dstate"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/databricks-sdk-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type modeTransport struct {
	handler http.HandlerFunc
}

func (m modeTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	w := httptest.NewRecorder()
	if r.URL.Path == "/.well-known/databricks-config" || m.handler == nil {
		w.WriteHeader(http.StatusNotFound)
	} else {
		m.handler.ServeHTTP(w, r)
	}
	if err := r.Context().Err(); err != nil {
		return nil, err
	}
	response := w.Result()
	response.Request = r
	return response, nil
}

// modeContext supplies an in-memory DMS endpoint without resolving workspace authentication.
func modeContext(t *testing.T, handler http.HandlerFunc) context.Context {
	t.Helper()
	client, err := databricks.NewWorkspaceClient(&databricks.Config{
		Host: "https://state.test", Token: "test-token", AuthType: "pat",
		HTTPTransport: modeTransport{handler: handler},
		// Keep SDK inactivity timeouts outside the drain budgets exercised here.
		HTTPTimeoutSeconds: 600,
	})
	require.NoError(t, err)
	return cmdctx.SetWorkspaceClient(t.Context(), client)
}

func TestGetSerialWithDivergingCounters(t *testing.T) {
	for _, tc := range []struct {
		mode         dstate.Mode
		versionAhead int
		serialAhead  int
	}{
		{dstate.ReadingFile, 4, 7},
		{dstate.WritingFile, 4, 7},
		{dstate.WritingDMS, 7, 4},
		{dstate.WritingFileWithHistory, 7, 7},
		{dstate.WritingFileWithHistoryEnforced, 7, 7},
	} {
		t.Run(fmt.Sprint(tc.mode), func(t *testing.T) {
			ctx := t.Context()
			var db dstate.DeploymentState
			path := filepath.Join(t.TempDir(), "resources.json")
			require.NoError(t, db.Open(ctx, path, tc.mode, false, dstate.OpenDmsArgs{}))
			db.Data.Serial, db.VersionID = 4, 7
			assert.Equal(t, tc.versionAhead, db.GetSerial(), "DMS version ahead of file serial")
			db.Data.Serial, db.VersionID = 7, 4
			assert.Equal(t, tc.serialAhead, db.GetSerial(), "file serial ahead of DMS version")
			_, err := db.Finalize(ctx)
			require.NoError(t, err)
		})
	}
}

func TestFileModesPersistStateAndHistory(t *testing.T) {
	for _, mode := range []dstate.Mode{dstate.WritingFile, dstate.WritingFileWithHistory, dstate.WritingFileWithHistoryEnforced} {
		t.Run(fmt.Sprint(mode), func(t *testing.T) {
			updates := make(map[string]map[string]any)
			ctx := modeContext(t, func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodPost {
					assert.Equal(t, "/api/2.0/bundle/deployments/dep-1/versions/1/complete", r.URL.Path)
					_, _ = w.Write([]byte(`{}`))
					return
				}
				assert.Equal(t, http.MethodPatch, r.Method)
				var body map[string]any
				if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&body)) {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				updates[filepath.Base(r.URL.Path)] = body
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"sequence_id":1}`))
			})
			path := filepath.Join(t.TempDir(), "resources.json")
			var db dstate.DeploymentState
			require.NoError(t, db.Open(ctx, path, mode, true, dstate.OpenDmsArgs{}))
			if mode != dstate.WritingFile {
				db.InitializeOperationBuffer(ctx, "dep-1", 1, 2)
			}
			dependsOn := []deployplan.DependsOnEntry{{Node: "resources.pipelines.upstream", Label: "upstream"}}
			require.NoError(t, db.SaveState(ctx, "resources.jobs.kept", "keep-id", map[string]string{"name": "kept"}, dependsOn))
			require.NoError(t, db.SaveState(ctx, "resources.jobs.deleted", "delete-id", map[string]string{"name": "deleted"}, nil))
			require.NoError(t, db.DeleteState(ctx, "resources.jobs.deleted", false))
			exported, err := db.Finalize(ctx)
			require.NoError(t, err)
			assert.Len(t, exported, 1)
			assert.Equal(t, "keep-id", exported["resources.jobs.kept"].ID)
			assert.NoFileExists(t, path+".wal")
			completed, err := db.CompleteVersion(ctx, true)
			require.NoError(t, err)
			assert.Equal(t, mode != dstate.WritingFile, completed)
			completed, err = db.CompleteVersion(ctx, true)
			require.NoError(t, err)
			assert.False(t, completed)

			var reopened dstate.DeploymentState
			require.NoError(t, reopened.Open(t.Context(), path, dstate.ReadingFile, false, dstate.OpenDmsArgs{}))
			assert.Equal(t, 1, reopened.GetSerial())
			assert.Len(t, reopened.Data.State, 1)
			assert.Equal(t, dependsOn, reopened.Data.State["resources.jobs.kept"].DependsOn)
			_, enforced := reopened.Data.Features[dstate.FeatureDeploymentHistoryBestEffort]
			assert.Equal(t, mode == dstate.WritingFileWithHistoryEnforced, enforced)
			_, err = reopened.Finalize(t.Context())
			require.NoError(t, err)

			if mode == dstate.WritingFile {
				assert.Empty(t, updates)
			} else {
				require.Len(t, updates, 2)
				assert.JSONEq(t, `{"state":{"name":"kept"},"depends_on":[{"node":"resources.pipelines.upstream","label":"upstream"}]}`, updates["jobs.kept"]["state"].(string))
				assert.NotContains(t, updates["jobs.deleted"], "state")
			}
		})
	}
}

func TestHistoryErrorsFailOpen(t *testing.T) {
	for _, mode := range []dstate.Mode{dstate.WritingDMS, dstate.WritingFileWithHistory, dstate.WritingFileWithHistoryEnforced} {
		t.Run(fmt.Sprint(mode), func(t *testing.T) {
			goodUploads := 0
			ctx := modeContext(t, func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodPatch && filepath.Base(r.URL.Path) == "jobs.good" {
					goodUploads++
					_, _ = w.Write([]byte(`{"sequence_id":1}`))
					return
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{"error_code":"INVALID_PARAMETER_VALUE","message":"history unavailable"}`))
			})
			path := filepath.Join(t.TempDir(), "resources.json")
			var db dstate.DeploymentState
			require.NoError(t, db.Open(ctx, path, mode, false, dstate.OpenDmsArgs{}))
			db.InitializeOperationBuffer(ctx, "dep-1", 1, 2)
			require.NoError(t, db.SaveState(ctx, "resources.jobs.job", "id-1", map[string]string{"name": "job"}, nil))
			require.NoError(t, db.SaveState(ctx, "resources.jobs.good", "id-2", struct{}{}, nil))
			_, err := db.Finalize(ctx)
			assert.Equal(t, 1, goodUploads, "a bad upload must not stop the next resource")
			if mode == dstate.WritingDMS {
				require.ErrorContains(t, err, "history unavailable")
				assert.Error(t, db.RecordingError())
			} else {
				require.NoError(t, err)
				assert.NoError(t, db.RecordingError())
				var reopened dstate.DeploymentState
				require.NoError(t, reopened.Open(t.Context(), path, dstate.ReadingFile, false, dstate.OpenDmsArgs{}))
				assert.Equal(t, "id-1", reopened.GetResourceID("resources.jobs.job"))
				_, err = reopened.Finalize(t.Context())
				require.NoError(t, err)
			}
			_, err = db.CompleteVersion(ctx, true)
			if mode == dstate.WritingDMS {
				assert.ErrorContains(t, err, "history unavailable")
			} else {
				assert.NoError(t, err)
			}
			completed, err := db.CompleteVersion(ctx, true)
			require.NoError(t, err)
			assert.False(t, completed)
		})
	}
}

func TestEnforcedHistorySurvivesWALRecovery(t *testing.T) {
	ctx := t.Context()
	path := filepath.Join(t.TempDir(), "resources.json")
	var db dstate.DeploymentState
	require.NoError(t, db.Open(ctx, path, dstate.WritingFileWithHistoryEnforced, false, dstate.OpenDmsArgs{}))
	wal, err := os.ReadFile(path + ".wal")
	require.NoError(t, err)
	_, err = db.Finalize(ctx)
	require.NoError(t, err)

	// Recover an interrupted first deployment: its WAL contains only the feature marker.
	recoveredPath := filepath.Join(t.TempDir(), "resources.json")
	require.NoError(t, os.WriteFile(recoveredPath+".wal", wal, 0o600))
	var recovered dstate.DeploymentState
	require.NoError(t, recovered.Open(ctx, recoveredPath, dstate.ReadingFile, true, dstate.OpenDmsArgs{}))
	assert.Contains(t, recovered.StateFeatures(), dstate.FeatureDeploymentHistoryBestEffort)
	assert.Equal(t, 1, recovered.GetSerial())
	assert.FileExists(t, recoveredPath)
	assert.NoFileExists(t, recoveredPath+".wal")
	_, err = recovered.Finalize(ctx)
	require.NoError(t, err)
}

func TestHistoryQueueOnlyTimesOutAtDrain(t *testing.T) {
	for _, mode := range []dstate.Mode{dstate.WritingFileWithHistory, dstate.WritingFileWithHistoryEnforced} {
		t.Run(fmt.Sprint(mode), func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				started := make(chan struct{})
				finished := make(chan struct{})
				ctx := modeContext(t, func(w http.ResponseWriter, r *http.Request) {
					_, _ = io.Copy(io.Discard, r.Body)
					close(started)
					<-r.Context().Done()
					close(finished)
				})
				path := filepath.Join(t.TempDir(), "resources.json")
				var db dstate.DeploymentState
				require.NoError(t, db.Open(ctx, path, mode, false, dstate.OpenDmsArgs{}))
				const operationCount = 32
				db.InitializeOperationBuffer(ctx, "dep-1", 1, operationCount)
				start := time.Now()
				for i := range operationCount {
					key := fmt.Sprintf("resources.jobs.job_%d", i)
					require.NoError(t, db.SaveState(ctx, key, strconv.Itoa(i), struct{}{}, nil))
				}
				assert.Zero(t, time.Since(start))
				<-started
				time.Sleep(time.Minute + time.Second)
				select {
				case <-finished:
					t.Fatal("recording stopped before drain")
				default:
				}
				drainStart := time.Now()
				exported, err := db.Finalize(ctx)
				require.NoError(t, err)
				assert.Equal(t, 160*time.Second, time.Since(drainStart))
				assert.Len(t, exported, operationCount)
				assert.FileExists(t, path)
				assert.NoFileExists(t, path+".wal")
				<-finished
				assert.Equal(t, 160*time.Second, time.Since(drainStart))
			})
		})
	}
}
