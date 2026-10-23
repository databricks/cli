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
	"sync"
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
			ctx := modeContext(t, nil)
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

func TestModeAssertions(t *testing.T) {
	for _, tc := range []struct {
		mode     dstate.Mode
		writable bool
		dms      bool
		history  bool
	}{
		{dstate.ReadingFile, false, false, false},
		{dstate.WritingFile, true, false, false},
		{dstate.WritingDMS, true, true, true},
		{dstate.WritingFileWithHistory, true, false, true},
		{dstate.WritingFileWithHistoryEnforced, true, false, true},
	} {
		t.Run(fmt.Sprint(tc.mode), func(t *testing.T) {
			ctx := modeContext(t, func(w http.ResponseWriter, r *http.Request) {
				t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			})
			path := filepath.Join(t.TempDir(), "resources.json")
			var db dstate.DeploymentState
			require.NoError(t, db.Open(ctx, path, tc.mode, false, dstate.OpenDmsArgs{}))
			assert.Equal(t, tc.dms, db.IsDeploymentMetadataService())
			assert.Equal(t, tc.history, db.RecordsHistory())
			assert.NotPanics(t, db.AssertOpenedForReadOrWrite)
			if tc.writable {
				assert.NotPanics(t, db.AssertOpenedForWrite)
				if tc.dms {
					assert.NotPanics(t, db.AssertOpenedForRead)
				} else {
					assert.Panics(t, db.AssertOpenedForRead)
				}
			} else {
				assert.NotPanics(t, db.AssertOpenedForRead)
				assert.Panics(t, db.AssertOpenedForWrite)
			}
			if tc.writable && !tc.dms {
				assert.FileExists(t, path+".wal")
			} else {
				assert.NoFileExists(t, path+".wal")
			}
			_, err := db.Finalize(ctx)
			require.NoError(t, err)
			assert.Panics(t, db.AssertOpenedForReadOrWrite)
			assert.NoFileExists(t, path+".wal")
		})
	}
}

