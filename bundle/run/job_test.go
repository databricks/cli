package run

import (
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/service/jobs"
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
