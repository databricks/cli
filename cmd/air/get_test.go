package aircmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"maps"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"text/template"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// renderGet renders the get template against the JSON envelope, exactly as the
// command does for a sweep run, so the test covers the real template branches.
func renderGet(t *testing.T, data getData) string {
	t.Helper()
	tmpl, err := template.New("get").Parse(getTemplate)
	require.NoError(t, err)
	var buf bytes.Buffer
	require.NoError(t, tmpl.Execute(&buf, envelope{V: envelopeVersion, Data: data}))
	return buf.String()
}

// TestGetCommandShape locks in that `get` takes the run id directly as
// `air get JOB_RUN_ID` (there is no `run` subcommand — it was collapsed back
// into `get`). The acceptance test exercises the happy path end to end.
func TestGetCommandShape(t *testing.T) {
	cmd := newGetCommand()
	assert.Equal(t, "get JOB_RUN_ID", cmd.Use)
	assert.Empty(t, cmd.Commands())
	// ExactArgs(1): exactly one run id is required.
	assert.NoError(t, cmd.Args(cmd, []string{"123"}))
	assert.Error(t, cmd.Args(cmd, []string{}))
	assert.Error(t, cmd.Args(cmd, []string{"1", "2"}))
}

func TestGetRunInvalidID(t *testing.T) {
	m := mocks.NewMockWorkspaceClient(t)
	ctx := cmdctx.SetWorkspaceClient(cmdio.MockDiscard(t.Context()), m.WorkspaceClient)
	cmd := withOutput(newGetCommand(), flags.OutputText)
	cmd.SetContext(ctx)

	err := cmd.RunE(cmd, []string{"abc"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid JOB_RUN_ID")
}

// notFoundGetServer serves a runs/get that reports the run as missing (400
// INVALID_PARAMETER_VALUE, which the SDK maps to ErrResourceDoesNotExist for
// this path).
func notFoundGetServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/2.2/jobs/runs/get" {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error_code":"INVALID_PARAMETER_VALUE","message":"Run 5 does not exist."}`))
			return
		}
		// SDK config discovery.
		_, _ = w.Write([]byte(`{"userName":"u@example.com"}`))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestGetRunNotFound(t *testing.T) {
	srv := notFoundGetServer(t)
	ctx := cmdctx.SetWorkspaceClient(cmdio.MockDiscard(t.Context()), newTestWorkspaceClient(t, srv.URL))
	cmd := withOutput(newGetCommand(), flags.OutputText)
	cmd.SetContext(ctx)

	err := cmd.RunE(cmd, []string{"5"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "run 5 not found")
}

// identityMeConfig returns a config whose identity/me call answers with resp.
func identityMeConfig(t *testing.T, resp testserver.Response) *config.Config {
	t.Helper()
	server := testserver.New(t)
	t.Cleanup(server.Close)
	server.Handle("GET", "/api/2.0/identity/me", func(testserver.Request) any {
		return resp
	})
	return &config.Config{Host: server.URL, Token: "token"}
}

func TestGetRunAuthFailed(t *testing.T) {
	m := mocks.NewMockWorkspaceClient(t)
	m.WorkspaceClient.Config = identityMeConfig(t, testserver.Response{StatusCode: http.StatusForbidden, Body: map[string]any{"error_code": "PERMISSION_DENIED", "message": "denied"}})
	// GetRun's 403 is confirmed as a credential failure by the current-user probe, so
	// nothing is rendered.
	m.GetMockJobsAPI().EXPECT().GetRun(mock.Anything, mock.Anything).Return(nil, apierr.ErrPermissionDenied)
	ctx := cmdctx.SetWorkspaceClient(cmdio.MockDiscard(t.Context()), m.WorkspaceClient)
	cmd := withOutput(newGetCommand(), flags.OutputText)
	cmd.SetContext(ctx)

	err := cmd.RunE(cmd, []string{"5"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "authentication was not successful")
}

func TestGetRunAuthProbeBadRequest(t *testing.T) {
	m := mocks.NewMockWorkspaceClient(t)
	// A probe failure that is not a 401/403 must not be misreported as an auth
	// error; it surfaces as a retryable internal error instead.
	m.WorkspaceClient.Config = identityMeConfig(t, testserver.Response{StatusCode: http.StatusBadRequest, Body: map[string]any{"error_code": "BAD_REQUEST", "message": "unavailable"}})
	m.GetMockJobsAPI().EXPECT().GetRun(mock.Anything, mock.Anything).Return(nil, apierr.ErrUnauthenticated)
	ctx := cmdctx.SetWorkspaceClient(cmdio.MockDiscard(t.Context()), m.WorkspaceClient)
	cmd := withOutput(newGetCommand(), flags.OutputText)
	cmd.SetContext(ctx)

	err := cmd.RunE(cmd, []string{"5"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to verify authentication")
	assert.NotContains(t, err.Error(), "authentication was not successful")
}

func TestGetRunDeniedWithValidCredentials(t *testing.T) {
	m := mocks.NewMockWorkspaceClient(t)
	m.WorkspaceClient.Config = identityMeConfig(t, testserver.Response{StatusCode: http.StatusOK, Body: map[string]any{"user": map[string]any{"user_id": "1000012345", "username": "me@example.com"}}})
	// A 403 on the run with credentials the current-user probe accepts is missing
	// access to the run, not a failed login.
	m.GetMockJobsAPI().EXPECT().GetRun(mock.Anything, mock.Anything).Return(nil, apierr.ErrPermissionDenied)
	ctx := cmdctx.SetWorkspaceClient(cmdio.MockDiscard(t.Context()), m.WorkspaceClient)
	cmd := withOutput(newGetCommand(), flags.OutputText)
	cmd.SetContext(ctx)

	err := cmd.RunE(cmd, []string{"5"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get status for run 5")
	assert.NotContains(t, err.Error(), "authentication was not successful")
}

func TestAuthError(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	cmd := withOutput(newGetCommand(), flags.OutputText)

	// No configurable credentials maps to the missing-profile hint.
	noProfile := authError(ctx, cmd, config.ErrCannotConfigureDefault)
	require.Error(t, noProfile)
	assert.Contains(t, noProfile.Error(), "no default profile is set")

	// A 401 / 403 (via the SDK sentinels) is a real auth failure.
	unauth := authError(ctx, cmd, apierr.ErrUnauthenticated)
	require.Error(t, unauth)
	assert.Contains(t, unauth.Error(), "authentication was not successful")

	denied := authError(ctx, cmd, apierr.ErrPermissionDenied)
	require.Error(t, denied)
	assert.Contains(t, denied.Error(), "authentication was not successful")

	transient := authError(ctx, cmd, errors.New("connection reset"))
	require.Error(t, transient)
	assert.Contains(t, transient.Error(), "failed to verify authentication")
	assert.Contains(t, transient.Error(), "connection reset")
}

func TestGetRunNotFoundJSON(t *testing.T) {
	var buf bytes.Buffer
	srv := notFoundGetServer(t)
	ctx := cmdctx.SetWorkspaceClient(t.Context(), newTestWorkspaceClient(t, srv.URL))
	ctx = cmdio.InContext(ctx, cmdio.NewIO(ctx, flags.OutputJSON, nil, &buf, &buf, "", ""))
	cmd := withOutput(newGetCommand(), flags.OutputJSON)
	cmd.SetContext(ctx)

	// In JSON mode the not-found error is a structured envelope, not a bare error.
	err := cmd.RunE(cmd, []string{"5"})
	require.ErrorIs(t, err, root.ErrAlreadyPrinted)

	var got errorEnvelope
	require.NoError(t, json.Unmarshal(buf.Bytes(), &got))
	assert.Equal(t, jsonError{Code: "NOT_FOUND", Kind: "NOT_FOUND", Message: "run 5 not found: check the run ID and that it is a job run ID"}, got.Error)
}

const succeededRun = `{"run_id": 5, "state": {"life_cycle_state": "TERMINATED", "result_state": "SUCCESS"}}`

// getRunServer serves run from runs/get, outputs[task run ID] from
// runs/get-output (a 500 for a task run with no entry), and a /Me that reports
// workspace 12345 in the X-Databricks-Org-Id header. *meCalls counts /Me
// requests.
func getRunServer(t *testing.T, meCalls *atomic.Int32, run string, outputs map[string]string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(withAirIdentityMe(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/2.2/jobs/runs/get":
			_, _ = w.Write([]byte(run))
		case r.URL.Path == "/api/2.2/jobs/runs/get-output":
			output, ok := outputs[r.URL.Query().Get("run_id")]
			if !ok {
				w.WriteHeader(http.StatusInternalServerError)
				output = `{"error_code": "INTERNAL_ERROR", "message": "get-output failed"}`
			}
			_, _ = w.Write([]byte(output))
		case strings.HasSuffix(r.URL.Path, "/scim/v2/Me"):
			meCalls.Add(1)
			w.Header().Set("X-Databricks-Org-Id", "12345")
			_, _ = w.Write([]byte(`{"userName": "u@example.com"}`))
		default:
			// SDK config discovery.
			_, _ = w.Write([]byte(`{}`))
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

// retryGetServer serves two runs for the full `air get` JSON flow: run 123, a
// failed two-attempt run whose last retry (458) never reached MLflow while
// attempt 0 (456) did, and run 124, a control whose only attempt (457) logged
// to MLflow. get-output is query-aware, so each retry's output is served
// separately. The Me endpoint also carries the workspace-ID header
// CurrentWorkspaceID reads.
func retryGetServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/2.0/preview/scim/v2/Me":
			w.Header().Set("X-Databricks-Org-Id", "9999")
			_, _ = w.Write([]byte(`{"userName":"u@example.com"}`))
		case r.URL.Path == "/api/2.2/jobs/runs/get" && r.URL.Query().Get("run_id") == "123":
			_, _ = w.Write([]byte(`{
				"run_id": 123,
				"start_time": 1700000000000,
				"end_time": 1700000012000,
				"state": {"life_cycle_state": "TERMINATED", "result_state": "FAILED"},
				"tasks": [
					{"task_key": "train", "run_id": 456, "attempt_number": 0},
					{"task_key": "train", "run_id": 458, "attempt_number": 2}
				]
			}`))
		case r.URL.Path == "/api/2.2/jobs/runs/get" && r.URL.Query().Get("run_id") == "124":
			_, _ = w.Write([]byte(`{
				"run_id": 124,
				"start_time": 1700000000000,
				"end_time": 1700000012000,
				"state": {"life_cycle_state": "TERMINATED", "result_state": "SUCCESS"},
				"tasks": [{"task_key": "train", "run_id": 457, "attempt_number": 0}]
			}`))
		case r.URL.Path == "/api/2.2/jobs/runs/get-output":
			switch r.URL.Query().Get("run_id") {
			case "456":
				_, _ = w.Write([]byte(`{"ai_runtime_task_output":{"mlflow_experiment_id":"E1","mlflow_run_id":"R0"}}`))
			case "457":
				_, _ = w.Write([]byte(`{"ai_runtime_task_output":{"mlflow_experiment_id":"E1","mlflow_run_id":"R1"}}`))
			default:
				_, _ = w.Write([]byte(`{}`))
			}
		default:
			_, _ = w.Write([]byte(`{}`))
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func runGetJSON(t *testing.T, w *databricks.WorkspaceClient) (string, error) {
	t.Helper()
	var buf bytes.Buffer
	ctx := cmdctx.SetWorkspaceClient(t.Context(), w)
	ctx = cmdio.InContext(ctx, cmdio.NewIO(ctx, flags.OutputJSON, nil, &buf, &buf, "", ""))
	cmd := withOutput(newGetCommand(), flags.OutputJSON)
	cmd.SetContext(ctx)
	err := cmd.RunE(cmd, []string{"5"})
	return buf.String(), err
}

func TestGetRunURLUsesConfiguredWorkspaceID(t *testing.T) {
	var meCalls atomic.Int32
	srv := getRunServer(t, &meCalls, succeededRun, nil)
	w, err := databricks.NewWorkspaceClient(&databricks.Config{Host: srv.URL, Token: "token", WorkspaceID: "67890"})
	require.NoError(t, err)

	out, err := runGetJSON(t, w)
	require.NoError(t, err)
	assert.Contains(t, out, "/jobs/runs/5?o=67890")
	assert.Zero(t, meCalls.Load(), "a configured workspace ID needs no /Me lookup")
}

func TestGetRunURLLooksUpUnsetWorkspaceID(t *testing.T) {
	// The SDK also reads the workspace ID from the environment.
	t.Setenv("DATABRICKS_WORKSPACE_ID", "")
	var meCalls atomic.Int32
	srv := getRunServer(t, &meCalls, succeededRun, nil)

	out, err := runGetJSON(t, newTestWorkspaceClient(t, srv.URL))
	require.NoError(t, err)
	assert.Contains(t, out, "/jobs/runs/5?o=12345")
	assert.Equal(t, int32(1), meCalls.Load())
}

func TestGetRunRejectsNonNumericWorkspaceID(t *testing.T) {
	var meCalls atomic.Int32
	srv := getRunServer(t, &meCalls, succeededRun, nil)
	w, err := databricks.NewWorkspaceClient(&databricks.Config{Host: srv.URL, Token: "token", WorkspaceID: "not-a-number"})
	require.NoError(t, err)

	out, err := runGetJSON(t, w)
	require.ErrorIs(t, err, root.ErrAlreadyPrinted)
	var got errorEnvelope
	require.NoError(t, json.Unmarshal([]byte(out), &got))
	assert.Equal(t, jsonError{Code: "INVALID_ARGS", Kind: "PERMANENT", Message: `configured workspace_id "not-a-number" is not a numeric workspace ID`}, got.Error)
}

func TestGetRunJSONMLflowIDs(t *testing.T) {
	const (
		succeededTaskRun = `{"run_id": 5, "state": {"life_cycle_state": "TERMINATED", "result_state": "SUCCESS"}, "tasks": [{"run_id": 50}]}`
		outputWithIDs    = `{"ai_runtime_task_output": {"mlflow_experiment_id": "E1", "mlflow_run_id": "R1"}}`
	)
	tests := []struct {
		name    string
		run     string
		outputs map[string]string
		// wantExperimentID and wantRunID are the IDs the JSON names, or empty
		// when mlflow_url and both IDs must be null.
		wantExperimentID string
		wantRunID        string
		// wantRetry is the retry mlflow_retry names, or nil when the key must be
		// absent.
		wantRetry *int
	}{
		{
			name:             "set for a finished run",
			run:              succeededTaskRun,
			outputs:          map[string]string{"50": outputWithIDs},
			wantExperimentID: "E1",
			wantRunID:        "R1",
		},
		{
			name:             "set while the run is running",
			run:              `{"run_id": 5, "state": {"life_cycle_state": "RUNNING"}, "tasks": [{"run_id": 50}]}`,
			outputs:          map[string]string{"50": outputWithIDs},
			wantExperimentID: "E1",
			wantRunID:        "R1",
		},
		{
			name: "from an earlier retry when the last retry never started",
			run: `{"run_id": 5, "state": {"life_cycle_state": "TERMINATED", "result_state": "FAILED"},
				"tasks": [{"task_key": "train", "run_id": 50}, {"task_key": "train", "run_id": 51, "attempt_number": 1}]}`,
			outputs: map[string]string{
				"50": `{"ai_runtime_task_output": {"mlflow_experiment_id": "E1", "mlflow_run_id": "R0"}}`,
				"51": `{}`,
			},
			wantExperimentID: "E1",
			wantRunID:        "R0",
			wantRetry:        new(0),
		},
		{
			name:    "null when the run has no MLflow run",
			run:     succeededTaskRun,
			outputs: map[string]string{"50": `{}`},
		},
		{
			name:    "null when the output has only an experiment ID",
			run:     succeededTaskRun,
			outputs: map[string]string{"50": `{"ai_runtime_task_output": {"mlflow_experiment_id": "E1"}}`},
		},
		{
			// No output for task run 50, so its get-output fails.
			name: "null when get-output fails",
			run:  succeededTaskRun,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			srv := getRunServer(t, new(atomic.Int32), tc.run, tc.outputs)
			w, err := databricks.NewWorkspaceClient(&databricks.Config{Host: srv.URL, Token: "token", WorkspaceID: "67890"})
			require.NoError(t, err)

			out, err := runGetJSON(t, w)
			require.NoError(t, err)
			// Decode into a map, which tells a missing key from null.
			var got struct {
				Data map[string]any `json:"data"`
			}
			require.NoError(t, json.Unmarshal([]byte(out), &got))
			maps.DeleteFunc(got.Data, func(key string, _ any) bool { return !strings.HasPrefix(key, "mlflow_") })
			want := map[string]any{"mlflow_url": nil, "mlflow_experiment_id": nil, "mlflow_run_id": nil}
			if tc.wantRunID != "" {
				want = map[string]any{
					"mlflow_url":           srv.URL + "/ml/experiments/" + tc.wantExperimentID + "/runs/" + tc.wantRunID + "/logs",
					"mlflow_experiment_id": tc.wantExperimentID,
					"mlflow_run_id":        tc.wantRunID,
				}
			}
			// getRunServer serves no MLflow run, so mlflow_run_name stays null.
			want["mlflow_run_name"] = nil
			if tc.wantRetry != nil {
				want["mlflow_retry"] = float64(*tc.wantRetry)
			}
			assert.Equal(t, want, got.Data)
		})
	}
}

// getRunJSON runs `air get RUN_ID -o json` against srv and returns the decoded
// data payload.
func getRunJSON(t *testing.T, srv *httptest.Server, runID string) map[string]any {
	t.Helper()
	var buf bytes.Buffer
	ctx := cmdctx.SetWorkspaceClient(t.Context(), newTestWorkspaceClient(t, srv.URL))
	ctx = cmdio.InContext(ctx, cmdio.NewIO(ctx, flags.OutputJSON, nil, &buf, &buf, "", ""))
	cmd := withOutput(newGetCommand(), flags.OutputJSON)
	cmd.SetContext(ctx)
	require.NoError(t, cmd.RunE(cmd, []string{runID}))

	var got envelope
	require.NoError(t, json.Unmarshal(buf.Bytes(), &got))
	data, ok := got.Data.(map[string]any)
	require.True(t, ok, "data payload should be an object")
	return data
}

// TestGetRunMLflowRetryJSON locks the JSON shape of the MLflow fields. A run
// whose last retry never reached MLflow links to an earlier retry's MLflow run
// and names that retry in mlflow_retry; a run whose latest attempt logged to
// MLflow links to it without the key.
func TestGetRunMLflowRetryJSON(t *testing.T) {
	srv := retryGetServer(t)
	host := srv.URL

	t.Run("an older retry's mlflow run names the retry", func(t *testing.T) {
		data := getRunJSON(t, srv, "123")
		assert.JSONEq(t, `{
			"run_id":"123","task_key":null,"status":"FAILED","started_at":"2023-11-14T22:13:20+00:00",
			"duration_seconds":12,"attempt_number":2,"experiment_name":null,
			"dashboard_url":"`+host+`/jobs/runs/123?o=9999",
			"mlflow_url":"`+host+`/ml/experiments/E1/runs/R0/logs",
			"mlflow_experiment_id":"E1","mlflow_run_id":"R0",
			"mlflow_retry":0,
			"est_remaining_seconds":null,"est_percent_complete":null,
			"user":null,"accelerator_type":null,"num_accelerators":null,
			"environment_version":null,"base_environment":null,
			"max_retries":null,"mlflow_run_name":null,"sweep":null
		}`, mustJSON(t, data))
	})

	t.Run("the latest attempt's mlflow run has no mlflow_retry key", func(t *testing.T) {
		data := getRunJSON(t, srv, "124")
		assert.JSONEq(t, `{
			"run_id":"124","task_key":null,"status":"SUCCESS","started_at":"2023-11-14T22:13:20+00:00",
			"duration_seconds":12,"attempt_number":0,"experiment_name":null,
			"dashboard_url":"`+host+`/jobs/runs/124?o=9999",
			"mlflow_url":"`+host+`/ml/experiments/E1/runs/R1/logs",
			"mlflow_experiment_id":"E1","mlflow_run_id":"R1",
			"est_remaining_seconds":null,"est_percent_complete":null,
			"user":null,"accelerator_type":null,"num_accelerators":null,
			"environment_version":null,"base_environment":null,
			"max_retries":null,"mlflow_run_name":null,"sweep":null
		}`, mustJSON(t, data))
	})
}

func mustJSON(t *testing.T, v any) string {
	t.Helper()
	body, err := json.Marshal(v)
	require.NoError(t, err)
	return string(body)
}

func TestGetTemplateSweep(t *testing.T) {
	out := renderGet(t, getData{
		RunID:  "456",
		Status: "RUNNING",
		Sweep: &sweepInfo{
			Total: 4, Completed: 2, Succeeded: 1, Failed: 1, Active: 2,
			Tasks: []sweepTask{
				{TaskKey: "iter_0", RunID: "789", Status: "SUCCESS", Experiment: "my-exp"},
				{TaskKey: "iter_1", RunID: "790", Status: "FAILED", Experiment: "my-exp"},
			},
		},
	})
	assert.Contains(t, out, "Sweep Run ID: 456")
	assert.Contains(t, out, "Total:        4")
	assert.Contains(t, out, "Sweep Tasks:")
	assert.Contains(t, out, "iter_0")
	assert.Contains(t, out, "iter_1")
	assert.Contains(t, out, "FAILED")
	assert.Contains(t, out, "my-exp")
}

func TestGetTemplateSweepNoTasks(t *testing.T) {
	// A sweep whose iterations haven't materialized yet: counts show, but the
	// task table header is hidden.
	out := renderGet(t, getData{
		RunID:  "456",
		Status: "RUNNING",
		Sweep:  &sweepInfo{Total: 4, Active: 4},
	})
	assert.Contains(t, out, "Sweep Run ID: 456")
	assert.Contains(t, out, "Total:        4")
	assert.NotContains(t, out, "Sweep Tasks:")
}

func TestBuildGetData(t *testing.T) {
	run := &jobs.Run{
		RunId:           123,
		CreatorUserName: "me@example.com",
		StartTime:       1700000000000,
		EndTime:         1700000012000,
		State:           &jobs.RunState{ResultState: jobs.RunResultStateSuccess},
		Tasks: []jobs.RunTask{{
			AttemptNumber: 1,
			GenAiComputeTask: &jobs.GenAiComputeTask{
				MlflowExperimentName: "/Users/me@example.com/exp",
				Compute:              &jobs.ComputeConfig{NumGpus: 8, GpuType: "GPU_8xH100"},
			},
		}},
	}
	d := buildGetData(run)
	assert.Equal(t, "123", d.RunID)
	assert.Equal(t, "SUCCESS", d.Status)
	assert.Equal(t, 1, d.AttemptNumber)
	assert.Equal(t, "2023-11-14 22:13 UTC", d.SubmittedDisplay)
	assert.Equal(t, "me@example.com", d.UserDisplay)
	assert.Equal(t, "8x H100", d.AcceleratorsDisplay)
	assert.Equal(t, "12s", d.DurationDisplay)
	assert.Equal(t, "exp", d.ExperimentDisplay)
	require.NotNil(t, d.ExperimentName)
	assert.Equal(t, "exp", *d.ExperimentName)
	require.NotNil(t, d.DurationSeconds)
	assert.Equal(t, int64(12), *d.DurationSeconds)
}

func TestBuildGetDataTerminationReasonJSON(t *testing.T) {
	failed := buildGetData(&jobs.Run{
		RunId: 5,
		State: &jobs.RunState{ResultState: jobs.RunResultStateFailed},
		Status: &jobs.RunStatus{TerminationDetails: &jobs.TerminationDetails{
			Message: "out of memory",
		}},
	})
	body, err := json.Marshal(failed)
	require.NoError(t, err)
	assert.JSONEq(t, `{
		"run_id":"5","task_key":null,"status":"FAILED","started_at":null,"duration_seconds":null,
		"attempt_number":0,"experiment_name":null,"dashboard_url":"","mlflow_url":null,
		"mlflow_experiment_id":null,"mlflow_run_id":null,
		"termination_reason":"out of memory","est_remaining_seconds":null,"est_percent_complete":null,
		"user":null,"accelerator_type":null,"num_accelerators":null,"environment_version":null,"base_environment":null,
		"max_retries":null,"mlflow_run_name":null,"sweep":null
	}`, string(body))

	canceled := buildGetData(&jobs.Run{RunId: 6, State: &jobs.RunState{ResultState: jobs.RunResultStateCanceled, StateMessage: "user canceled"}})
	body, err = json.Marshal(canceled)
	require.NoError(t, err)
	assert.NotContains(t, string(body), "termination_reason")
}

func TestBuildGetDataWaitingForComputeStatus(t *testing.T) {
	run := &jobs.Run{
		RunId: 8,
		State: &jobs.RunState{LifeCycleState: jobs.RunLifeCycleStateRunning},
		Tasks: []jobs.RunTask{{State: &jobs.RunState{LifeCycleState: jobs.RunLifeCycleStateQueued}}},
	}

	body, err := json.Marshal(buildGetData(run))
	require.NoError(t, err)

	assert.JSONEq(t, `{
		"run_id":"8","task_key":null,"status":"PENDING","started_at":null,"duration_seconds":null,
		"attempt_number":0,"experiment_name":null,"dashboard_url":"","mlflow_url":null,
		"mlflow_experiment_id":null,"mlflow_run_id":null,
		"est_remaining_seconds":null,"est_percent_complete":null,
		"user":null,"accelerator_type":null,"num_accelerators":null,"environment_version":null,"base_environment":null,
		"max_retries":null,"mlflow_run_name":null,"sweep":null
	}`, string(body))
}

func TestGetTemplateUsesDisplayStatus(t *testing.T) {
	out := renderGet(t, getData{
		RunID:         "456",
		Status:        "RUNNING",
		DisplayStatus: "PENDING",
		Sweep:         &sweepInfo{Total: 1, Active: 1},
	})
	assert.Contains(t, out, "Status:       PENDING")
}

func TestEnrichFromAiRuntimeTask(t *testing.T) {
	t.Run("fills config path, experiment, and accelerators", func(t *testing.T) {
		run := &jobs.Run{RunId: 5, Tasks: []jobs.RunTask{{
			AiRuntimeTask: &jobs.AiRuntimeTask{
				Experiment: "/Users/me@example.com/my-exp",
				Deployments: []jobs.DeploymentSpec{{
					CommandPath: "/Workspace/run/command.sh",
					Compute:     jobs.ComputeSpec{AcceleratorType: jobs.ComputeSpecAcceleratorTypeGpu1xA10, AcceleratorCount: 1},
				}},
			},
		}}}
		data := &getData{ExperimentDisplay: na, AcceleratorsDisplay: na}
		enrichFromAiRuntimeTask(run, data)
		assert.Equal(t, "/Workspace/run/training_config.yaml", data.TrainingConfigPath)
		assert.Equal(t, "my-exp", data.ExperimentDisplay)
		require.NotNil(t, data.ExperimentName)
		assert.Equal(t, "my-exp", *data.ExperimentName)
		assert.Equal(t, "1x A10", data.AcceleratorsDisplay)
	})

	t.Run("leaves fallbacks when the run has no ai_runtime_task", func(t *testing.T) {
		run := &jobs.Run{RunId: 5, Tasks: []jobs.RunTask{{}}}
		data := &getData{ExperimentDisplay: na, AcceleratorsDisplay: na}
		enrichFromAiRuntimeTask(run, data)
		assert.Empty(t, data.TrainingConfigPath)
		assert.Equal(t, na, data.ExperimentDisplay)
		assert.Equal(t, na, data.AcceleratorsDisplay)
	})
}

func TestBuildGetDataEmpty(t *testing.T) {
	// A run with no tasks, creator, or timing renders every text cell as "N/A".
	d := buildGetData(&jobs.Run{RunId: 7})
	assert.Equal(t, "7", d.RunID)
	assert.Equal(t, na, d.SubmittedDisplay)
	assert.Equal(t, na, d.DurationDisplay)
	assert.Equal(t, na, d.ExperimentDisplay)
	assert.Equal(t, na, d.UserDisplay)
	assert.Equal(t, na, d.AcceleratorsDisplay)
}

func TestBuildGetDataMetadata(t *testing.T) {
	t.Run("legacy task", func(t *testing.T) {
		d := buildGetData(&jobs.Run{
			CreatorUserName: "me@example.com",
			Tasks: []jobs.RunTask{{GenAiComputeTask: &jobs.GenAiComputeTask{
				Compute: &jobs.ComputeConfig{GpuType: "GPU_8xH100", NumGpus: 8},
			}}},
		})
		assert.Equal(t, "me@example.com", *d.User)
		assert.Equal(t, new("GPU_8xH100"), d.AcceleratorType)
		assert.Equal(t, new(8), d.NumAccelerators)
		assert.Equal(t, "8x H100", d.AcceleratorsDisplay)
	})

	t.Run("ai_runtime task", func(t *testing.T) {
		d := buildGetData(&jobs.Run{Tasks: []jobs.RunTask{{AiRuntimeTask: &jobs.AiRuntimeTask{
			Deployments: []jobs.DeploymentSpec{{
				Compute: jobs.ComputeSpec{AcceleratorType: jobs.ComputeSpecAcceleratorTypeGpu1xA10, AcceleratorCount: 1},
			}},
		}}}})
		assert.Equal(t, new("GPU_1xA10"), d.AcceleratorType)
		assert.Equal(t, new(1), d.NumAccelerators)
		assert.Equal(t, "1x A10", d.AcceleratorsDisplay)
	})

	t.Run("no compute or creator stays null", func(t *testing.T) {
		d := buildGetData(&jobs.Run{Tasks: []jobs.RunTask{{GenAiComputeTask: &jobs.GenAiComputeTask{
			Compute: &jobs.ComputeConfig{NumGpus: 0},
		}}}})
		assert.Nil(t, d.User)
		assert.Nil(t, d.AcceleratorType)
		assert.Nil(t, d.NumAccelerators)
		assert.Equal(t, na, d.AcceleratorsDisplay)
		assert.Equal(t, na, d.UserDisplay)
	})
}

func TestGetAcceleratorJSONMatchesListWhenCountMissing(t *testing.T) {
	run := &jobs.Run{Tasks: []jobs.RunTask{{GenAiComputeTask: &jobs.GenAiComputeTask{Compute: &jobs.ComputeConfig{GpuType: "GPU_1xA10"}}}}}
	data := buildGetData(run)
	encoded, err := json.Marshal(data)
	require.NoError(t, err)
	var fields map[string]any
	require.NoError(t, json.Unmarshal(encoded, &fields))
	assert.Equal(t, "GPU_1xA10", fields["accelerator_type"])
	assert.Nil(t, fields["num_accelerators"])
}

func TestEnrichRunDetails(t *testing.T) {
	const uploaded = "/Workspace/run/training_config.yaml"
	submitAiRuntime := `"run_type": "SUBMIT_RUN", "tasks": [{"ai_runtime_task": {}}]`
	tests := []struct {
		name        string
		run         string
		config      string // served at uploaded; empty means not found
		mlflowName  string // empty means the MLflow lookup fails
		wantVersion *string
		wantBase    *string
		wantEnvCell string
		wantSource  *string
		wantRetries *runMaxRetries
		wantBody    string
	}{
		{
			name:        "uploaded config and environment version",
			run:         `{"run_id": 1, "environments": [{"environment_key": "default", "spec": {"environment_version": "6"}}], ` + submitAiRuntime + `}`,
			config:      "max_retries: 1\n",
			mlflowName:  "my-run",
			wantVersion: new("6"), wantEnvCell: "6",
			wantSource: new("uploaded"), wantRetries: &runMaxRetries{Limit: 1, Source: maxRetriesConfigured}, wantBody: "max_retries: 1",
		},
		{
			name:        "uploaded config without max_retries uses the CLI default",
			run:         `{"run_id": 1, ` + submitAiRuntime + `}`,
			config:      "experiment_name: e\n",
			wantEnvCell: na, wantSource: new("uploaded"),
			wantRetries: &runMaxRetries{Limit: 3, Source: maxRetriesCLIDefault}, wantBody: "experiment_name: e",
		},
		{
			name:        "server-reported unlimited",
			run:         `{"run_id": 1, "run_type": "SUBMIT_RUN", "tasks": [{"max_retries": -1, "ai_runtime_task": {}}]}`,
			wantEnvCell: na, wantRetries: &runMaxRetries{Limit: -1, Source: maxRetriesReported},
		},
		{
			name:        "job run with a readable config is unknown",
			run:         `{"run_id": 1, "run_type": "JOB_RUN", "tasks": [{"ai_runtime_task": {}}]}`,
			config:      "max_retries: 1\n",
			wantEnvCell: na, wantSource: new("uploaded"), wantBody: "max_retries: 1",
		},
		{
			name:     "base environment, config unreadable",
			run:      `{"run_id": 1, "environments": [{"environment_key": "default", "spec": {"base_environment": "workspace-base-environments/databricks_ai_v5"}}], ` + submitAiRuntime + `}`,
			wantBase: new("workspace-base-environments/databricks_ai_v5"), wantEnvCell: "databricks_ai_v5",
		},
		{
			name:        "legacy inline parameters",
			run:         `{"run_id": 1, "run_type": "SUBMIT_RUN", "tasks": [{"gen_ai_compute_task": {"yaml_parameters": "max_retries: 2\ncommand: echo hi\n"}}]}`,
			wantEnvCell: na, wantSource: new("yaml_parameters"),
			wantRetries: &runMaxRetries{Limit: 2, Source: maxRetriesConfigured}, wantBody: "max_retries: 2\ncommand: echo hi",
		},
		{
			name:        "legacy inline parameters without max_retries are unknown",
			run:         `{"run_id": 1, "run_type": "SUBMIT_RUN", "tasks": [{"gen_ai_compute_task": {"yaml_parameters": "command: echo hi\n"}}]}`,
			wantEnvCell: na, wantSource: new("yaml_parameters"), wantBody: "command: echo hi",
		},
		{
			name:        "legacy synthesized config is not exposed",
			run:         `{"run_id": 1, "run_type": "SUBMIT_RUN", "tasks": [{"gen_ai_compute_task": {"mlflow_experiment_name": "exp"}}]}`,
			wantEnvCell: na, wantBody: "experiment_name: exp",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch {
				case r.URL.Path == "/api/2.2/jobs/runs/get":
					_, _ = w.Write([]byte(tt.run))
				case strings.HasPrefix(r.URL.Path, "/api/2.0/workspace/export") && tt.config != "":
					_, _ = w.Write([]byte(tt.config))
				case r.URL.Path == "/api/2.0/mlflow/runs/get" && tt.mlflowName != "":
					_, _ = w.Write([]byte(`{"run": {"info": {"run_name": "` + tt.mlflowName + `"}}}`))
				default:
					w.WriteHeader(http.StatusNotFound)
					_, _ = w.Write([]byte(`{"error_code": "RESOURCE_DOES_NOT_EXIST", "message": "not found"}`))
				}
			}))
			t.Cleanup(srv.Close)
			w := newTestWorkspaceClient(t, srv.URL)
			ctx := withQuietEnrichment(cmdio.MockDiscard(t.Context()))
			run, err := w.Jobs.GetRun(ctx, jobs.GetRunRequest{RunId: 1})
			require.NoError(t, err)
			data := buildGetData(run)
			if genAIComputeTask(run) == nil {
				data.TrainingConfigPath = uploaded
			}

			enrichRunDetails(ctx, w, run, &data, &mlflowIdentifiers{ExperimentID: "e1", RunID: "r1"}, true)

			assert.Equal(t, tt.wantVersion, data.EnvironmentVersion)
			assert.Equal(t, tt.wantBase, data.BaseEnvironment)
			assert.Equal(t, tt.wantEnvCell, data.EnvironmentDisplay)
			assert.Equal(t, tt.wantSource, data.ConfigSource)
			assert.Equal(t, tt.wantSource == nil, data.Config == nil)
			assert.Equal(t, tt.wantRetries, data.MaxRetries)
			assert.Equal(t, tt.wantBody, data.configBody)
			if tt.mlflowName == "" {
				assert.Nil(t, data.MLflowRunName)
			} else {
				assert.Equal(t, tt.mlflowName, *data.MLflowRunName)
			}

			// JSON is null exactly when the text cell is N/A, and both render the
			// one resolved value.
			body, err := json.Marshal(data)
			require.NoError(t, err)
			var got struct {
				MaxRetries *runMaxRetries `json:"max_retries"`
			}
			require.NoError(t, json.Unmarshal(body, &got))
			assert.Equal(t, maxRetriesDisplay(data.MaxRetries) == na, got.MaxRetries == nil)
			assert.Equal(t, maxRetriesDisplay(data.MaxRetries), maxRetriesDisplay(got.MaxRetries))
		})
	}
}

func TestEnrichRunDetailsWithoutConfig(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/2.2/jobs/runs/get" {
			_, _ = w.Write([]byte(`{"run_id":1}`))
			return
		}
		if strings.HasPrefix(r.URL.Path, "/api/2.0/workspace/export") {
			t.Errorf("unexpected config download: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(srv.Close)
	w := newTestWorkspaceClient(t, srv.URL)
	run := &jobs.Run{RunId: 1, RunType: jobs.RunTypeSubmitRun, Tasks: []jobs.RunTask{{AiRuntimeTask: &jobs.AiRuntimeTask{}}}}
	data := buildGetData(run)
	data.TrainingConfigPath = "/Workspace/run/training_config.yaml"
	enrichRunDetails(withQuietEnrichment(t.Context()), w, run, &data, nil, false)
	assert.Nil(t, data.Config)
	assert.Nil(t, data.ConfigSource)
	assert.Nil(t, data.MaxRetries)

	// Inline legacy parameters must not be parsed either without opt-in.
	legacy := &jobs.Run{RunId: 1, RunType: jobs.RunTypeSubmitRun, Tasks: []jobs.RunTask{{
		GenAiComputeTask: &jobs.GenAiComputeTask{YamlParameters: "max_retries: 2\n"},
	}}}
	legacyData := buildGetData(legacy)
	enrichRunDetails(withQuietEnrichment(t.Context()), w, legacy, &legacyData, nil, false)
	assert.Nil(t, legacyData.Config)
	assert.Nil(t, legacyData.MaxRetries)

	// Server-reported limits remain available without reading config.
	legacy.Tasks[0].MaxRetries = 4
	enrichRunDetails(withQuietEnrichment(t.Context()), w, legacy, &legacyData, nil, false)
	assert.Equal(t, &runMaxRetries{Limit: 4, Source: maxRetriesReported}, legacyData.MaxRetries)
}

func TestSweepJSON(t *testing.T) {
	exp := "exp"
	body, err := json.Marshal(getData{RunID: "1", Sweep: &sweepInfo{
		Total: 2, Succeeded: 1, Active: 1, Completed: 1,
		Tasks: []sweepTask{
			{TaskKey: "t0", RunID: "10", Status: "SUCCESS", Experiment: exp, ExperimentName: &exp},
			{TaskKey: "t1", RunID: "11", Status: "RUNNING"},
		},
	}})
	require.NoError(t, err)
	var got struct {
		Sweep json.RawMessage `json:"sweep"`
	}
	require.NoError(t, json.Unmarshal(body, &got))
	assert.JSONEq(t, `{"total":2,"succeeded":1,"failed":0,"active":1,"completed":1,"tasks":[
		{"task_key":"t0","run_id":"10","status":"SUCCESS","experiment_name":"exp"},
		{"task_key":"t1","run_id":"11","status":"RUNNING","experiment_name":null}]}`, string(got.Sweep))
}

func TestAiRuntimeTaskOfSkipsNonAirTasks(t *testing.T) {
	run := &jobs.Run{RunId: 5, Tasks: []jobs.RunTask{
		notebookTask("prep", 51),
		aiRuntimeTask("train", 52, "/Users/me@example.com/exp", "GPU_1xA10", 1),
	}}
	task := aiRuntimeTaskOf(defaultTaskRun(run))
	require.NotNil(t, task)
	assert.Equal(t, "/Users/me@example.com/exp", task.Experiment)
}

// multiTaskGetServer serves runs/get for a job with a notebook prep task and two
// ai_runtime_tasks, and stubs every other request.
func multiTaskGetServer(t *testing.T) *httptest.Server {
	t.Helper()
	run := jobs.Run{
		RunId: 5,
		State: &jobs.RunState{LifeCycleState: jobs.RunLifeCycleStateTerminated, ResultState: jobs.RunResultStateSuccess},
		Tasks: []jobs.RunTask{
			notebookTask("prep", 51),
			aiRuntimeTask("train", 52, "/Users/me@example.com/exp-train", "GPU_1xA10", 1),
			aiRuntimeTask("eval", 53, "/Users/me@example.com/exp-eval", "GPU_8xH100", 8),
		},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/2.2/jobs/runs/get" {
			_ = json.NewEncoder(w).Encode(run)
			return
		}
		_, _ = w.Write([]byte(`{}`))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func runGetJSONTask(t *testing.T, w *databricks.WorkspaceClient, taskKey string) (string, error) {
	t.Helper()
	var buf bytes.Buffer
	ctx := cmdctx.SetWorkspaceClient(t.Context(), w)
	ctx = cmdio.InContext(ctx, cmdio.NewIO(ctx, flags.OutputJSON, nil, &buf, &buf, "", ""))
	cmd := withOutput(newGetCommand(), flags.OutputJSON)
	cmd.SetContext(ctx)
	if taskKey != "" {
		require.NoError(t, cmd.Flags().Set("task-key", taskKey))
	}
	err := cmd.RunE(cmd, []string{"5"})
	return buf.String(), err
}

func TestGetRunSelectsAirTask(t *testing.T) {
	srv := multiTaskGetServer(t)
	w, err := databricks.NewWorkspaceClient(&databricks.Config{Host: srv.URL, Token: "token", WorkspaceID: "67890"})
	require.NoError(t, err)

	type got struct {
		Data struct {
			TaskKey        *string `json:"task_key"`
			ExperimentName *string `json:"experiment_name"`
		} `json:"data"`
	}

	t.Run("defaults to the first AIR task, skipping the notebook", func(t *testing.T) {
		out, err := runGetJSONTask(t, w, "")
		require.NoError(t, err)
		var g got
		require.NoError(t, json.Unmarshal([]byte(out), &g))
		require.NotNil(t, g.Data.TaskKey)
		assert.Equal(t, "train", *g.Data.TaskKey)
		require.NotNil(t, g.Data.ExperimentName)
		assert.Equal(t, "exp-train", *g.Data.ExperimentName)
	})

	t.Run("--task-key selects a later AIR task", func(t *testing.T) {
		out, err := runGetJSONTask(t, w, "eval")
		require.NoError(t, err)
		var g got
		require.NoError(t, json.Unmarshal([]byte(out), &g))
		require.NotNil(t, g.Data.TaskKey)
		assert.Equal(t, "eval", *g.Data.TaskKey)
		require.NotNil(t, g.Data.ExperimentName)
		assert.Equal(t, "exp-eval", *g.Data.ExperimentName)
	})

	t.Run("--task-key must name an AIR task", func(t *testing.T) {
		out, err := runGetJSONTask(t, w, "prep")
		require.ErrorIs(t, err, root.ErrAlreadyPrinted)
		var e errorEnvelope
		require.NoError(t, json.Unmarshal([]byte(out), &e))
		assert.Equal(t, jsonError{Code: "INVALID_ARGS", Kind: "PERMANENT", Message: `run 5 has no AI Runtime task "prep"; its AI Runtime tasks are: train, eval`}, e.Error)
	})
}

func TestSelectAirTaskErrors(t *testing.T) {
	_, err := selectAirTask(&jobs.Run{RunId: 5, Tasks: []jobs.RunTask{notebookTask("prep", 51)}}, "prep")
	assert.EqualError(t, err, "run 5 has no AI Runtime tasks")

	two := &jobs.Run{RunId: 7, Tasks: []jobs.RunTask{
		notebookTask("prep", 70),
		aiRuntimeTask("train", 71, "e", "GPU_1xA10", 1),
	}}
	_, err = selectAirTask(two, "prep")
	assert.EqualError(t, err, `run 7 has no AI Runtime task "prep"; its AI Runtime tasks are: train`)

	run, err := selectAirTask(&jobs.Run{RunId: 8, Tasks: []jobs.RunTask{notebookTask("prep", 80)}}, "")
	require.NoError(t, err)
	assert.Len(t, run.Tasks, 1, "a run without an AI Runtime task is shown as is")
}
