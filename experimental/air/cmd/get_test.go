package aircmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"text/template"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
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
// `air get JOB_RUN_ID` and has no `run` subcommand (it was collapsed back into
// `get`). The acceptance test exercises the happy path end to end.
func TestGetCommandShape(t *testing.T) {
	cmd := newGetCommand()
	assert.Equal(t, "get JOB_RUN_ID", cmd.Use)
	assert.Empty(t, cmd.Commands(), "get must not register subcommands")
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

// notFoundGetServer serves the auth probe plus a runs/get that reports the run
// as missing (400 INVALID_PARAMETER_VALUE, which the SDK maps to
// ErrResourceDoesNotExist for this path).
func notFoundGetServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/2.2/jobs/runs/get" {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error_code":"INVALID_PARAMETER_VALUE","message":"Run 5 does not exist."}`))
			return
		}
		// Me() probe and any other config discovery.
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
	// A genuine auth failure (permission denied) is validated before the run is
	// fetched, so GetRun is never reached and nothing is rendered.
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
	m.GetMockCurrentUserAPI().EXPECT().Me(mock.Anything, mock.Anything).Return(nil, errors.New("connection reset"))
	ctx := cmdctx.SetWorkspaceClient(cmdio.MockDiscard(t.Context()), m.WorkspaceClient)
	cmd := withOutput(newGetCommand(), flags.OutputText)
	cmd.SetContext(ctx)

	err := cmd.RunE(cmd, []string{"5"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to verify authentication")
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
		"termination_reason":"out of memory","est_remaining_seconds":null,"est_percent_complete":null
	}`, string(body))

	canceled := buildGetData(&jobs.Run{RunId: 6, State: &jobs.RunState{ResultState: jobs.RunResultStateCanceled, StateMessage: "user canceled"}})
	body, err = json.Marshal(canceled)
	require.NoError(t, err)
	assert.NotContains(t, string(body), "termination_reason")
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
