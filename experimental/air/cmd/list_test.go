package aircmd

import (
	"bytes"
	"strconv"
	"testing"
	"text/template"

	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/databricks/databricks-sdk-go/listing"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// renderList renders the list template against the JSON envelope, exactly as
// the command does, so the test covers the real template branches.
func renderList(t *testing.T, data listData) string {
	t.Helper()
	tmpl, err := template.New("list").Parse(listTemplate)
	require.NoError(t, err)
	var buf bytes.Buffer
	require.NoError(t, tmpl.Execute(&buf, envelope{V: envelopeVersion, Data: data}))
	return buf.String()
}

// airBaseRun builds a single-task AIR run for a given user and compute.
func airBaseRun(id int64, user, gpuType string, numGpus int, experiment string) jobs.BaseRun {
	return jobs.BaseRun{
		RunId:           id,
		RunName:         "run-" + strconv.FormatInt(id, 10),
		CreatorUserName: user,
		State:           &jobs.RunState{LifeCycleState: jobs.RunLifeCycleStateRunning},
		Tasks: []jobs.RunTask{{
			GenAiComputeTask: &jobs.GenAiComputeTask{
				TrainingScriptPath:   "/Workspace/train.py",
				MlflowExperimentName: experiment,
				Compute:              &jobs.ComputeConfig{GpuType: gpuType, NumGpus: numGpus},
			},
		}},
	}
}

// sweepBaseRun builds a foreach (sweep) AIR run.
func sweepBaseRun(id int64, user, experiment string) jobs.BaseRun {
	return jobs.BaseRun{
		RunId:           id,
		CreatorUserName: user,
		State:           &jobs.RunState{LifeCycleState: jobs.RunLifeCycleStateRunning},
		Tasks: []jobs.RunTask{{
			ForEachTask: &jobs.RunForEachTask{
				Task: jobs.Task{
					GenAiComputeTask: &jobs.GenAiComputeTask{
						TrainingScriptPath:   "/Workspace/train.py",
						MlflowExperimentName: experiment,
					},
				},
			},
		}},
	}
}

func TestListTemplate(t *testing.T) {
	started := "2023-11-14T22:13:20Z"
	out := renderList(t, listData{Rows: []listRow{
		{
			RunID: "1", Status: "RUNNING", User: "me@example.com", StartedAt: &started,
			Experiment: "exp-a", Duration: "12s", MLflowURL: "https://ml/1", Accelerators: "8x H100",
		},
		// A still-running, just-started run with no optional data: every optional
		// cell, including Started, falls back to "-".
		{
			RunID: "2", Status: "PENDING", User: "me@example.com",
			Experiment: "-", Duration: "-", MLflowURL: "-", Accelerators: "-",
		},
	}})

	for _, want := range []string{"Run ID", "Experiment", "Status", "Started", "Accelerators"} {
		assert.Contains(t, out, want)
	}
	assert.Contains(t, out, "exp-a")
	assert.Contains(t, out, "8x H100")
	assert.Contains(t, out, "2023-11-14T22:13:20Z")
	// The second row has no start time and renders a dash.
	assert.Contains(t, out, "2\t-\tPENDING\t-\t")
}

func TestListAirRunsFiltersUserAndType(t *testing.T) {
	runs := []jobs.BaseRun{
		airBaseRun(1, "me@example.com", "GPU_8xH100", 8, "/Users/me@example.com/exp-a"),
		{RunId: 2, CreatorUserName: "me@example.com", Tasks: []jobs.RunTask{{}}}, // not an AIR run
		airBaseRun(3, "other@example.com", "GPU_1xA10", 1, "/Users/other/exp-b"), // wrong user
		sweepBaseRun(4, "me@example.com", "/Users/me@example.com/sweep"),
		airBaseRun(5, "me@example.com", "GPU_1xH100", 1, "/Users/me@example.com/exp-c"),
	}
	iter := listing.SliceIterator[jobs.BaseRun](runs)

	m := mocks.NewMockWorkspaceClient(t)
	// Limit is left at zero: filtering and truncation happen client-side.
	m.GetMockJobsAPI().EXPECT().ListRuns(mock.Anything, jobs.ListRunsRequest{
		ExpandTasks: true,
		RunType:     jobs.RunTypeSubmitRun,
	}).Return(&iter)

	entries, err := listAirRuns(t.Context(), m.WorkspaceClient, listQuery{
		userFilter: "me@example.com",
		limit:      10,
	})
	require.NoError(t, err)

	require.Len(t, entries, 3)
	assert.Equal(t, "1", entries[0].row.RunID)
	assert.Equal(t, "4", entries[1].row.RunID)
	assert.True(t, entries[1].row.IsSweep, "run 4 is a foreach sweep")
	assert.Equal(t, "5", entries[2].row.RunID)
}

func TestListAirRunsLimitTruncates(t *testing.T) {
	runs := []jobs.BaseRun{
		airBaseRun(1, "me@example.com", "GPU_1xH100", 1, "exp-a"),
		airBaseRun(2, "me@example.com", "GPU_1xH100", 1, "exp-b"),
		airBaseRun(3, "me@example.com", "GPU_1xH100", 1, "exp-c"),
	}
	iter := listing.SliceIterator[jobs.BaseRun](runs)

	m := mocks.NewMockWorkspaceClient(t)
	m.GetMockJobsAPI().EXPECT().ListRuns(mock.Anything, mock.Anything).Return(&iter)

	entries, err := listAirRuns(t.Context(), m.WorkspaceClient, listQuery{limit: 2})
	require.NoError(t, err)
	require.Len(t, entries, 2)
	assert.Equal(t, "1", entries[0].row.RunID)
	assert.Equal(t, "2", entries[1].row.RunID)
}

func TestListAirRunsExperimentFilter(t *testing.T) {
	runs := []jobs.BaseRun{
		airBaseRun(1, "me@example.com", "GPU_1xH100", 1, "/Users/me@example.com/qwen-train"),
		airBaseRun(2, "me@example.com", "GPU_1xH100", 1, "/Users/me@example.com/llama-train"),
	}
	iter := listing.SliceIterator[jobs.BaseRun](runs)

	m := mocks.NewMockWorkspaceClient(t)
	m.GetMockJobsAPI().EXPECT().ListRuns(mock.Anything, mock.Anything).Return(&iter)

	entries, err := listAirRuns(t.Context(), m.WorkspaceClient, listQuery{
		limit:   10,
		filters: listFilters{Experiment: "qwen*"},
	})
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, "1", entries[0].row.RunID)
}

