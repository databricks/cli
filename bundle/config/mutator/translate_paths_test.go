package mutator_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/internal/bundletest"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func touchNotebookFile(t *testing.T, path string) {
	f, err := os.Create(path)
	require.NoError(t, err)
	f.WriteString("# Databricks notebook source\n")
	f.Close()
}

func touchEmptyFile(t *testing.T, path string) {
	err := os.MkdirAll(filepath.Dir(path), 0700)
	require.NoError(t, err)
	f, err := os.Create(path)
	require.NoError(t, err)
	f.Close()
}

func TestTranslatePathsSkippedWithGitSource(t *testing.T) {
	dir := t.TempDir()
	b := &bundle.Bundle{
		Config: config.Root{
			Path: dir,
			Workspace: config.Workspace{
				FilePath: "/bundle",
			},
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"job": {
						JobSettings: &jobs.JobSettings{
							GitSource: &jobs.GitSource{
								GitBranch:   "somebranch",
								GitCommit:   "somecommit",
								GitProvider: "github",
								GitTag:      "sometag",
								GitUrl:      "https://github.com/someuser/somerepo",
							},
							Tasks: []jobs.Task{
								{
									NotebookTask: &jobs.NotebookTask{
										NotebookPath: "my_job_notebook.py",
									},
								},
								{
									PythonWheelTask: &jobs.PythonWheelTask{
										PackageName: "foo",
									},
								},
								{
									SparkPythonTask: &jobs.SparkPythonTask{
										PythonFile: "my_python_file.py",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	bundletest.SetLocation(b, dyn.EmptyPath, filepath.Join(dir, "resource.yml"))

	err := bundle.Apply(context.Background(), b, mutator.TranslatePaths())
	require.NoError(t, err)

	assert.Equal(
		t,
		"my_job_notebook.py",
		b.Config.Resources.Jobs["job"].Tasks[0].NotebookTask.NotebookPath,
	)
	assert.Equal(
		t,
		"foo",
		b.Config.Resources.Jobs["job"].Tasks[1].PythonWheelTask.PackageName,
	)
	assert.Equal(
		t,
		"my_python_file.py",
		b.Config.Resources.Jobs["job"].Tasks[2].SparkPythonTask.PythonFile,
	)
}

func TestTranslatePaths(t *testing.T) {
	dir := t.TempDir()
	touchNotebookFile(t, filepath.Join(dir, "my_job_notebook.py"))
	touchNotebookFile(t, filepath.Join(dir, "my_pipeline_notebook.py"))
	touchEmptyFile(t, filepath.Join(dir, "my_python_file.py"))
	touchEmptyFile(t, filepath.Join(dir, "dist", "task.jar"))

	b := &bundle.Bundle{
		Config: config.Root{
			Path: dir,
			Workspace: config.Workspace{
				FilePath: "/bundle",
			},
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"job": {
						JobSettings: &jobs.JobSettings{
							Tasks: []jobs.Task{
								{
									NotebookTask: &jobs.NotebookTask{
										NotebookPath: "./my_job_notebook.py",
									},
									Libraries: []compute.Library{
										{Whl: "./dist/task.whl"},
									},
								},
								{
									NotebookTask: &jobs.NotebookTask{
										NotebookPath: "/Users/jane.doe@databricks.com/doesnt_exist.py",
									},
								},
								{
									NotebookTask: &jobs.NotebookTask{
										NotebookPath: "./my_job_notebook.py",
									},
								},
								{
									PythonWheelTask: &jobs.PythonWheelTask{
										PackageName: "foo",
									},
								},
								{
									SparkPythonTask: &jobs.SparkPythonTask{
										PythonFile: "./my_python_file.py",
									},
								},
								{
									SparkJarTask: &jobs.SparkJarTask{
										MainClassName: "HelloWorld",
									},
									Libraries: []compute.Library{
										{Jar: "./dist/task.jar"},
									},
								},
								{
									SparkJarTask: &jobs.SparkJarTask{
										MainClassName: "HelloWorldRemote",
									},
									Libraries: []compute.Library{
										{Jar: "dbfs:/bundle/dist/task_remote.jar"},
									},
								},
							},
						},
					},
				},
				Pipelines: map[string]*resources.Pipeline{
					"pipeline": {
						PipelineSpec: &pipelines.PipelineSpec{
							Libraries: []pipelines.PipelineLibrary{
								{
									Notebook: &pipelines.NotebookLibrary{
										Path: "./my_pipeline_notebook.py",
									},
								},
								{
									Notebook: &pipelines.NotebookLibrary{
										Path: "/Users/jane.doe@databricks.com/doesnt_exist.py",
									},
								},
								{
									Notebook: &pipelines.NotebookLibrary{
										Path: "./my_pipeline_notebook.py",
									},
								},
								{
									Jar: "foo",
								},
								{
									File: &pipelines.FileLibrary{
										Path: "./my_python_file.py",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	bundletest.SetLocation(b, dyn.EmptyPath, filepath.Join(dir, "resource.yml"))

	err := bundle.Apply(context.Background(), b, mutator.TranslatePaths())
	require.NoError(t, err)

	// Assert that the path in the tasks now refer to the artifact.
	assert.Equal(
		t,
		"/bundle/my_job_notebook",
		b.Config.Resources.Jobs["job"].Tasks[0].NotebookTask.NotebookPath,
	)
	assert.Equal(
		t,
		filepath.Join("dist", "task.whl"),
		b.Config.Resources.Jobs["job"].Tasks[0].Libraries[0].Whl,
	)
	assert.Equal(
		t,
		"/Users/jane.doe@databricks.com/doesnt_exist.py",
		b.Config.Resources.Jobs["job"].Tasks[1].NotebookTask.NotebookPath,
	)
	assert.Equal(
		t,
		"/bundle/my_job_notebook",
		b.Config.Resources.Jobs["job"].Tasks[2].NotebookTask.NotebookPath,
	)
	assert.Equal(
		t,
		"/bundle/my_python_file.py",
		b.Config.Resources.Jobs["job"].Tasks[4].SparkPythonTask.PythonFile,
	)
	assert.Equal(
		t,
		filepath.Join("dist", "task.jar"),
		b.Config.Resources.Jobs["job"].Tasks[5].Libraries[0].Jar,
	)
	assert.Equal(
		t,
		"dbfs:/bundle/dist/task_remote.jar",
		b.Config.Resources.Jobs["job"].Tasks[6].Libraries[0].Jar,
	)

	// Assert that the path in the libraries now refer to the artifact.
	assert.Equal(
		t,
		"/bundle/my_pipeline_notebook",
		b.Config.Resources.Pipelines["pipeline"].Libraries[0].Notebook.Path,
	)
	assert.Equal(
		t,
		"/Users/jane.doe@databricks.com/doesnt_exist.py",
		b.Config.Resources.Pipelines["pipeline"].Libraries[1].Notebook.Path,
	)
	assert.Equal(
		t,
		"/bundle/my_pipeline_notebook",
		b.Config.Resources.Pipelines["pipeline"].Libraries[2].Notebook.Path,
	)
	assert.Equal(
		t,
		"/bundle/my_python_file.py",
		b.Config.Resources.Pipelines["pipeline"].Libraries[4].File.Path,
	)
}

func TestTranslatePathsInSubdirectories(t *testing.T) {
	dir := t.TempDir()
	touchEmptyFile(t, filepath.Join(dir, "job", "my_python_file.py"))
	touchEmptyFile(t, filepath.Join(dir, "job", "dist", "task.jar"))
	touchEmptyFile(t, filepath.Join(dir, "pipeline", "my_python_file.py"))
	touchEmptyFile(t, filepath.Join(dir, "job", "my_sql_file.sql"))
	touchEmptyFile(t, filepath.Join(dir, "job", "my_dbt_project", "dbt_project.yml"))

	b := &bundle.Bundle{
		Config: config.Root{
			Path: dir,
			Workspace: config.Workspace{
				FilePath: "/bundle",
			},
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"job": {
						JobSettings: &jobs.JobSettings{
							Tasks: []jobs.Task{
								{
									SparkPythonTask: &jobs.SparkPythonTask{
										PythonFile: "./my_python_file.py",
									},
								},
								{
									SparkJarTask: &jobs.SparkJarTask{
										MainClassName: "HelloWorld",
									},
									Libraries: []compute.Library{
										{Jar: "./dist/task.jar"},
									},
								},
								{
									SqlTask: &jobs.SqlTask{
										File: &jobs.SqlTaskFile{
											Path: "./my_sql_file.sql",
										},
									},
								},
								{
									DbtTask: &jobs.DbtTask{
										ProjectDirectory: "./my_dbt_project",
									},
								},
							},
						},
					},
				},
				Pipelines: map[string]*resources.Pipeline{
					"pipeline": {
						PipelineSpec: &pipelines.PipelineSpec{
							Libraries: []pipelines.PipelineLibrary{
								{
									File: &pipelines.FileLibrary{
										Path: "./my_python_file.py",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	bundletest.SetLocation(b, dyn.NewPath(dyn.Key("resources"), dyn.Key("jobs")), filepath.Join(dir, "job/resource.yml"))
	bundletest.SetLocation(b, dyn.NewPath(dyn.Key("resources"), dyn.Key("pipelines")), filepath.Join(dir, "pipeline/resource.yml"))

	err := bundle.Apply(context.Background(), b, mutator.TranslatePaths())
	require.NoError(t, err)

	assert.Equal(
		t,
		"/bundle/job/my_python_file.py",
		b.Config.Resources.Jobs["job"].Tasks[0].SparkPythonTask.PythonFile,
	)
	assert.Equal(
		t,
		filepath.Join("job", "dist", "task.jar"),
		b.Config.Resources.Jobs["job"].Tasks[1].Libraries[0].Jar,
	)
	assert.Equal(
		t,
		"/bundle/job/my_sql_file.sql",
		b.Config.Resources.Jobs["job"].Tasks[2].SqlTask.File.Path,
	)
	assert.Equal(
		t,
		"/bundle/job/my_dbt_project",
		b.Config.Resources.Jobs["job"].Tasks[3].DbtTask.ProjectDirectory,
	)

	assert.Equal(
		t,
		"/bundle/pipeline/my_python_file.py",
		b.Config.Resources.Pipelines["pipeline"].Libraries[0].File.Path,
	)
}

func TestTranslatePathsOutsideBundleRoot(t *testing.T) {
	dir := t.TempDir()

	b := &bundle.Bundle{
		Config: config.Root{
			Path: dir,
			Workspace: config.Workspace{
				FilePath: "/bundle",
			},
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"job": {
						JobSettings: &jobs.JobSettings{
							Tasks: []jobs.Task{
								{
									SparkPythonTask: &jobs.SparkPythonTask{
										PythonFile: "./my_python_file.py",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	bundletest.SetLocation(b, dyn.EmptyPath, filepath.Join(dir, "../resource.yml"))

	err := bundle.Apply(context.Background(), b, mutator.TranslatePaths())
	assert.ErrorContains(t, err, "is not contained in bundle root")
}

func TestJobNotebookDoesNotExistError(t *testing.T) {
	dir := t.TempDir()

	b := &bundle.Bundle{
		Config: config.Root{
			Path: dir,
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"job": {
						JobSettings: &jobs.JobSettings{
							Tasks: []jobs.Task{
								{
									NotebookTask: &jobs.NotebookTask{
										NotebookPath: "./doesnt_exist.py",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	bundletest.SetLocation(b, dyn.EmptyPath, filepath.Join(dir, "fake.yml"))

	err := bundle.Apply(context.Background(), b, mutator.TranslatePaths())
	assert.EqualError(t, err, "notebook ./doesnt_exist.py not found")
}

func TestJobFileDoesNotExistError(t *testing.T) {
	dir := t.TempDir()

	b := &bundle.Bundle{
		Config: config.Root{
			Path: dir,
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"job": {
						JobSettings: &jobs.JobSettings{
							Tasks: []jobs.Task{
								{
									SparkPythonTask: &jobs.SparkPythonTask{
										PythonFile: "./doesnt_exist.py",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	bundletest.SetLocation(b, dyn.EmptyPath, filepath.Join(dir, "fake.yml"))

	err := bundle.Apply(context.Background(), b, mutator.TranslatePaths())
	assert.EqualError(t, err, "file ./doesnt_exist.py not found")
}

func TestPipelineNotebookDoesNotExistError(t *testing.T) {
	dir := t.TempDir()

	b := &bundle.Bundle{
		Config: config.Root{
			Path: dir,
			Resources: config.Resources{
				Pipelines: map[string]*resources.Pipeline{
					"pipeline": {
						PipelineSpec: &pipelines.PipelineSpec{
							Libraries: []pipelines.PipelineLibrary{
								{
									Notebook: &pipelines.NotebookLibrary{
										Path: "./doesnt_exist.py",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	bundletest.SetLocation(b, dyn.EmptyPath, filepath.Join(dir, "fake.yml"))

	err := bundle.Apply(context.Background(), b, mutator.TranslatePaths())
	assert.EqualError(t, err, "notebook ./doesnt_exist.py not found")
}

func TestPipelineFileDoesNotExistError(t *testing.T) {
	dir := t.TempDir()

	b := &bundle.Bundle{
		Config: config.Root{
			Path: dir,
			Resources: config.Resources{
				Pipelines: map[string]*resources.Pipeline{
					"pipeline": {
						PipelineSpec: &pipelines.PipelineSpec{
							Libraries: []pipelines.PipelineLibrary{
								{
									File: &pipelines.FileLibrary{
										Path: "./doesnt_exist.py",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	bundletest.SetLocation(b, dyn.EmptyPath, filepath.Join(dir, "fake.yml"))

	err := bundle.Apply(context.Background(), b, mutator.TranslatePaths())
	assert.EqualError(t, err, "file ./doesnt_exist.py not found")
}

func TestJobSparkPythonTaskWithNotebookSourceError(t *testing.T) {
	dir := t.TempDir()
	touchNotebookFile(t, filepath.Join(dir, "my_notebook.py"))

	b := &bundle.Bundle{
		Config: config.Root{
			Path: dir,
			Workspace: config.Workspace{
				FilePath: "/bundle",
			},
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"job": {
						JobSettings: &jobs.JobSettings{
							Tasks: []jobs.Task{
								{
									SparkPythonTask: &jobs.SparkPythonTask{
										PythonFile: "./my_notebook.py",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	bundletest.SetLocation(b, dyn.EmptyPath, filepath.Join(dir, "resource.yml"))

	err := bundle.Apply(context.Background(), b, mutator.TranslatePaths())
	assert.ErrorContains(t, err, `expected a file for "tasks.spark_python_task.python_file" but got a notebook`)
}

func TestJobNotebookTaskWithFileSourceError(t *testing.T) {
	dir := t.TempDir()
	touchEmptyFile(t, filepath.Join(dir, "my_file.py"))

	b := &bundle.Bundle{
		Config: config.Root{
			Path: dir,
			Workspace: config.Workspace{
				FilePath: "/bundle",
			},
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"job": {
						JobSettings: &jobs.JobSettings{
							Tasks: []jobs.Task{
								{
									NotebookTask: &jobs.NotebookTask{
										NotebookPath: "./my_file.py",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	bundletest.SetLocation(b, dyn.EmptyPath, filepath.Join(dir, "resource.yml"))

	err := bundle.Apply(context.Background(), b, mutator.TranslatePaths())
	assert.ErrorContains(t, err, `expected a notebook for "tasks.notebook_task.notebook_path" but got a file`)
}

func TestPipelineNotebookLibraryWithFileSourceError(t *testing.T) {
	dir := t.TempDir()
	touchEmptyFile(t, filepath.Join(dir, "my_file.py"))

	b := &bundle.Bundle{
		Config: config.Root{
			Path: dir,
			Workspace: config.Workspace{
				FilePath: "/bundle",
			},
			Resources: config.Resources{
				Pipelines: map[string]*resources.Pipeline{
					"pipeline": {
						PipelineSpec: &pipelines.PipelineSpec{
							Libraries: []pipelines.PipelineLibrary{
								{
									Notebook: &pipelines.NotebookLibrary{
										Path: "./my_file.py",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	bundletest.SetLocation(b, dyn.EmptyPath, filepath.Join(dir, "resource.yml"))

	err := bundle.Apply(context.Background(), b, mutator.TranslatePaths())
	assert.ErrorContains(t, err, `expected a notebook for "libraries.notebook.path" but got a file`)
}

func TestPipelineFileLibraryWithNotebookSourceError(t *testing.T) {
	dir := t.TempDir()
	touchNotebookFile(t, filepath.Join(dir, "my_notebook.py"))

	b := &bundle.Bundle{
		Config: config.Root{
			Path: dir,
			Workspace: config.Workspace{
				FilePath: "/bundle",
			},
			Resources: config.Resources{
				Pipelines: map[string]*resources.Pipeline{
					"pipeline": {
						PipelineSpec: &pipelines.PipelineSpec{
							Libraries: []pipelines.PipelineLibrary{
								{
									File: &pipelines.FileLibrary{
										Path: "./my_notebook.py",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	bundletest.SetLocation(b, dyn.EmptyPath, filepath.Join(dir, "resource.yml"))

	err := bundle.Apply(context.Background(), b, mutator.TranslatePaths())
	assert.ErrorContains(t, err, `expected a file for "libraries.file.path" but got a notebook`)
}
