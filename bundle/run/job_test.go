package run

import (
	"context"
	"testing"
	"time"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
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
