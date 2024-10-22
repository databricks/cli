package run

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestConvertPythonParams(t *testing.T) {
	job := &resources.Job{
		JobSettings: &jobs.JobSettings{
			Tasks: []jobs.Task{
				{PythonWheelTask: &jobs.PythonWheelTask{
					PackageName: "my_test_code",
					EntryPoint:  "run",
				}},
			},
		},
	}
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"test_job": job,
				},
			},
		},
	}
	runner := jobRunner{key: "test", bundle: b, job: job}

	opts := &Options{
		Job: JobOptions{},
	}
	runner.convertPythonParams(opts)
	require.NotContains(t, opts.Job.notebookParams, "__python_params")

	opts = &Options{
		Job: JobOptions{
			pythonParams: []string{"param1", "param2", "param3"},
		},
	}
	runner.convertPythonParams(opts)
	require.Contains(t, opts.Job.notebookParams, "__python_params")
	require.Equal(t, opts.Job.notebookParams["__python_params"], `["param1","param2","param3"]`)
}

func TestJobRunnerCancel(t *testing.T) {
	job := &resources.Job{
		ID: "123",
	}
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"test_job": job,
				},
			},
		},
	}

	runner := jobRunner{key: "test", bundle: b, job: job}

	m := mocks.NewMockWorkspaceClient(t)
	b.SetWorkpaceClient(m.WorkspaceClient)

	jobApi := m.GetMockJobsAPI()
	jobApi.EXPECT().ListRunsAll(mock.Anything, jobs.ListRunsRequest{
		ActiveOnly: true,
		JobId:      123,
	}).Return([]jobs.BaseRun{
		{RunId: 1},
		{RunId: 2},
	}, nil)

	mockWait := &jobs.WaitGetRunJobTerminatedOrSkipped[struct{}]{
		Poll: func(time time.Duration, f func(j *jobs.Run)) (*jobs.Run, error) {
			return nil, nil
		},
	}
	jobApi.EXPECT().CancelRun(mock.Anything, jobs.CancelRun{
		RunId: 1,
	}).Return(mockWait, nil)
	jobApi.EXPECT().CancelRun(mock.Anything, jobs.CancelRun{
		RunId: 2,
	}).Return(mockWait, nil)

	err := runner.Cancel(context.Background())
	require.NoError(t, err)
}

func TestJobRunnerCancelWithNoActiveRuns(t *testing.T) {
	job := &resources.Job{
		ID: "123",
	}
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"test_job": job,
				},
			},
		},
	}

	runner := jobRunner{key: "test", bundle: b, job: job}

	m := mocks.NewMockWorkspaceClient(t)
	b.SetWorkpaceClient(m.WorkspaceClient)

	jobApi := m.GetMockJobsAPI()
	jobApi.EXPECT().ListRunsAll(mock.Anything, jobs.ListRunsRequest{
		ActiveOnly: true,
		JobId:      123,
	}).Return([]jobs.BaseRun{}, nil)

	jobApi.AssertNotCalled(t, "CancelRun")

	err := runner.Cancel(context.Background())
	require.NoError(t, err)
}

