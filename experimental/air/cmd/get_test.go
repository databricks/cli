package aircmd

import (
	"bytes"
	"io"
	"strings"
	"testing"
	"text/template"

	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
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
		RunID:        "123",
		Status:       "RUNNING",
		User:         "me@example.com",
		DashboardURL: "https://example.test/run/123",
	})
	assert.Contains(t, out, "Run ID:       123")
	assert.Contains(t, out, "Status:       RUNNING")
	assert.Contains(t, out, "User:")
	assert.Contains(t, out, "me@example.com")
	assert.Contains(t, out, "Dashboard:    https://example.test/run/123")
	assert.NotContains(t, out, "Sweep")
}

func TestGetRunInvalidID(t *testing.T) {
	m := mocks.NewMockWorkspaceClient(t)
	ctx := cmdctx.SetWorkspaceClient(cmdio.MockDiscard(t.Context()), m.WorkspaceClient)
	cmd := newGetCommand()
	cmd.SetContext(ctx)

	err := cmd.RunE(cmd, []string{"abc"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid RUN_ID")
}

func TestGetRunNotFound(t *testing.T) {
	m := mocks.NewMockWorkspaceClient(t)
	m.GetMockJobsAPI().EXPECT().GetRun(mock.Anything, jobs.GetRunRequest{RunId: 5}).Return(
		nil, apierr.ErrResourceDoesNotExist)
	ctx := cmdctx.SetWorkspaceClient(cmdio.MockDiscard(t.Context()), m.WorkspaceClient)
	cmd := newGetCommand()
	cmd.SetContext(ctx)

	err := cmd.RunE(cmd, []string{"5"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "run 5 not found")
}

func TestPrintConfigYAML(t *testing.T) {
	t.Run("downloads and prints", func(t *testing.T) {
		ctx := cmdio.MockDiscard(t.Context())
		m := mocks.NewMockWorkspaceClient(t)
		// The mock asserts Download is called with the resolved path.
		m.GetMockWorkspaceAPI().EXPECT().
			Download(mock.Anything, "/Workspace/cfg.yaml").
			Return(io.NopCloser(strings.NewReader("epochs: 3\n")), nil)

		printConfigYAML(ctx, m.WorkspaceClient, "/Workspace/cfg.yaml")
	})

	t.Run("download failure is non-fatal", func(t *testing.T) {
		ctx := cmdio.MockDiscard(t.Context())
		m := mocks.NewMockWorkspaceClient(t)
		m.GetMockWorkspaceAPI().EXPECT().
			Download(mock.Anything, "/Workspace/missing.yaml").
			Return(nil, apierr.ErrResourceDoesNotExist)

		// Must not panic: a failed config fetch is best-effort.
		printConfigYAML(ctx, m.WorkspaceClient, "/Workspace/missing.yaml")
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
	// Only the always-present rows render; optional rows are hidden when empty.
	out := renderGet(t, getData{RunID: "1", Status: "PENDING", DashboardURL: "https://example.test/1"})
	assert.Contains(t, out, "Run ID:       1")
	assert.Contains(t, out, "Status:       PENDING")
	assert.Contains(t, out, "Retries:      0")
	assert.Contains(t, out, "Dashboard:    https://example.test/1")
	for _, hidden := range []string{"Submitted:", "Duration:", "Experiment:", "User:", "Accelerators:", "MLflow:"} {
		assert.NotContains(t, out, hidden)
	}
}

func TestGetTemplateAllFields(t *testing.T) {
	started := "2023-11-14T22:13:20Z"
	exp := "exp"
	mlflow := "https://example.test/ml/exp/1"
	out := renderGet(t, getData{
		RunID:          "1",
		Status:         "SUCCESS",
		StartedAt:      &started,
		Duration:       "12s",
		AttemptNumber:  2,
		ExperimentName: &exp,
		User:           "me@example.com",
		Accelerators:   "8x H100",
		MLflowURL:      &mlflow,
		DashboardURL:   "https://example.test/1",
	})
	for _, want := range []string{
		"Submitted:    2023-11-14T22:13:20Z",
		"Duration:     12s",
		"Retries:      2",
		"Experiment:   exp",
		"User:         me@example.com",
		"Accelerators: 8x H100",
		"MLflow:       https://example.test/ml/exp/1",
		"Dashboard:    https://example.test/1",
	} {
		assert.Contains(t, out, want)
	}
}

func TestBuildGetData(t *testing.T) {
	run := &jobs.Run{
		RunId:           123,
		RunPageUrl:      "https://example.test/run/123",
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
	assert.Equal(t, "https://example.test/run/123", d.DashboardURL)
	assert.Equal(t, "me@example.com", d.User)
	assert.Equal(t, "8x H100", d.Accelerators)
	assert.Equal(t, "12s", d.Duration)
	require.NotNil(t, d.ExperimentName)
	assert.Equal(t, "exp", *d.ExperimentName)
	require.NotNil(t, d.DurationSeconds)
	assert.Equal(t, int64(12), *d.DurationSeconds)
}
