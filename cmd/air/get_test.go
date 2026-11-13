package aircmd

import (
	"bytes"
	"encoding/json"
	"errors"
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
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/databricks/databricks-sdk-go/service/iam"
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

func TestGetRunAuthFailed(t *testing.T) {
	m := mocks.NewMockWorkspaceClient(t)
	// GetRun's 403 is confirmed as a credential failure by the Me() probe, so
	// nothing is rendered.
	m.GetMockJobsAPI().EXPECT().GetRun(mock.Anything, mock.Anything).Return(nil, apierr.ErrPermissionDenied)
	m.GetMockCurrentUserAPI().EXPECT().Me(mock.Anything, mock.Anything).Return(nil, apierr.ErrPermissionDenied)
	ctx := cmdctx.SetWorkspaceClient(cmdio.MockDiscard(t.Context()), m.WorkspaceClient)
	cmd := withOutput(newGetCommand(), flags.OutputText)
	cmd.SetContext(ctx)

	err := cmd.RunE(cmd, []string{"5"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "authentication was not successful")
}

func TestGetRunAuthTransient(t *testing.T) {
	m := mocks.NewMockWorkspaceClient(t)
	// A transient failure at the auth probe must not be misreported as an auth
	// error; it surfaces as a retryable internal error instead.
	m.GetMockJobsAPI().EXPECT().GetRun(mock.Anything, mock.Anything).Return(nil, apierr.ErrUnauthenticated)
	m.GetMockCurrentUserAPI().EXPECT().Me(mock.Anything, mock.Anything).Return(nil, errors.New("connection reset"))
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
	// A 403 on the run with credentials the Me() probe accepts is missing access
	// to the run, not a failed login.
	m.GetMockJobsAPI().EXPECT().GetRun(mock.Anything, mock.Anything).Return(nil, apierr.ErrPermissionDenied)
	m.GetMockCurrentUserAPI().EXPECT().Me(mock.Anything, mock.Anything).Return(&iam.User{UserName: "u@example.com"}, nil)
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

// getRunServer serves a terminal runs/get and a /Me that reports workspace
// 12345 in the X-Databricks-Org-Id header. *meCalls counts /Me requests.
func getRunServer(t *testing.T, meCalls *atomic.Int32) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/2.2/jobs/runs/get":
			_, _ = w.Write([]byte(`{"run_id": 5, "state": {"life_cycle_state": "TERMINATED", "result_state": "SUCCESS"}}`))
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
	srv := getRunServer(t, &meCalls)
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
	srv := getRunServer(t, &meCalls)

	out, err := runGetJSON(t, newTestWorkspaceClient(t, srv.URL))
	require.NoError(t, err)
	assert.Contains(t, out, "/jobs/runs/5?o=12345")
	assert.Equal(t, int32(1), meCalls.Load())
}

func TestGetRunRejectsNonNumericWorkspaceID(t *testing.T) {
	var meCalls atomic.Int32
	srv := getRunServer(t, &meCalls)
	w, err := databricks.NewWorkspaceClient(&databricks.Config{Host: srv.URL, Token: "token", WorkspaceID: "not-a-number"})
	require.NoError(t, err)

	out, err := runGetJSON(t, w)
	require.ErrorIs(t, err, root.ErrAlreadyPrinted)
	var got errorEnvelope
	require.NoError(t, json.Unmarshal([]byte(out), &got))
	assert.Equal(t, jsonError{Code: "INVALID_ARGS", Kind: "PERMANENT", Message: `configured workspace_id "not-a-number" is not a numeric workspace ID`}, got.Error)
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
		"run_id":"5","status":"FAILED","started_at":null,"duration_seconds":null,
		"attempt_number":0,"experiment_name":null,"dashboard_url":"","mlflow_url":null,
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
		"run_id":"8","status":"PENDING","started_at":null,"duration_seconds":null,
		"attempt_number":0,"experiment_name":null,"dashboard_url":"","mlflow_url":null,
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
			wantBase: new("workspace-base-environments/databricks_ai_v5"), wantEnvCell: na,
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