func runJobRunnerRestartTest(t *testing.T, jobSettings *jobs.JobSettings) {
	job := &resources.Job{
		ID:          "123",
		JobSettings: jobSettings,
	}
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"test_job": job,
				},
			},
		},
	}

	runner := jobRunner{key: "test", bundle: b, job: job}

	m := mocks.NewMockWorkspaceClient(t)
	b.SetWorkpaceClient(m.WorkspaceClient)
	ctx := context.Background()
	ctx = cmdio.InContext(ctx, cmdio.NewIO(flags.OutputText, &bytes.Buffer{}, &bytes.Buffer{}, &bytes.Buffer{}, "", ""))
	ctx = cmdio.NewContext(ctx, cmdio.NewLogger(flags.ModeAppend))

	jobApi := m.GetMockJobsAPI()
	jobApi.EXPECT().ListRunsAll(mock.Anything, jobs.ListRunsRequest{
		ActiveOnly: true,
		JobId:      123,
	}).Return([]jobs.BaseRun{
		{RunId: 1},
		{RunId: 2},
	}, nil)

	// Mock the runner cancelling existing job runs.
	mockWait := &jobs.WaitGetRunJobTerminatedOrSkipped[struct{}]{
		Poll: func(time time.Duration, f func(j *jobs.Run)) (*jobs.Run, error) {
			return nil, nil
		},
	}
	jobApi.EXPECT().CancelRun(mock.Anything, jobs.CancelRun{
		RunId: 1,
	}).Return(mockWait, nil)
	jobApi.EXPECT().CancelRun(mock.Anything, jobs.CancelRun{
		RunId: 2,
	}).Return(mockWait, nil)

	// Mock the runner triggering a job run
	mockWaitForRun := &jobs.WaitGetRunJobTerminatedOrSkipped[jobs.RunNowResponse]{
		Poll: func(d time.Duration, f func(*jobs.Run)) (*jobs.Run, error) {
			return &jobs.Run{
				State: &jobs.RunState{
					ResultState: jobs.RunResultStateSuccess,
				},
			}, nil
		},
	}
	jobApi.EXPECT().RunNow(mock.Anything, jobs.RunNow{
		JobId: 123,
	}).Return(mockWaitForRun, nil)

	// Mock the runner getting the job output
	jobApi.EXPECT().GetRun(mock.Anything, jobs.GetRunRequest{}).Return(&jobs.Run{}, nil)

	_, err := runner.Restart(ctx, &Options{})
	require.NoError(t, err)
}

func TestJobRunnerRestart(t *testing.T) {
	for _, jobSettings := range []*jobs.JobSettings{
		{},
		{
			Continuous: &jobs.Continuous{
				PauseStatus: jobs.PauseStatusPaused,
			},
		},
	} {
		runJobRunnerRestartTest(t, jobSettings)
	}
}

func TestJobRunnerRestartForContinuousUnpausedJobs(t *testing.T) {
	job := &resources.Job{
		ID: "123",
		JobSettings: &jobs.JobSettings{
			Continuous: &jobs.Continuous{
				PauseStatus: jobs.PauseStatusUnpaused,
			},
		},
	}
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"test_job": job,
				},
			},
		},
	}

	runner := jobRunner{key: "test", bundle: b, job: job}

	m := mocks.NewMockWorkspaceClient(t)
	b.SetWorkpaceClient(m.WorkspaceClient)
	ctx := context.Background()
	ctx = cmdio.InContext(ctx, cmdio.NewIO(flags.OutputText, &bytes.Buffer{}, &bytes.Buffer{}, &bytes.Buffer{}, "", "..."))
	ctx = cmdio.NewContext(ctx, cmdio.NewLogger(flags.ModeAppend))

	jobApi := m.GetMockJobsAPI()

	// The runner should not try and cancel existing job runs for unpaused continuous jobs.
	jobApi.AssertNotCalled(t, "ListRunsAll")
	jobApi.AssertNotCalled(t, "CancelRun")

	// Mock the runner triggering a job run
	mockWaitForRun := &jobs.WaitGetRunJobTerminatedOrSkipped[jobs.RunNowResponse]{
		Poll: func(d time.Duration, f func(*jobs.Run)) (*jobs.Run, error) {
			return &jobs.Run{
				State: &jobs.RunState{
					ResultState: jobs.RunResultStateSuccess,
				},
			}, nil
		},
	}
	jobApi.EXPECT().RunNow(mock.Anything, jobs.RunNow{
		JobId: 123,
	}).Return(mockWaitForRun, nil)

	// Mock the runner getting the job output
	jobApi.EXPECT().GetRun(mock.Anything, jobs.GetRunRequest{}).Return(&jobs.Run{}, nil)

	_, err := runner.Restart(ctx, &Options{})
	require.NoError(t, err)
}