func TestBuildListRow(t *testing.T) {
	run := baseRunToRun(&jobs.BaseRun{
		RunId:           123,
		CreatorUserName: "me@example.com",
		StartTime:       1700000000000,
		EndTime:         1700000012000,
		State:           &jobs.RunState{ResultState: jobs.RunResultStateSuccess},
		Tasks: []jobs.RunTask{{
			GenAiComputeTask: &jobs.GenAiComputeTask{
				MlflowExperimentName: "/Users/me@example.com/exp",
				Compute:              &jobs.ComputeConfig{NumGpus: 8, GpuType: "GPU_8xH100"},
			},
		}},
	})

	row := buildListRow(run)
	assert.Equal(t, "123", row.RunID)
	assert.Equal(t, "me@example.com", row.User)
	assert.Equal(t, "SUCCESS", row.Status)
	assert.Equal(t, "exp", row.Experiment)
	assert.Equal(t, "12s", row.Duration)
	assert.Equal(t, "8x H100", row.Accelerators)
	assert.Equal(t, "-", row.MLflowURL)
	assert.False(t, row.IsSweep)
	require.NotNil(t, row.StartedAt)
}

func TestBuildListRowDashFallbacks(t *testing.T) {
	// A run with no experiment, compute, or start time falls back to dashes for
	// the optional columns.
	row := buildListRow(&jobs.Run{RunId: 7, State: &jobs.RunState{LifeCycleState: jobs.RunLifeCycleStateRunning}})
	assert.Equal(t, "-", row.Experiment)
	assert.Equal(t, "-", row.Duration)
	assert.Equal(t, "-", row.Accelerators)
	assert.Nil(t, row.StartedAt)
}

func TestListInvalidLimit(t *testing.T) {
	m := mocks.NewMockWorkspaceClient(t)
	ctx := cmdctx.SetWorkspaceClient(cmdio.MockDiscard(t.Context()), m.WorkspaceClient)
	cmd := newListCommand()
	cmd.SetContext(ctx)
	require.NoError(t, cmd.Flags().Set("limit", "0"))

	err := cmd.RunE(cmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid --limit")
}

func TestListInvalidFilter(t *testing.T) {
	m := mocks.NewMockWorkspaceClient(t)
	ctx := cmdctx.SetWorkspaceClient(cmdio.MockDiscard(t.Context()), m.WorkspaceClient)
	cmd := newListCommand()
	cmd.SetContext(ctx)
	require.NoError(t, cmd.Flags().Set("filter", "bogus=1"))

	err := cmd.RunE(cmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported --filter key")
}
