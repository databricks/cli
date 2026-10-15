package run

import (
	"encoding/json"
	"slices"
	"sync"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/internal/bundletest"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/sdktypes/jobs/v2"
	"github.com/databricks/cli/libs/testserver"
	"github.com/stretchr/testify/assert"
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

// fakeJobs serves the jobs API calls made by jobRunner and records them.
type fakeJobs struct {
	mu sync.Mutex

	// activeRuns is returned by runs/list.
	activeRuns []int64
	// getRunError, if set, is returned by runs/get.
	getRunError *testserver.Response

	listCalls  int
	cancelled  []int64
	runNowJobs []int64
}

func newJobRunner(t *testing.T, settings jobs.JobSettings) (jobRunner, *fakeJobs) {
	job := &resources.Job{
		ID:          "123",
		JobSettings: settings,
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

	w, server := bundletest.NewClients(t)
	b.SetWorkpaceClient(w)

	f := &fakeJobs{}
	server.Handle("GET", "/api/2.2/jobs/runs/list", func(req testserver.Request) any {
		f.mu.Lock()
		defer f.mu.Unlock()
		assert.Equal(t, "123", req.URL.Query().Get("job_id"))
		assert.Equal(t, "true", req.URL.Query().Get("active_only"))
		f.listCalls++
		var runs []map[string]any
		for _, id := range f.activeRuns {
			runs = append(runs, map[string]any{"run_id": id})
		}
		return map[string]any{"runs": runs}
	})
	server.Handle("POST", "/api/2.2/jobs/runs/cancel", func(req testserver.Request) any {
		f.mu.Lock()
		defer f.mu.Unlock()
		var body struct {
			RunId int64 `json:"run_id"`
		}
		assert.NoError(t, json.Unmarshal(req.Body, &body))
		f.cancelled = append(f.cancelled, body.RunId)
		return map[string]any{}
	})
	server.Handle("POST", "/api/2.2/jobs/run-now", func(req testserver.Request) any {
		f.mu.Lock()
		defer f.mu.Unlock()
		var body struct {
			JobId int64 `json:"job_id"`
		}
		assert.NoError(t, json.Unmarshal(req.Body, &body))
		f.runNowJobs = append(f.runNowJobs, body.JobId)
		return map[string]any{"run_id": 456}
	})
	server.Handle("GET", "/api/2.2/jobs/runs/get", func(req testserver.Request) any {
		f.mu.Lock()
		defer f.mu.Unlock()
		if f.getRunError != nil {
			return *f.getRunError
		}
		return map[string]any{
			"run_id": req.URL.Query().Get("run_id"),
			"state":  map[string]any{"life_cycle_state": "TERMINATED", "result_state": "SUCCESS"},
		}
	})

	return jobRunner{key: "test", bundle: b, job: job}, f
}

func (f *fakeJobs) cancelledRuns() []int64 {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Sorted(slices.Values(f.cancelled))
}

func TestJobRunnerCancel(t *testing.T) {
	runner, f := newJobRunner(t, jobs.JobSettings{})
	f.activeRuns = []int64{1, 2}

	err := runner.Cancel(t.Context())
	require.NoError(t, err)
	assert.Equal(t, []int64{1, 2}, f.cancelledRuns())
}

func TestJobRunnerCancelWithNoActiveRuns(t *testing.T) {
	runner, f := newJobRunner(t, jobs.JobSettings{})

	err := runner.Cancel(t.Context())
	require.NoError(t, err)
	assert.Equal(t, 1, f.listCalls)
	assert.Empty(t, f.cancelledRuns())
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
		runner, f := newJobRunner(t, jobSettings)
		f.activeRuns = []int64{1, 2}

		ctx := cmdio.MockDiscard(t.Context())
		_, err := runner.Restart(ctx, &Options{})
		require.NoError(t, err)
		assert.Equal(t, []int64{1, 2}, f.cancelledRuns())
		assert.Equal(t, []int64{123}, f.runNowJobs)
	}
}

func TestJobRunnerRunNoWaitGetRunFails(t *testing.T) {
	runner, f := newJobRunner(t, jobs.JobSettings{})
	f.getRunError = &testserver.Response{
		StatusCode: 400,
		Body:       map[string]string{"error_code": "INVALID_PARAMETER_VALUE", "message": "transient error"},
	}

	ctx := cmdio.MockDiscard(t.Context())

	// Run must surface the error instead of dereferencing the nil run details.
	_, err := runner.Run(ctx, &Options{NoWait: true})
	require.ErrorContains(t, err, "transient error")
}

func TestJobRunnerRestartForContinuousUnpausedJobs(t *testing.T) {
	runner, f := newJobRunner(t, jobs.JobSettings{
		Continuous: &jobs.ContinuousSettings{
			PauseStatus: "UNPAUSED",
		},
	})

	ctx := cmdio.MockDiscard(t.Context())
	_, err := runner.Restart(ctx, &Options{})
	require.NoError(t, err)

	// The runner should not try and cancel existing job runs for unpaused continuous jobs.
	assert.Equal(t, 0, f.listCalls)
	assert.Empty(t, f.cancelledRuns())
	assert.Equal(t, []int64{123}, f.runNowJobs)
}
