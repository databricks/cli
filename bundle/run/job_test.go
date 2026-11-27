package run

import (
	"errors"
	"testing"
	"time"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestConvertPythonParams(t *testing.T) {
	job := &resources.Job{
		Tasks: []jobs.Task{
			{PythonWheelTask: &jobs.PythonWheelTask{
				PackageName: "my_test_code",
				EntryPoint:  "run",
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
		ActiveOnly: true,
		JobId:      123,
	}).Return([]jobs.BaseRun{}, nil)

	jobApi.AssertNotCalled(t, "CancelRun")

	err := runner.Cancel(t.Context())
	require.NoError(t, err)
}

func TestJobRunnerRestart(t *testing.T) {
	for _, jobSettings := range []jobs.JobSettings{
		{},
		{
			Continuous: &jobs.Continuous{
				PauseStatus: jobs.PauseStatusPaused,
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
		RunId: 456,
	}).Return(nil, errors.New("transient error"))

	_, err := runner.Run(ctx, &Options{NoWait: true})
	require.ErrorContains(t, err, "transient error")
}

func TestJobRunnerRestartForContinuousUnpausedJobs(t *testing.T) {
	job := &resources.Job{
		ID: "123",
		Continuous: &jobs.Continuous{
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

func TestJobRunnerLogsFailedTaskTraceWithItsColors(t *testing.T) {
	b := &bundle.Bundle{}
	runner := jobRunner{key: "test", bundle: b, job: &resources.Job{ID: "123"}}

	m := mocks.NewMockWorkspaceClient(t)
	b.SetWorkpaceClient(m.WorkspaceClient)

	jobApi := m.GetMockJobsAPI()
	jobApi.EXPECT().GetRun(mock.Anything, jobs.GetRunRequest{RunId: 456}).Return(&jobs.Run{
		State: &jobs.RunState{ResultState: jobs.RunResultStateFailed},
		Tasks: []jobs.RunTask{{
			TaskKey: "main",
			RunId:   789,
			State:   &jobs.RunState{LifeCycleState: jobs.RunLifeCycleStateInternalError, ResultState: jobs.RunResultStateFailed},
		}},
	}, nil)
	// A notebook's trace is colored. The error can quote a path from the task.
	jobApi.EXPECT().GetRunOutput(mock.Anything, jobs.GetRunOutputRequest{RunId: 789}).Return(&jobs.RunOutput{
		Error:      "Unable to access the notebook \"/a\x1b[2J\x1b[31m\rb\"",
		ErrorTrace: "\x1b[0;31m----\x1b[0m\n\x1b[0;31mValueError\x1b[0m: boom\x1b[2J",
	}, nil)

	ctx, stderr := cmdio.NewTestContextWithStderr(t.Context())
	runner.logFailedTasks(ctx, 456)

	// Colors are off here: the trace keeps its own, nothing else gets through.
	require.Equal(t, "Task main FAILED:\n"+`Unable to access the notebook "/a\u001b[2J\u001b[31m\rb"`+"\n"+
		"\x1b[0;31m----\x1b[0m\n\x1b[0;31mValueError\x1b[0m: boom"+`\u001b[2J`+"\n\n", stderr.String())
}
