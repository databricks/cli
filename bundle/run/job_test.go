package run

import (
	"errors"
	"github.com/databricks/cli/libs/sdktypes/jobs/v2"
	"testing"
	"time"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestConvertPythonParams(t *testing.T) {
	job := &resources.Job{
		Tasks: []jobs.TaskSettings{
			{PythonWheelTask: &jobs.PythonWheelTask{
				PackageName: new("my_test_code"),
				EntryPoint:  new("run"),
			}},
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
	err := runner.convertPythonParams(opts)
	require.NoError(t, err)
	require.NotContains(t, opts.Job.notebookParams, "__python_params")

	opts = &Options{
		Job: JobOptions{
			pythonParams: []string{"param1", "param2", "param3"},
		},
	}
	err = runner.convertPythonParams(opts)
	require.NoError(t, err)
	require.Contains(t, opts.Job.notebookParams, "__python_params")
	require.Equal(t, `["param1","param2","param3"]`, opts.Job.notebookParams["__python_params"])
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
		ActiveOnly: new(true),
		JobId:      new(int64(123)),
	}).Return([]jobs.BaseRun{
		{RunId: new(int64(1))},
		{RunId: new(int64(2))},
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

	err := runner.Cancel(t.Context())
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
		ActiveOnly: new(true),
		JobId:      new(int64(123)),
	}).Return([]jobs.BaseRun{}, nil)

	jobApi.AssertNotCalled(t, "CancelRun")

	err := runner.Cancel(t.Context())
	require.NoError(t, err)
}

func TestJobRunnerRestart(t *testing.T) {
	for _, jobSettings := range []jobs.JobSettings{
		{},
		{
			Continuous: &jobs.ContinuousSettings{
				PauseStatus: jobs.SchedulePauseStatus_Paused,
			},
		},
	} {
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

		ctx := cmdio.MockDiscard(t.Context())

		jobApi := m.GetMockJobsAPI()
		jobApi.EXPECT().ListRunsAll(mock.Anything, jobs.ListRunsRequest{
			ActiveOnly: new(true),
			JobId:      new(int64(123)),
		}).Return([]jobs.BaseRun{
			{RunId: new(int64(1))},
			{RunId: new(int64(2))},
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
						ResultState: jobs.RunResultState_RunResultState_Success,
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
}

func TestJobRunnerRunNoWaitGetRunFails(t *testing.T) {
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

	ctx := cmdio.MockDiscard(t.Context())

	jobApi := m.GetMockJobsAPI()
	jobApi.EXPECT().RunNow(mock.Anything, jobs.RunNow{
		JobId: 123,
	}).Return(&jobs.WaitGetRunJobTerminatedOrSkipped[jobs.RunNowResponse]{RunId: 456}, nil)

	// Run must surface the error instead of dereferencing the nil run details.
	jobApi.EXPECT().GetRun(mock.Anything, jobs.GetRunRequest{
		RunId: new(int64(456)),
	}).Return(nil, errors.New("transient error"))

	_, err := runner.Run(ctx, &Options{NoWait: true})
	require.ErrorContains(t, err, "transient error")
}

func TestJobRunnerRestartForContinuousUnpausedJobs(t *testing.T) {
	job := &resources.Job{
		ID: "123",
		Continuous: &jobs.ContinuousSettings{
			PauseStatus: jobs.PauseStatusUnpaused,
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

	ctx := cmdio.MockDiscard(t.Context())

	jobApi := m.GetMockJobsAPI()

	// The runner should not try and cancel existing job runs for unpaused continuous jobs.
	jobApi.AssertNotCalled(t, "ListRunsAll")
	jobApi.AssertNotCalled(t, "CancelRun")

	// Mock the runner triggering a job run
	mockWaitForRun := &jobs.WaitGetRunJobTerminatedOrSkipped[jobs.RunNowResponse]{
		Poll: func(d time.Duration, f func(*jobs.Run)) (*jobs.Run, error) {
			return &jobs.Run{
				State: &jobs.RunState{
					ResultState: jobs.RunResultState_RunResultState_Success,
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
