package aircmd

import (
	"testing"

	"github.com/databricks/cli/libs/apierrs"
	"github.com/databricks/cli/libs/sdktypes/jobs/v2"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestFindForEachTask(t *testing.T) {
	// No tasks at all.
	assert.Nil(t, findForEachTask(&jobs.GetRunResponse{}))

	// A task that is not a foreach.
	assert.Nil(t, findForEachTask(&jobs.GetRunResponse{Tasks: []jobs.RunTask{{TaskKey: new("a")}}}))

	// The foreach task is found even when it isn't first.
	run := &jobs.GetRunResponse{Tasks: []jobs.RunTask{
		{TaskKey: new("a")},
		{TaskKey: new("sweep"), ForEachTask: &jobs.RunForEachTask{}},
	}}
	got := findForEachTask(run)
	require.NotNil(t, got)
	assert.Equal(t, "sweep", got.TaskKey)
}

func sweepTaskFixture() *jobs.RunTask {
	return &jobs.RunTask{
		RunId: new(int64(99)),
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
		m.GetMockJobsAPI().EXPECT().GetRun(mock.Anything, jobs.GetRunRequest{RunId: new(int64(99))}).Return(
			&jobs.GetRunResponse{Iterations: []jobs.RunTask{{
				TaskKey:          new("iter_0"),
				RunId:            new(int64(100)),
				State:            &jobs.RunState{ResultState: jobs.RunResultState_RunResultState_Success},
				GenAiComputeTask: &jobs.GenAiComputeTask{MlflowExperimentName: new("/Users/me@example.com/exp")},
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
		m.GetMockJobsAPI().EXPECT().GetRun(mock.Anything, jobs.GetRunRequest{RunId: new(int64(99))}).Return(
			nil, apierrs.ErrResourceDoesNotExist)

		info := buildSweepInfo(ctx, m.WorkspaceClient, sweepTaskFixture())
		assert.Equal(t, 4, info.Total)
		assert.Empty(t, info.Tasks)
	})
}
