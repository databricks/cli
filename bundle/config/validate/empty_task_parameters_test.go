package validate

import (
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/engine"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/require"
)

func emptyTaskParametersBundle(eng engine.EngineType) *bundle.Bundle {
	return &bundle.Bundle{
		Config: config.Root{
			Bundle: config.Bundle{
				Engine: eng,
			},
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"job1": {
						JobSettings: jobs.JobSettings{
							Name: "job1",
							Tasks: []jobs.Task{
								{
									TaskKey: "python",
									SparkPythonTask: &jobs.SparkPythonTask{
										PythonFile: "./main.py",
										Parameters: []string{"--suffix", ""},
									},
								},
								{
									TaskKey: "wheel",
									PythonWheelTask: &jobs.PythonWheelTask{
										PackageName: "pkg",
										Parameters:  []string{""},
									},
								},
								{
									TaskKey: "ok",
									SparkPythonTask: &jobs.SparkPythonTask{
										PythonFile: "./main.py",
										Parameters: []string{"--suffix", "value"},
									},
								},
								{
									TaskKey: "foreach",
									ForEachTask: &jobs.ForEachTask{
										Task: jobs.Task{
											TaskKey: "inner",
											SparkJarTask: &jobs.SparkJarTask{
												MainClassName: "Main",
												Parameters:    []string{""},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func TestEmptyTaskParametersTerraformEngine(t *testing.T) {
	b := emptyTaskParametersBundle(engine.EngineTerraform)

	diags := EmptyTaskParameters().Apply(t.Context(), b)
	require.Len(t, diags, 3)
	require.NoError(t, diags.Error())

	var paths []string
	for _, d := range diags {
		require.Equal(t, diag.Warning, d.Severity)
		require.Len(t, d.Paths, 1)
		paths = append(paths, d.Paths[0].String())
	}
	require.ElementsMatch(t, []string{
		"resources.jobs.job1.tasks[0].spark_python_task.parameters[1]",
		"resources.jobs.job1.tasks[1].python_wheel_task.parameters[0]",
		"resources.jobs.job1.tasks[3].for_each_task.task.spark_jar_task.parameters[0]",
	}, paths)
}

func TestEmptyTaskParametersDirectEngine(t *testing.T) {
	b := emptyTaskParametersBundle(engine.EngineDirect)

	diags := EmptyTaskParameters().Apply(t.Context(), b)
	require.Empty(t, diags)
}

func TestEmptyTaskParametersNoEmptyValues(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Bundle: config.Bundle{
				Engine: engine.EngineTerraform,
			},
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"job1": {
						JobSettings: jobs.JobSettings{
							Name: "job1",
							Tasks: []jobs.Task{
								{
									TaskKey: "python",
									SparkPythonTask: &jobs.SparkPythonTask{
										PythonFile: "./main.py",
										Parameters: []string{"--suffix", "value"},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	diags := EmptyTaskParameters().Apply(t.Context(), b)
	require.Empty(t, diags)
}