func TestFileModesPersistStateAndHistory(t *testing.T) {
	for _, mode := range []dstate.Mode{dstate.WritingFile, dstate.WritingFileWithHistory, dstate.WritingFileWithHistoryEnforced} {
		t.Run(fmt.Sprint(mode), func(t *testing.T) {
			var mu sync.Mutex
			updates := make(map[string]map[string]any)
			ctx := modeContext(t, func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodPatch, r.Method)
				var body map[string]any
				if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&body)) {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				mu.Lock()
				updates[filepath.Base(r.URL.Path)] = body
				mu.Unlock()
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

			var reopened dstate.DeploymentState
			require.NoError(t, reopened.Open(t.Context(), path, dstate.ReadingFile, false, dstate.OpenDmsArgs{}))
			assert.Equal(t, 1, reopened.GetSerial())
			assert.Len(t, reopened.Data.State, 1)
			assert.Equal(t, dependsOn, reopened.Data.State["resources.jobs.kept"].DependsOn)
			_, enforced := reopened.Data.Features[dstate.FeatureDeploymentHistoryBestEffort]
			assert.Equal(t, mode == dstate.WritingFileWithHistoryEnforced, enforced)
			_, err = reopened.Finalize(t.Context())
			require.NoError(t, err)

			mu.Lock()
			defer mu.Unlock()
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

func TestDMSReadsAndWritesWithoutWAL(t *testing.T) {
	ctx := modeContext(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodGet {
			_, _ = w.Write([]byte(`{"resources":[{"resource_key":"jobs.remote","resource_id":"remote-id","state":"{\"state\":{\"name\":\"remote\"}}"}]}`))
		} else {
			_, _ = w.Write([]byte(`{"sequence_id":1}`))
		}
	})
	path := filepath.Join(t.TempDir(), "resources.json")
	var db dstate.DeploymentState
	require.NoError(t, db.Open(ctx, path, dstate.WritingDMS, false, dstate.OpenDmsArgs{DeploymentID: "dep-1", LastVersionID: 7}))
	assert.Equal(t, "remote-id", db.GetResourceID("resources.jobs.remote"))
	assert.Equal(t, 7, db.GetSerial())
	require.NoError(t, db.UpgradeToWrite())
	assert.NoFileExists(t, path+".wal")
	db.InitializeOperationBuffer(ctx, "dep-1", 8, 2)
	require.NoError(t, db.SaveState(ctx, "resources.jobs.new", "new-id", map[string]string{"name": "new"}, nil))
	require.NoError(t, db.DeleteState(ctx, "resources.jobs.remote", false))
	exported, err := db.Finalize(ctx)
	require.NoError(t, err)
	assert.Len(t, exported, 1)
	assert.Equal(t, "new-id", exported["resources.jobs.new"].ID)
	assert.NoFileExists(t, path+".wal")
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	var marker dstate.Database
	require.NoError(t, json.Unmarshal(data, &marker))
	assert.Empty(t, marker.State)
	assert.Equal(t, 8, marker.Serial)
	assert.Contains(t, marker.Features, dstate.FeatureDeploymentHistory)
}

func TestUpgradeToWrite(t *testing.T) {
	for _, tc := range []struct {
		name       string
		mode       dstate.Mode
		bestEffort bool
		wantErr    string
	}{
		{"file", dstate.ReadingFile, false, ""},
		{"DMS", dstate.WritingDMS, false, ""},
		{"history", dstate.WritingFileWithHistory, false, ""},
		{"enforced history", dstate.WritingFileWithHistoryEnforced, false, ""},
		{"file already writable", dstate.WritingFile, false, "already open for write"},
		{"enforcement retained", dstate.WritingFileWithHistory, true, ""},
		{"file upgrade with history marker", dstate.ReadingFile, true, ""},
		{"file open with history marker", dstate.WritingFile, true, "already open for write"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := modeContext(t, nil)
			path := filepath.Join(t.TempDir(), "resources.json")
			if tc.bestEffort {
				require.NoError(t, os.WriteFile(path, []byte(`{"state_version":3,"features":{"deployment_history_best_effort":{}},"state":{}}`), 0o600))
			}
			var db dstate.DeploymentState
			require.NoError(t, db.Open(ctx, path, tc.mode, false, dstate.OpenDmsArgs{}))
			err := db.UpgradeToWrite()
			if tc.wantErr != "" {
				require.ErrorContains(t, err, tc.wantErr)
			} else {
				require.NoError(t, err)
				assert.NotPanics(t, db.AssertOpenedForWrite)
			}
			_, err = db.Finalize(ctx)
			require.NoError(t, err)
			if tc.bestEffort {
				data, err := os.ReadFile(path)
				require.NoError(t, err)
				var persisted dstate.Database
				require.NoError(t, json.Unmarshal(data, &persisted))
				assert.Contains(t, persisted.Features, dstate.FeatureDeploymentHistoryBestEffort)
			}
		})
	}
}

func TestDMSRecordingContinuesAfterUploadError(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		firstRequest := make(chan struct{})
		updates := 0
		ctx := modeContext(t, func(w http.ResponseWriter, r *http.Request) {
			var body struct {
				SequenceID int64  `json:"sequence_id"`
				State      string `json:"state"`
			}
			assert.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			assert.Zero(t, body.SequenceID, "a rejected update earns no sequence ID")
			updates++
			if updates == 1 {
				close(firstRequest)
				http.Error(w, `{"error_code":"INVALID_PARAMETER_VALUE","message":"first update failed"}`, http.StatusBadRequest)
				return
			}
			assert.JSONEq(t, `{"state":{"name":"second"}}`, body.State)
			_, _ = w.Write([]byte(`{"sequence_id":1}`))
		})
		var db dstate.DeploymentState
		path := filepath.Join(t.TempDir(), "resources.json")
		require.NoError(t, db.Open(ctx, path, dstate.WritingDMS, false, dstate.OpenDmsArgs{}))
		db.InitializeOperationBuffer(ctx, "dep-1", 1, 1)
		require.NoError(t, db.SaveState(ctx, "resources.jobs.job", "id-1", map[string]string{"name": "first"}, nil))
		<-firstRequest
		synctest.Wait()
		require.ErrorContains(t, db.RecordingError(), "first update failed")
		require.NoError(t, db.SaveState(ctx, "resources.jobs.job", "id-1", map[string]string{"name": "second"}, nil))
		_, err := db.Finalize(ctx)
		require.ErrorContains(t, err, "first update failed")
		assert.Equal(t, 2, updates)
	})
}

