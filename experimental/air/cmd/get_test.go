package aircmd

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"
	"testing"
	"text/template"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// renderGet renders the get template against the JSON envelope, exactly as
// the command does, so the test covers the real template branches.
func renderGet(t *testing.T, data getData) string {
	t.Helper()
	tmpl, err := template.New("get").Parse(getTemplate)
	require.NoError(t, err)
	var buf bytes.Buffer
	require.NoError(t, tmpl.Execute(&buf, envelope{V: envelopeVersion, Data: data}))
	return buf.String()
}

func TestGetTemplateSingleRun(t *testing.T) {
	out := renderGet(t, getData{
		RunIDDisplay: "123",
		Status:       "RUNNING",
		UserDisplay:  "me@example.com",
	})
	assert.Contains(t, out, "Run ID:       123")
	assert.Contains(t, out, "Status:       RUNNING")
	assert.Contains(t, out, "User:         me@example.com")
	assert.NotContains(t, out, "Sweep")
	// Python embeds the dashboard link on the Run ID; there is no Dashboard row.
	assert.NotContains(t, out, "Dashboard")
}

func TestGetRunInvalidID(t *testing.T) {
	m := mocks.NewMockWorkspaceClient(t)
	ctx := cmdctx.SetWorkspaceClient(cmdio.MockDiscard(t.Context()), m.WorkspaceClient)
	cmd := withOutput(newGetRunCommand(), flags.OutputText)
	cmd.SetContext(ctx)

	err := cmd.RunE(cmd, []string{"abc"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid JOB_RUN_ID")
}

func TestGetRunNotFound(t *testing.T) {
	m := mocks.NewMockWorkspaceClient(t)
	m.GetMockJobsAPI().EXPECT().GetRun(mock.Anything, jobs.GetRunRequest{RunId: 5}).Return(
		nil, apierr.ErrResourceDoesNotExist)
	ctx := cmdctx.SetWorkspaceClient(cmdio.MockDiscard(t.Context()), m.WorkspaceClient)
	cmd := withOutput(newGetRunCommand(), flags.OutputText)
	cmd.SetContext(ctx)

	err := cmd.RunE(cmd, []string{"5"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "run 5 not found")
}

func TestGetRunNotFoundJSON(t *testing.T) {
	var buf bytes.Buffer
	m := mocks.NewMockWorkspaceClient(t)
	m.GetMockJobsAPI().EXPECT().GetRun(mock.Anything, jobs.GetRunRequest{RunId: 5}).Return(
		nil, apierr.ErrResourceDoesNotExist)
	ctx := cmdctx.SetWorkspaceClient(t.Context(), m.WorkspaceClient)
	ctx = cmdio.InContext(ctx, cmdio.NewIO(ctx, flags.OutputJSON, nil, &buf, &buf, "", ""))
	cmd := withOutput(newGetRunCommand(), flags.OutputJSON)
	cmd.SetContext(ctx)

	// In JSON mode the not-found error is a structured envelope, not a bare error.
	err := cmd.RunE(cmd, []string{"5"})
	require.ErrorIs(t, err, root.ErrAlreadyPrinted)

	var got errorEnvelope
	require.NoError(t, json.Unmarshal(buf.Bytes(), &got))
	assert.Equal(t, jsonError{Code: "NOT_FOUND", Kind: "NOT_FOUND", Message: "run 5 not found: check the run ID and that it is a job run ID"}, got.Error)
}

func TestPrintConfigYAML(t *testing.T) {
	t.Run("downloads and prints to stdout", func(t *testing.T) {
		ctx := cmdio.MockDiscard(t.Context())
		m := mocks.NewMockWorkspaceClient(t)
		// The mock asserts Download is called with the resolved path.
		m.GetMockWorkspaceAPI().EXPECT().
			Download(mock.Anything, "/Workspace/cfg.yaml").
			Return(io.NopCloser(strings.NewReader("epochs: 3\ncommand: \"set -e\\npython train.py\\n\"\n")), nil)

		// The config is data output and must land on stdout (the out writer),
		// matching the Python `air get` behavior.
		var out bytes.Buffer
		printConfigYAML(ctx, &out, m.WorkspaceClient, "/Workspace/cfg.yaml")
		assert.Contains(t, out.String(), "Training Configuration:")
		assert.Contains(t, out.String(), "epochs: 3")
		// The multi-line command is reformatted to a `|` block literal.
		assert.Contains(t, out.String(), "command: |\n  set -e\n  python train.py")
	})

	t.Run("download failure is non-fatal and writes nothing", func(t *testing.T) {
		ctx := cmdio.MockDiscard(t.Context())
		m := mocks.NewMockWorkspaceClient(t)
		m.GetMockWorkspaceAPI().EXPECT().
			Download(mock.Anything, "/Workspace/missing.yaml").
			Return(nil, apierr.ErrResourceDoesNotExist)

		// Must not panic and must not write to stdout: a failed config fetch is
		// best-effort, surfaced only as a stderr warning.
		var out bytes.Buffer
		printConfigYAML(ctx, &out, m.WorkspaceClient, "/Workspace/missing.yaml")
		assert.Empty(t, out.String())
	})
}

func TestYAMLConfigPath(t *testing.T) {
	// No tasks, or a task without GenAiComputeTask, yields no path.
	assert.Empty(t, yamlConfigPath(&jobs.Run{}))
	assert.Empty(t, yamlConfigPath(&jobs.Run{Tasks: []jobs.RunTask{{}}}))

	run := &jobs.Run{Tasks: []jobs.RunTask{{
		GenAiComputeTask: &jobs.GenAiComputeTask{YamlParametersFilePath: "/Workspace/cfg.yaml"},
	}}}
	assert.Equal(t, "/Workspace/cfg.yaml", yamlConfigPath(run))
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
	// The single-run rows must not appear in the sweep view.
	assert.NotContains(t, out, "Dashboard:")
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

func TestGetTemplateMinimal(t *testing.T) {
	// Every row always renders; missing values show "N/A", in Python's order.
	out := renderGet(t, getData{
		RunIDDisplay:        "1",
		Status:              "PENDING",
		SubmittedDisplay:    na,
		DurationDisplay:     na,
		ExperimentDisplay:   na,
		MLflowDisplay:       na,
		UserDisplay:         na,
		AcceleratorsDisplay: na,
	})
	for _, want := range []string{
		"Run ID:       1",
		"Status:       PENDING",
		"Submitted:    N/A",
		"Retries:      0",
		"Duration:     N/A",
		"Experiment:   N/A",
		"MLflow Run:   N/A",
		"User:         N/A",
		"Accelerators: N/A",
	} {
		assert.Contains(t, out, want)
	}
}

func TestGetTemplateAllFields(t *testing.T) {
	out := renderGet(t, getData{
		RunIDDisplay:        "1",
		Status:              "SUCCESS",
		SubmittedDisplay:    "2023-11-14 22:13 UTC",
		DurationDisplay:     "12s",
		AttemptNumber:       2,
		ExperimentDisplay:   "exp",
		MLflowDisplay:       "sunny-cat-42",
		UserDisplay:         "me@example.com",
		AcceleratorsDisplay: "8x H100",
	})
	for _, want := range []string{
		"Run ID:       1",
		"Status:       SUCCESS",
		"Submitted:    2023-11-14 22:13 UTC",
		"Retries:      2",
		"Duration:     12s",
		"Experiment:   exp",
		"MLflow Run:   sunny-cat-42",
		"User:         me@example.com",
		"Accelerators: 8x H100",
	} {
		assert.Contains(t, out, want)
	}
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
	assert.Equal(t, "123", d.RunIDDisplay)
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

func TestBuildGetDataEmpty(t *testing.T) {
	// A run with no tasks, creator, or timing renders every text cell as "N/A".
	d := buildGetData(&jobs.Run{RunId: 7})
	assert.Equal(t, "7", d.RunIDDisplay)
	assert.Equal(t, na, d.SubmittedDisplay)
	assert.Equal(t, na, d.DurationDisplay)
	assert.Equal(t, na, d.ExperimentDisplay)
	assert.Equal(t, na, d.MLflowDisplay)
	assert.Equal(t, na, d.UserDisplay)
	assert.Equal(t, na, d.AcceleratorsDisplay)
}
