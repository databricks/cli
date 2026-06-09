package ai

import (
	"testing"

	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestFindForEachTask(t *testing.T) {
	// No tasks at all.
	assert.Nil(t, findForEachTask(&jobs.Run{}))

	// A task that is not a foreach.
	assert.Nil(t, findForEachTask(&jobs.Run{Tasks: []jobs.RunTask{{TaskKey: "a"}}}))

	// The foreach task is found even when it isn't first.
	run := &jobs.Run{Tasks: []jobs.RunTask{
		{TaskKey: "a"},
		{TaskKey: "sweep", ForEachTask: &jobs.RunForEachTask{}},
	}}
	got := findForEachTask(run)
	require.NotNil(t, got)
	assert.Equal(t, "sweep", got.TaskKey)
}

func sweepTaskFixture() *jobs.RunTask {
	return &jobs.RunTask{
		RunId: 99,
		ForEachTask: &jobs.RunForEachTask{
			Stats: &jobs.ForEachStats{TaskRunStats: &jobs.ForEachTaskTaskRunStats{
				TotalIterations:     4,
				SucceededIterations: 1,
				FailedIterations:    1,
				ActiveIterations:    2,
				CompletedIterations: 2,
			}},
		},
	}
}

func TestBuildSweepInfo(t *testing.T) {
	ctx := t.Context()

	t.Run("counts and iteration rows", func(t *testing.T) {
		m := mocks.NewMockWorkspaceClient(t)
		m.GetMockJobsAPI().EXPECT().GetRun(mock.Anything, jobs.GetRunRequest{RunId: 99}).Return(
			&jobs.Run{Iterations: []jobs.RunTask{{
				TaskKey:          "iter_0",
				RunId:            100,
				State:            &jobs.RunState{ResultState: jobs.RunResultStateSuccess},
				GenAiComputeTask: &jobs.GenAiComputeTask{MlflowExperimentName: "/Users/me@example.com/exp"},
			}}}, nil)

		info := buildSweepInfo(ctx, m.WorkspaceClient, sweepTaskFixture())
		assert.Equal(t, 4, info.Total)
		assert.Equal(t, 2, info.Completed)
		assert.Equal(t, 1, info.Succeeded)
		assert.Equal(t, 1, info.Failed)
		assert.Equal(t, 2, info.Active)
		require.Len(t, info.Tasks, 1)
		assert.Equal(t, "iter_0", info.Tasks[0].TaskKey)
		assert.Equal(t, "100", info.Tasks[0].RunID)
		assert.Equal(t, "SUCCESS", info.Tasks[0].Status)
		assert.Equal(t, "exp", info.Tasks[0].Experiment)
	})

	t.Run("iteration lookup failure still returns counts", func(t *testing.T) {
		m := mocks.NewMockWorkspaceClient(t)
		m.GetMockJobsAPI().EXPECT().GetRun(mock.Anything, jobs.GetRunRequest{RunId: 99}).Return(
			nil, apierr.ErrResourceDoesNotExist)

		info := buildSweepInfo(ctx, m.WorkspaceClient, sweepTaskFixture())
		assert.Equal(t, 4, info.Total)
		assert.Empty(t, info.Tasks)
	})
}
