package mutator_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/config/variable"
	"github.com/databricks/cli/bundle/internal/bundletest"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/vfs"
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
		SyncRootPath: dir,
		SyncRoot:     vfs.MustNew(dir),
		Config: config.Root{
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

	bundletest.SetLocation(b, ".", []dyn.Location{{File: filepath.Join(dir, "resource.yml")}})

	diags := bundle.Apply(context.Background(), b, mutator.TranslatePaths())
	require.NoError(t, diags.Error())

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
	touchEmptyFile(t, filepath.Join(dir, "requirements.txt"))

	b := &bundle.Bundle{
		SyncRootPath: dir,
		SyncRoot:     vfs.MustNew(dir),
		Config: config.Root{
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
									Libraries: []compute.Library{
										{Requirements: "./requirements.txt"},
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

	bundletest.SetLocation(b, ".", []dyn.Location{{File: filepath.Join(dir, "resource.yml")}})

	diags := bundle.Apply(context.Background(), b, mutator.TranslatePaths())
	require.NoError(t, diags.Error())

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
		"/bundle/requirements.txt",
		b.Config.Resources.Jobs["job"].Tasks[2].Libraries[0].Requirements,
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
		SyncRootPath: dir,
		SyncRoot:     vfs.MustNew(dir),
		Config: config.Root{
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

	bundletest.SetLocation(b, "resources.jobs", []dyn.Location{{File: filepath.Join(dir, "job/resource.yml")}})
	bundletest.SetLocation(b, "resources.pipelines", []dyn.Location{{File: filepath.Join(dir, "pipeline/resource.yml")}})

	diags := bundle.Apply(context.Background(), b, mutator.TranslatePaths())
	require.NoError(t, diags.Error())

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

func TestTranslatePathsOutsideSyncRoot(t *testing.T) {
	dir := t.TempDir()

	b := &bundle.Bundle{
		SyncRootPath: dir,
		SyncRoot:     vfs.MustNew(dir),
		Config: config.Root{
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

	bundletest.SetLocation(b, ".", []dyn.Location{{File: filepath.Join(dir, "../resource.yml")}})

	diags := bundle.Apply(context.Background(), b, mutator.TranslatePaths())
	assert.ErrorContains(t, diags.Error(), "is not contained in sync root path")
}

func TestJobNotebookDoesNotExistError(t *testing.T) {
	dir := t.TempDir()

	b := &bundle.Bundle{
		SyncRootPath: dir,
		SyncRoot:     vfs.MustNew(dir),
		Config: config.Root{
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

	bundletest.SetLocation(b, ".", []dyn.Location{{File: filepath.Join(dir, "fake.yml")}})

	diags := bundle.Apply(context.Background(), b, mutator.TranslatePaths())
	assert.EqualError(t, diags.Error(), "notebook ./doesnt_exist.py not found")
}

func TestJobFileDoesNotExistError(t *testing.T) {
	dir := t.TempDir()

	b := &bundle.Bundle{
		SyncRootPath: dir,
		SyncRoot:     vfs.MustNew(dir),
		Config: config.Root{
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

	bundletest.SetLocation(b, ".", []dyn.Location{{File: filepath.Join(dir, "fake.yml")}})

	diags := bundle.Apply(context.Background(), b, mutator.TranslatePaths())
	assert.EqualError(t, diags.Error(), "file ./doesnt_exist.py not found")
}

func TestPipelineNotebookDoesNotExistError(t *testing.T) {
	dir := t.TempDir()

	b := &bundle.Bundle{
		SyncRootPath: dir,
		SyncRoot:     vfs.MustNew(dir),
		Config: config.Root{
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

	bundletest.SetLocation(b, ".", []dyn.Location{{File: filepath.Join(dir, "fake.yml")}})

	diags := bundle.Apply(context.Background(), b, mutator.TranslatePaths())
	assert.EqualError(t, diags.Error(), "notebook ./doesnt_exist.py not found")
}

func TestPipelineFileDoesNotExistError(t *testing.T) {
	dir := t.TempDir()

	b := &bundle.Bundle{
		SyncRootPath: dir,
		SyncRoot:     vfs.MustNew(dir),
		Config: config.Root{
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

	bundletest.SetLocation(b, ".", []dyn.Location{{File: filepath.Join(dir, "fake.yml")}})

	diags := bundle.Apply(context.Background(), b, mutator.TranslatePaths())
	assert.EqualError(t, diags.Error(), "file ./doesnt_exist.py not found")
}

func TestJobSparkPythonTaskWithNotebookSourceError(t *testing.T) {
	dir := t.TempDir()
	touchNotebookFile(t, filepath.Join(dir, "my_notebook.py"))

	b := &bundle.Bundle{
		SyncRootPath: dir,
		SyncRoot:     vfs.MustNew(dir),
		Config: config.Root{
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

	bundletest.SetLocation(b, ".", []dyn.Location{{File: filepath.Join(dir, "resource.yml")}})

	diags := bundle.Apply(context.Background(), b, mutator.TranslatePaths())
	assert.ErrorContains(t, diags.Error(), `expected a file for "resources.jobs.job.tasks[0].spark_python_task.python_file" but got a notebook`)
}

func TestJobNotebookTaskWithFileSourceError(t *testing.T) {
	dir := t.TempDir()
	touchEmptyFile(t, filepath.Join(dir, "my_file.py"))

	b := &bundle.Bundle{
		SyncRootPath: dir,
		SyncRoot:     vfs.MustNew(dir),
		Config: config.Root{
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

	bundletest.SetLocation(b, ".", []dyn.Location{{File: filepath.Join(dir, "resource.yml")}})

	diags := bundle.Apply(context.Background(), b, mutator.TranslatePaths())
	assert.ErrorContains(t, diags.Error(), `expected a notebook for "resources.jobs.job.tasks[0].notebook_task.notebook_path" but got a file`)
}

func TestPipelineNotebookLibraryWithFileSourceError(t *testing.T) {
	dir := t.TempDir()
	touchEmptyFile(t, filepath.Join(dir, "my_file.py"))

	b := &bundle.Bundle{
		SyncRootPath: dir,
		SyncRoot:     vfs.MustNew(dir),
		Config: config.Root{
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

	bundletest.SetLocation(b, ".", []dyn.Location{{File: filepath.Join(dir, "resource.yml")}})

	diags := bundle.Apply(context.Background(), b, mutator.TranslatePaths())
	assert.ErrorContains(t, diags.Error(), `expected a notebook for "resources.pipelines.pipeline.libraries[0].notebook.path" but got a file`)
}

func TestPipelineFileLibraryWithNotebookSourceError(t *testing.T) {
	dir := t.TempDir()
	touchNotebookFile(t, filepath.Join(dir, "my_notebook.py"))

	b := &bundle.Bundle{
		SyncRootPath: dir,
		SyncRoot:     vfs.MustNew(dir),
		Config: config.Root{
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

	bundletest.SetLocation(b, ".", []dyn.Location{{File: filepath.Join(dir, "resource.yml")}})

	diags := bundle.Apply(context.Background(), b, mutator.TranslatePaths())
	assert.ErrorContains(t, diags.Error(), `expected a file for "resources.pipelines.pipeline.libraries[0].file.path" but got a notebook`)
}

func TestTranslatePathJobEnvironments(t *testing.T) {
	dir := t.TempDir()
	touchEmptyFile(t, filepath.Join(dir, "env1.py"))
	touchEmptyFile(t, filepath.Join(dir, "env2.py"))

	b := &bundle.Bundle{
		SyncRootPath: dir,
		SyncRoot:     vfs.MustNew(dir),
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"job": {
						JobSettings: &jobs.JobSettings{
							Environments: []jobs.JobEnvironment{
								{
									Spec: &compute.Environment{
										Dependencies: []string{
											"./dist/env1.whl",
											"../dist/env2.whl",
											"simplejson",
											"/Workspace/Users/foo@bar.com/test.whl",
											"--extra-index-url https://name:token@gitlab.com/api/v4/projects/9876/packages/pypi/simple foobar",
											"foobar --extra-index-url https://name:token@gitlab.com/api/v4/projects/9876/packages/pypi/simple",
											"https://foo@bar.com/packages/pypi/simple",
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

	bundletest.SetLocation(b, "resources.jobs", []dyn.Location{{File: filepath.Join(dir, "job/resource.yml")}})

	diags := bundle.Apply(context.Background(), b, mutator.TranslatePaths())
	require.NoError(t, diags.Error())

	assert.Equal(t, strings.Join([]string{".", "job", "dist", "env1.whl"}, string(os.PathSeparator)), b.Config.Resources.Jobs["job"].JobSettings.Environments[0].Spec.Dependencies[0])
	assert.Equal(t, strings.Join([]string{".", "dist", "env2.whl"}, string(os.PathSeparator)), b.Config.Resources.Jobs["job"].JobSettings.Environments[0].Spec.Dependencies[1])
	assert.Equal(t, "simplejson", b.Config.Resources.Jobs["job"].JobSettings.Environments[0].Spec.Dependencies[2])
	assert.Equal(t, "/Workspace/Users/foo@bar.com/test.whl", b.Config.Resources.Jobs["job"].JobSettings.Environments[0].Spec.Dependencies[3])
	assert.Equal(t, "--extra-index-url https://name:token@gitlab.com/api/v4/projects/9876/packages/pypi/simple foobar", b.Config.Resources.Jobs["job"].JobSettings.Environments[0].Spec.Dependencies[4])
	assert.Equal(t, "foobar --extra-index-url https://name:token@gitlab.com/api/v4/projects/9876/packages/pypi/simple", b.Config.Resources.Jobs["job"].JobSettings.Environments[0].Spec.Dependencies[5])
	assert.Equal(t, "https://foo@bar.com/packages/pypi/simple", b.Config.Resources.Jobs["job"].JobSettings.Environments[0].Spec.Dependencies[6])
}

func TestTranslatePathWithComplexVariables(t *testing.T) {
	dir := t.TempDir()
	b := &bundle.Bundle{
		SyncRootPath: dir,
		SyncRoot:     vfs.MustNew(dir),
		Config: config.Root{
			Variables: map[string]*variable.Variable{
				"cluster_libraries": {
					Type: variable.VariableTypeComplex,
					Default: [](map[string]string){
						{
							"whl": "./local/whl.whl",
						},
					},
				},
			},
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"job": {
						JobSettings: &jobs.JobSettings{
							Tasks: []jobs.Task{
								{
									TaskKey: "test",
								},
							},
						},
					},
				},
			},
		},
	}

	bundletest.SetLocation(b, "variables", []dyn.Location{{File: filepath.Join(dir, "variables/variables.yml")}})
	bundletest.SetLocation(b, "resources.jobs", []dyn.Location{{File: filepath.Join(dir, "job/resource.yml")}})

	ctx := context.Background()
	// Assign the variables to the dynamic configuration.
	diags := bundle.ApplyFunc(ctx, b, func(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
		err := b.Config.Mutate(func(v dyn.Value) (dyn.Value, error) {
			p := dyn.MustPathFromString("resources.jobs.job.tasks[0]")
			return dyn.SetByPath(v, p.Append(dyn.Key("libraries")), dyn.V("${var.cluster_libraries}"))
		})
		return diag.FromErr(err)
	})
	require.NoError(t, diags.Error())

	diags = bundle.Apply(ctx, b,
		bundle.Seq(
			mutator.SetVariables(),
			mutator.ResolveVariableReferences("variables"),
			mutator.TranslatePaths(),
		))
	require.NoError(t, diags.Error())

	assert.Equal(
		t,
		filepath.Join("variables", "local", "whl.whl"),
		b.Config.Resources.Jobs["job"].Tasks[0].Libraries[0].Whl,
	)
}