func TestHistoryErrorsFailOpen(t *testing.T) {
	for _, mode := range []dstate.Mode{dstate.WritingDMS, dstate.WritingFileWithHistory, dstate.WritingFileWithHistoryEnforced} {
		t.Run(fmt.Sprint(mode), func(t *testing.T) {
			ctx := modeContext(t, func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{"error_code":"INVALID_PARAMETER_VALUE","message":"history unavailable"}`))
			})
			path := filepath.Join(t.TempDir(), "resources.json")
			var db dstate.DeploymentState
			require.NoError(t, db.Open(ctx, path, mode, false, dstate.OpenDmsArgs{}))
			db.InitializeOperationBuffer(ctx, "dep-1", 1, 1)
			require.NoError(t, db.SaveState(ctx, "resources.jobs.job", "id-1", map[string]string{"name": "job"}, nil))
			_, err := db.Finalize(ctx)
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

func TestHistoryDrainTimeoutPreservesFileState(t *testing.T) {
	for _, mode := range []dstate.Mode{dstate.WritingFileWithHistory, dstate.WritingFileWithHistoryEnforced} {
		for _, tc := range []struct {
			name           string
			operationCount int
			callerTimeout  time.Duration
			wantDuration   time.Duration
		}{
			{name: "one operation", operationCount: 1, wantDuration: 5 * time.Second},
			{name: "all staged operations", operationCount: 4, wantDuration: 20 * time.Second},
			{name: "shorter caller deadline", operationCount: 4, callerTimeout: 20 * time.Millisecond, wantDuration: 20 * time.Millisecond},
		} {
			t.Run(fmt.Sprintf("%d/%s", mode, tc.name), func(t *testing.T) {
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
					db.InitializeOperationBuffer(ctx, "dep-1", 1, tc.operationCount)
					// One buffered update must still get the budget for all staged operations.
					require.NoError(t, db.SaveState(ctx, "resources.jobs.job", "id-1", struct{}{}, nil))
					<-started
					drainCtx := ctx
					if tc.callerTimeout > 0 {
						var cancel context.CancelFunc
						drainCtx, cancel = context.WithTimeout(ctx, tc.callerTimeout)
						defer cancel()
					}
					start := time.Now()
					_, err := db.Finalize(drainCtx)
					require.NoError(t, err)
					assert.Equal(t, tc.wantDuration, time.Since(start))
					assert.FileExists(t, path)
					assert.NoFileExists(t, path+".wal")
					<-finished
				})
			})
		}
	}
}

func TestCompleteVersionHonorsCallerDeadline(t *testing.T) {
	for _, mode := range []dstate.Mode{dstate.WritingDMS, dstate.WritingFileWithHistory, dstate.WritingFileWithHistoryEnforced} {
		t.Run(fmt.Sprint(mode), func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				ctx := modeContext(t, func(w http.ResponseWriter, r *http.Request) {
					if r.Method == http.MethodPost {
						_, _ = io.Copy(io.Discard, r.Body)
						<-r.Context().Done()
					} else {
						w.Header().Set("Content-Type", "application/json")
						_, _ = w.Write([]byte(`{"sequence_id":1}`))
					}
				})
				var db dstate.DeploymentState
				path := filepath.Join(t.TempDir(), "resources.json")
				require.NoError(t, db.Open(ctx, path, mode, false, dstate.OpenDmsArgs{}))
				db.InitializeOperationBuffer(ctx, "dep-1", 1, 1)
				require.NoError(t, db.SaveState(ctx, "resources.jobs.job", "id-1", struct{}{}, nil))
				_, err := db.Finalize(ctx)
				require.NoError(t, err)
				completionCtx, cancel := context.WithTimeout(ctx, 90*time.Second)
				defer cancel()
				start := time.Now()
				completed, err := db.CompleteVersion(completionCtx, true)
				assert.Equal(t, 90*time.Second, time.Since(start))
				assert.False(t, completed)
				assert.False(t, db.RecordsHistory())
				assert.Empty(t, db.DeploymentID)
				assert.Zero(t, db.VersionID)
				if mode == dstate.WritingDMS {
					require.ErrorIs(t, err, context.DeadlineExceeded)
				} else {
					require.NoError(t, err)
				}
			})
		})
	}
}

func TestEnforcedHistorySurvivesWALRecovery(t *testing.T) {
	for _, withResource := range []bool{false, true} {
		t.Run(map[bool]string{false: "feature only", true: "with resource"}[withResource], func(t *testing.T) {
			ctx := modeContext(t, nil)
			path := filepath.Join(t.TempDir(), "resources.json")
			var db dstate.DeploymentState
			require.NoError(t, db.Open(ctx, path, dstate.WritingFileWithHistoryEnforced, false, dstate.OpenDmsArgs{}))
			if withResource {
				require.NoError(t, db.SaveState(ctx, "resources.jobs.job", "id-1", struct{}{}, nil))
			}
			wal, err := os.ReadFile(path + ".wal")
			require.NoError(t, err)
			_, err = db.Finalize(ctx)
			require.NoError(t, err)
			// Reproduce an interrupted deployment using only its WAL.
			recoveredPath := filepath.Join(t.TempDir(), "resources.json")
			require.NoError(t, os.WriteFile(recoveredPath+".wal", wal, 0o600))
			var recovered dstate.DeploymentState
			require.NoError(t, recovered.Open(t.Context(), recoveredPath, dstate.ReadingFile, true, dstate.OpenDmsArgs{}))
			assert.Contains(t, recovered.StateFeatures(), dstate.FeatureDeploymentHistoryBestEffort)
			assert.Equal(t, 1, recovered.GetSerial())
			assert.FileExists(t, recoveredPath)
			assert.NoFileExists(t, recoveredPath+".wal")
			require.NoError(t, recovered.UpgradeToWrite())
			if withResource {
				assert.Equal(t, "id-1", recovered.GetResourceID("resources.jobs.job"))
			}
			_, err = recovered.Finalize(ctx)
			require.NoError(t, err)
		})
	}
}

func TestOpenRejectsIncompatibleModesAndFeatures(t *testing.T) {
	for _, tc := range []struct {
		name     string
		mode     dstate.Mode
		features string
		wantErr  string
	}{
		{"unknown feature", dstate.ReadingFile, `{"future":{}}`, "requires features"},
		{"file reading DMS with best effort marker", dstate.ReadingFile, `{"deployment_history":{},"deployment_history_best_effort":{}}`, "not supported"},
		{"file writing DMS", dstate.WritingFile, `{"deployment_history":{}}`, "not supported"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "resources.json")
			seed := `{"state_version":3,"features":` + tc.features + `,"state":{}}`
			require.NoError(t, os.WriteFile(path, []byte(seed), 0o600))
			var db dstate.DeploymentState
			err := db.Open(t.Context(), path, tc.mode, false, dstate.OpenDmsArgs{})
			require.ErrorContains(t, err, tc.wantErr)
			assert.False(t, db.IsOpen())
			assert.NoFileExists(t, path+".wal")
		})
	}
}

func TestCompleteVersionAfterFinalize(t *testing.T) {
	for _, mode := range []dstate.Mode{dstate.WritingDMS, dstate.WritingFileWithHistory, dstate.WritingFileWithHistoryEnforced} {
		t.Run(fmt.Sprint(mode), func(t *testing.T) {
			completions := 0
			ctx := modeContext(t, func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if r.Method == http.MethodPost {
					completions++
					assert.Equal(t, "/api/2.0/bundle/deployments/dep-1/versions/1/complete", r.URL.Path)
					var body map[string]string
					assert.NoError(t, json.NewDecoder(r.Body).Decode(&body))
					assert.Equal(t, "VERSION_COMPLETE_SUCCESS", body["completion_reason"])
					_, _ = w.Write([]byte(`{}`))
				} else {
					_, _ = w.Write([]byte(`{"sequence_id":1}`))
				}
			})
			var db dstate.DeploymentState
			for cycle := range 2 {
				path := filepath.Join(t.TempDir(), "resources.json")
				require.NoError(t, db.Open(ctx, path, mode, false, dstate.OpenDmsArgs{}))
				db.InitializeOperationBuffer(ctx, "dep-1", 1, 1)
				require.NoError(t, db.SaveState(ctx, "resources.jobs.job", "id-1", struct{}{}, nil))
				_, err := db.Finalize(ctx)
				require.NoError(t, err)
				assert.False(t, db.IsOpen())
				completed, err := db.CompleteVersion(ctx, true)
				require.NoError(t, err)
				assert.True(t, completed)
				completed, err = db.CompleteVersion(ctx, true)
				require.NoError(t, err)
				assert.False(t, completed)
				assert.Equal(t, cycle+1, completions)
				assert.False(t, db.RecordsHistory())
				assert.Empty(t, db.DeploymentID)
				assert.Zero(t, db.VersionID)
			}
		})
	}
}

func TestFailedUpgradeLeavesReadModeAvailable(t *testing.T) {
	path := filepath.Join(t.TempDir(), "resources.json")
	var db dstate.DeploymentState
	require.NoError(t, db.Open(t.Context(), path, dstate.ReadingFile, true, dstate.OpenDmsArgs{}))
	require.NoError(t, os.WriteFile(path+".wal", []byte("occupied"), 0o600))
	require.Error(t, db.UpgradeToWrite())
	assert.NotPanics(t, db.AssertOpenedForRead)
	require.NoError(t, os.Remove(path+".wal"))
	require.NoError(t, db.UpgradeToWrite())
	_, err := db.Finalize(t.Context())
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
