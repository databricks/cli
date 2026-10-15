package paths

import (
	"testing"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/sdktypes/jobs/v2"
	"github.com/stretchr/testify/assert"
)

func TestVisitJobLibrariesPaths(t *testing.T) {
	task0 := jobs.TaskSettings{
		NotebookTask: &jobs.NotebookTask{
			NotebookPath: new("abc"),
		},
	}
	task1 := jobs.TaskSettings{
		SparkPythonTask: &jobs.SparkPythonTask{
			PythonFile: new("abc"),
		},
	}
	task2 := jobs.TaskSettings{
		DbtTask: &jobs.DbtTask{
			ProjectDirectory: new("abc"),
		},
	}
	task3 := jobs.TaskSettings{
		SqlTask: &jobs.SqlTask{
			File: &jobs.SqlTaskFile{
				Path: new("abc"),
			},
		},
	}
	task4 := jobs.TaskSettings{
		Libraries: []jobs.Library{
			{Whl: new("dist/foo.whl")},
		},
	}
	task5 := jobs.TaskSettings{
		Libraries: []jobs.Library{
			{Jar: new("dist/foo.jar")},
		},
	}
	task6 := jobs.TaskSettings{
		Libraries: []jobs.Library{
			{Requirements: new("requirements.txt")},
		},
	}

	job0 := &resources.Job{
		Tasks: []jobs.TaskSettings{
			task0,
			task1,
			task2,
			task3,
			task4,
			task5,
			task6,
		},
	}

	root := config.Root{
		Resources: config.Resources{
			Jobs: map[string]*resources.Job{
				"job0": job0,
			},
		},
	}

	actual := collectVisitedPaths(t, root, VisitJobLibrariesPaths)
	expected := []dyn.Path{
		dyn.MustPathFromString("resources.jobs.job0.tasks[4].libraries[0].whl"),
		dyn.MustPathFromString("resources.jobs.job0.tasks[5].libraries[0].jar"),
	}

	assert.ElementsMatch(t, expected, actual)
}

func TestVisitJobLibrariesPaths_environments(t *testing.T) {
	environment0 := jobs.JobEnvironment{
		Spec: &jobs.Environment{
			Dependencies: []string{
				"dist_0/*.whl",
				"dist_1/*.whl",
			},
		},
	}
	job0 := &resources.Job{
		Environments: []jobs.JobEnvironment{
			environment0,
		},
	}

	root := config.Root{
		Resources: config.Resources{
			Jobs: map[string]*resources.Job{
				"job0": job0,
			},
		},
	}

	actual := collectVisitedPaths(t, root, VisitJobLibrariesPaths)
	expected := []dyn.Path{
		dyn.MustPathFromString("resources.jobs.job0.environments[0].spec.dependencies[0]"),
		dyn.MustPathFromString("resources.jobs.job0.environments[0].spec.dependencies[1]"),
	}

	assert.ElementsMatch(t, expected, actual)
}
