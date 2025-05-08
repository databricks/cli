package paths

import (
	"testing"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/dyn"
	assert "github.com/databricks/cli/libs/dyn/dynassert"
	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/databricks/databricks-sdk-go/service/jobs"
)

func TestVisitJobPaths(t *testing.T) {
	task0 := jobs.Task{
		NotebookTask: &jobs.NotebookTask{
			NotebookPath: "abc",
		},
	}
	task1 := jobs.Task{
		SparkPythonTask: &jobs.SparkPythonTask{
			PythonFile: "abc",
		},
	}
	task2 := jobs.Task{
		DbtTask: &jobs.DbtTask{
			ProjectDirectory: "abc",
		},
	}
	task3 := jobs.Task{
		SqlTask: &jobs.SqlTask{
			File: &jobs.SqlTaskFile{
				Path: "abc",
			},
		},
	}
	task4 := jobs.Task{
		Libraries: []compute.Library{
			{Whl: "dist/foo.whl"},
		},
	}
	task5 := jobs.Task{
		Libraries: []compute.Library{
			{Jar: "dist/foo.jar"},
		},
	}
	task6 := jobs.Task{
		Libraries: []compute.Library{
			{Requirements: "requirements.txt"},
		},
	}

	job0 := &resources.Job{
		JobSettings: jobs.JobSettings{
			Tasks: []jobs.Task{
				task0,
				task1,
				task2,
				task3,
				task4,
				task5,
				task6,
			},
		},
	}

	root := config.Root{
		Resources: config.Resources{
			Jobs: map[string]*resources.Job{
				"job0": job0,
			},
		},
	}

	actual := collectVisitedPaths(t, root, VisitJobPaths)
	expected := []dyn.Path{
		dyn.MustPathFromString("resources.jobs.job0.tasks[0].notebook_task.notebook_path"),
		dyn.MustPathFromString("resources.jobs.job0.tasks[1].spark_python_task.python_file"),
		dyn.MustPathFromString("resources.jobs.job0.tasks[2].dbt_task.project_directory"),
		dyn.MustPathFromString("resources.jobs.job0.tasks[3].sql_task.file.path"),
		dyn.MustPathFromString("resources.jobs.job0.tasks[4].libraries[0].whl"),
		dyn.MustPathFromString("resources.jobs.job0.tasks[5].libraries[0].jar"),
		dyn.MustPathFromString("resources.jobs.job0.tasks[6].libraries[0].requirements"),
	}

	assert.ElementsMatch(t, expected, actual)
}

func TestVisitJobPaths_environments(t *testing.T) {
	environment0 := jobs.JobEnvironment{
		Spec: &compute.Environment{
			Dependencies: []string{
				"dist_0/*.whl",
				"dist_1/*.whl",
			},
		},
	}
	job0 := &resources.Job{
		JobSettings: jobs.JobSettings{
			Environments: []jobs.JobEnvironment{
				environment0,
			},
		},
	}

	root := config.Root{
		Resources: config.Resources{
			Jobs: map[string]*resources.Job{
				"job0": job0,
			},
		},
	}

	actual := collectVisitedPaths(t, root, VisitJobPaths)
	expected := []dyn.Path{
		dyn.MustPathFromString("resources.jobs.job0.environments[0].spec.dependencies[0]"),
		dyn.MustPathFromString("resources.jobs.job0.environments[0].spec.dependencies[1]"),
	}

	assert.ElementsMatch(t, expected, actual)
}

func TestVisitJobPaths_foreach(t *testing.T) {
	task0 := jobs.Task{
		ForEachTask: &jobs.ForEachTask{
			Task: jobs.Task{
				NotebookTask: &jobs.NotebookTask{
					NotebookPath: "abc",
				},
			},
		},
	}
	job0 := &resources.Job{
		JobSettings: jobs.JobSettings{
			Tasks: []jobs.Task{
				task0,
			},
		},
	}

	root := config.Root{
		Resources: config.Resources{
			Jobs: map[string]*resources.Job{
				"job0": job0,
			},
		},
	}

	actual := collectVisitedPaths(t, root, VisitJobPaths)
	expected := []dyn.Path{
		dyn.MustPathFromString("resources.jobs.job0.tasks[0].for_each_task.task.notebook_task.notebook_path"),
	}

	assert.ElementsMatch(t, expected, actual)
}
