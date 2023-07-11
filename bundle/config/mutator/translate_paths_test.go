package mutator_test

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/databricks/cli/bundle/config/resources"
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
	bundle := &bundle.Bundle{
		Config: config.Root{
			Path: dir,
			Workspace: config.Workspace{
				FilesPath: "/bundle",
			},
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"job": {

						Paths: resources.Paths{
							ConfigFilePath: filepath.Join(dir, "resource.yml"),
						},
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

	err := mutator.TranslatePaths().Apply(context.Background(), bundle)
	require.NoError(t, err)

	assert.Equal(
		t,
		"my_job_notebook.py",
		bundle.Config.Resources.Jobs["job"].Tasks[0].NotebookTask.NotebookPath,
	)
	assert.Equal(
		t,
		"foo",
		bundle.Config.Resources.Jobs["job"].Tasks[1].PythonWheelTask.PackageName,
	)
	assert.Equal(
		t,
		"my_python_file.py",
		bundle.Config.Resources.Jobs["job"].Tasks[2].SparkPythonTask.PythonFile,
	)
}

func TestTranslatePaths(t *testing.T) {
	dir := t.TempDir()
	touchNotebookFile(t, filepath.Join(dir, "my_job_notebook.py"))
	touchNotebookFile(t, filepath.Join(dir, "my_pipeline_notebook.py"))
	touchEmptyFile(t, filepath.Join(dir, "my_python_file.py"))

	bundle := &bundle.Bundle{
		Config: config.Root{
			Path: dir,
			Workspace: config.Workspace{
				FilesPath: "/bundle",
			},
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"job": {
						Paths: resources.Paths{
							ConfigFilePath: filepath.Join(dir, "resource.yml"),
						},
						JobSettings: &jobs.JobSettings{
							Tasks: []jobs.Task{
								{
									NotebookTask: &jobs.NotebookTask{
										NotebookPath: "./my_job_notebook.py",
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
							},
						},
					},
				},
				Pipelines: map[string]*resources.Pipeline{
					"pipeline": {
						Paths: resources.Paths{
							ConfigFilePath: filepath.Join(dir, "resource.yml"),
						},
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

	err := mutator.TranslatePaths().Apply(context.Background(), bundle)
	require.NoError(t, err)

	// Assert that the path in the tasks now refer to the artifact.
	assert.Equal(
		t,
		"/bundle/my_job_notebook",
		bundle.Config.Resources.Jobs["job"].Tasks[0].NotebookTask.NotebookPath,
	)
	assert.Equal(
		t,
		"/Users/jane.doe@databricks.com/doesnt_exist.py",
		bundle.Config.Resources.Jobs["job"].Tasks[1].NotebookTask.NotebookPath,
	)
	assert.Equal(
		t,
		"/bundle/my_job_notebook",
		bundle.Config.Resources.Jobs["job"].Tasks[2].NotebookTask.NotebookPath,
	)
	assert.Equal(
		t,
		"/bundle/my_python_file.py",
		bundle.Config.Resources.Jobs["job"].Tasks[4].SparkPythonTask.PythonFile,
	)

	// Assert that the path in the libraries now refer to the artifact.
	assert.Equal(
		t,
		"/bundle/my_pipeline_notebook",
		bundle.Config.Resources.Pipelines["pipeline"].Libraries[0].Notebook.Path,
	)
	assert.Equal(
		t,
		"/Users/jane.doe@databricks.com/doesnt_exist.py",
		bundle.Config.Resources.Pipelines["pipeline"].Libraries[1].Notebook.Path,
	)
	assert.Equal(
		t,
		"/bundle/my_pipeline_notebook",
		bundle.Config.Resources.Pipelines["pipeline"].Libraries[2].Notebook.Path,
	)
	assert.Equal(
		t,
		"/bundle/my_python_file.py",
		bundle.Config.Resources.Pipelines["pipeline"].Libraries[4].File.Path,
	)
}

func TestTranslatePathsInSubdirectories(t *testing.T) {
	dir := t.TempDir()
	touchEmptyFile(t, filepath.Join(dir, "job", "my_python_file.py"))
	touchEmptyFile(t, filepath.Join(dir, "pipeline", "my_python_file.py"))

	bundle := &bundle.Bundle{
		Config: config.Root{
			Path: dir,
			Workspace: config.Workspace{
				FilesPath: "/bundle",
			},
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"job": {
						Paths: resources.Paths{
							ConfigFilePath: filepath.Join(dir, "job/resource.yml"),
						},
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
				Pipelines: map[string]*resources.Pipeline{
					"pipeline": {
						Paths: resources.Paths{
							ConfigFilePath: filepath.Join(dir, "pipeline/resource.yml"),
						},

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

	err := mutator.TranslatePaths().Apply(context.Background(), bundle)
	require.NoError(t, err)

	assert.Equal(
		t,
		"/bundle/job/my_python_file.py",
		bundle.Config.Resources.Jobs["job"].Tasks[0].SparkPythonTask.PythonFile,
	)

	assert.Equal(
		t,
		"/bundle/pipeline/my_python_file.py",
		bundle.Config.Resources.Pipelines["pipeline"].Libraries[0].File.Path,
	)
}

func TestTranslatePathsOutsideBundleRoot(t *testing.T) {
	dir := t.TempDir()

	bundle := &bundle.Bundle{
		Config: config.Root{
			Path: dir,
			Workspace: config.Workspace{
				FilesPath: "/bundle",
			},
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"job": {
						Paths: resources.Paths{
							ConfigFilePath: filepath.Join(dir, "../resource.yml"),
						},
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

	err := mutator.TranslatePaths().Apply(context.Background(), bundle)
	assert.ErrorContains(t, err, "is not contained in bundle root")
}

func TestJobNotebookDoesNotExistError(t *testing.T) {
	dir := t.TempDir()

	bundle := &bundle.Bundle{
		Config: config.Root{
			Path: dir,
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"job": {
						Paths: resources.Paths{
							ConfigFilePath: filepath.Join(dir, "fake.yml"),
						},
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

	err := mutator.TranslatePaths().Apply(context.Background(), bundle)
	assert.EqualError(t, err, "notebook ./doesnt_exist.py not found")
}

func TestJobFileDoesNotExistError(t *testing.T) {
	dir := t.TempDir()

	bundle := &bundle.Bundle{
		Config: config.Root{
			Path: dir,
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"job": {
						Paths: resources.Paths{
							ConfigFilePath: filepath.Join(dir, "fake.yml"),
						},
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

	err := mutator.TranslatePaths().Apply(context.Background(), bundle)
	assert.EqualError(t, err, "file ./doesnt_exist.py not found")
}

func TestPipelineNotebookDoesNotExistError(t *testing.T) {
	dir := t.TempDir()

	bundle := &bundle.Bundle{
		Config: config.Root{
			Path: dir,
			Resources: config.Resources{
				Pipelines: map[string]*resources.Pipeline{
					"pipeline": {
						Paths: resources.Paths{
							ConfigFilePath: filepath.Join(dir, "fake.yml"),
						},
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

	err := mutator.TranslatePaths().Apply(context.Background(), bundle)
	assert.EqualError(t, err, "notebook ./doesnt_exist.py not found")
}

func TestPipelineFileDoesNotExistError(t *testing.T) {
	dir := t.TempDir()

	bundle := &bundle.Bundle{
		Config: config.Root{
			Path: dir,
			Resources: config.Resources{
				Pipelines: map[string]*resources.Pipeline{
					"pipeline": {
						Paths: resources.Paths{
							ConfigFilePath: filepath.Join(dir, "fake.yml"),
						},
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

	err := mutator.TranslatePaths().Apply(context.Background(), bundle)
	assert.EqualError(t, err, "file ./doesnt_exist.py not found")
}

func TestSparkPythonTaskJobWithNotebookSourceError(t *testing.T) {
	dir := t.TempDir()
	touchNotebookFile(t, filepath.Join(dir, "my_notebook.py"))

	bundle := &bundle.Bundle{
		Config: config.Root{
			Path: dir,
			Workspace: config.Workspace{
				FilesPath: "/bundle",
			},
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"job": {
						Paths: resources.Paths{
							ConfigFilePath: filepath.Join(dir, "resource.yml"),
						},
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

	err := mutator.TranslatePaths().Apply(context.Background(), bundle)
	assert.ErrorContains(t, err, "please use notebook task type for notebooks")
}

func TestNotebookTaskJobWithFileSourceError(t *testing.T) {
	dir := t.TempDir()
	touchEmptyFile(t, filepath.Join(dir, "my_file.py"))

	bundle := &bundle.Bundle{
		Config: config.Root{
			Path: dir,
			Workspace: config.Workspace{
				FilesPath: "/bundle",
			},
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"job": {
						Paths: resources.Paths{
							ConfigFilePath: filepath.Join(dir, "resource.yml"),
						},
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

	err := mutator.TranslatePaths().Apply(context.Background(), bundle)
	assert.Regexp(t, regexp.MustCompile("file at .* is not a notebook"), err.Error())
}

func TestNotebookLibraryPipelineWithFileSourceError(t *testing.T) {
	dir := t.TempDir()
	touchEmptyFile(t, filepath.Join(dir, "my_file.py"))

	bundle := &bundle.Bundle{
		Config: config.Root{
			Path: dir,
			Workspace: config.Workspace{
				FilesPath: "/bundle",
			},
			Resources: config.Resources{
				Pipelines: map[string]*resources.Pipeline{
					"pipeline": {
						Paths: resources.Paths{
							ConfigFilePath: filepath.Join(dir, "resource.yml"),
						},
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

	err := mutator.TranslatePaths().Apply(context.Background(), bundle)
	assert.Regexp(t, regexp.MustCompile("file at .* is not a notebook"), err.Error())
}

func TestFileLibraryPipelineWithNotebookSourceError(t *testing.T) {
	dir := t.TempDir()
	touchNotebookFile(t, filepath.Join(dir, "my_notebook.py"))

	bundle := &bundle.Bundle{
		Config: config.Root{
			Path: dir,
			Workspace: config.Workspace{
				FilesPath: "/bundle",
			},
			Resources: config.Resources{
				Pipelines: map[string]*resources.Pipeline{
					"pipeline": {
						Paths: resources.Paths{
							ConfigFilePath: filepath.Join(dir, "resource.yml"),
						},
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

	err := mutator.TranslatePaths().Apply(context.Background(), bundle)
	assert.ErrorContains(t, err, "please specify notebooks as notebook libraries (use libraries.notebook.path)")
}
