package mutator_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/config/variable"
	"github.com/databricks/cli/bundle/internal/bundletest"
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
	_, err = f.WriteString("# Databricks notebook source\n")
	require.NoError(t, err)
	f.Close()
}

func touchEmptyFile(t *testing.T, path string) {
	err := os.MkdirAll(filepath.Dir(path), 0o700)
	require.NoError(t, err)
	f, err := os.Create(path)
	require.NoError(t, err)
	f.Close()
}

func TestTranslatePathsSkippedWithGitSource(t *testing.T) {
	dir := t.TempDir()
	b := &bundle.Bundle{
		SyncRootPath:   dir,
		BundleRootPath: dir,
		SyncRoot:       vfs.MustNew(dir),
		Config: config.Root{
			Workspace: config.Workspace{
				FilePath: "/bundle",
			},
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"job": {
						JobSettings: jobs.JobSettings{
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

	diags := bundle.ApplySeq(context.Background(), b, mutator.NormalizePaths(), mutator.TranslatePaths())
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
		SyncRootPath:   dir,
		BundleRootPath: dir,
		SyncRoot:       vfs.MustNew(dir),
		Config: config.Root{
			Workspace: config.Workspace{
				FilePath: "/bundle",
			},
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"job": {
						JobSettings: jobs.JobSettings{
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
						CreatePipeline: pipelines.CreatePipeline{
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

	diags := bundle.ApplySeq(context.Background(), b, mutator.NormalizePaths(), mutator.TranslatePaths())
	require.NoError(t, diags.Error())

	// Assert that the path in the tasks now refer to the artifact.
	assert.Equal(
		t,
		"/bundle/my_job_notebook",
		b.Config.Resources.Jobs["job"].Tasks[0].NotebookTask.NotebookPath,
	)
	assert.Equal(
		t,
		"dist/task.whl",
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
		"dist/task.jar",
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
		SyncRootPath:   dir,
		BundleRootPath: dir,
		SyncRoot:       vfs.MustNew(dir),
		Config: config.Root{
			Workspace: config.Workspace{
				FilePath: "/bundle",
			},
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"job": {
						JobSettings: jobs.JobSettings{
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
						CreatePipeline: pipelines.CreatePipeline{
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

	diags := bundle.ApplySeq(context.Background(), b, mutator.NormalizePaths(), mutator.TranslatePaths())
	require.NoError(t, diags.Error())

	assert.Equal(
		t,
		"/bundle/job/my_python_file.py",
		b.Config.Resources.Jobs["job"].Tasks[0].SparkPythonTask.PythonFile,
	)
	assert.Equal(
		t,
		"job/dist/task.jar",
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
		SyncRootPath:   dir,
		BundleRootPath: dir,
		SyncRoot:       vfs.MustNew(dir),
		Config: config.Root{
			Workspace: config.Workspace{
				FilePath: "/bundle",
			},
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"job": {
						JobSettings: jobs.JobSettings{
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

	diags := bundle.ApplySeq(context.Background(), b, mutator.NormalizePaths(), mutator.TranslatePaths())
	assert.ErrorContains(t, diags.Error(), "is not contained in sync root path")
}

func TestJobNotebookDoesNotExistError(t *testing.T) {
	dir := t.TempDir()

	b := &bundle.Bundle{
		SyncRootPath:   dir,
		BundleRootPath: dir,
		SyncRoot:       vfs.MustNew(dir),
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"job": {
						JobSettings: jobs.JobSettings{
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

	diags := bundle.ApplySeq(context.Background(), b, mutator.NormalizePaths(), mutator.TranslatePaths())
	assert.EqualError(t, diags.Error(), "notebook doesnt_exist.py not found")
}

func TestJobFileDoesNotExistError(t *testing.T) {
	dir := t.TempDir()

	b := &bundle.Bundle{
		SyncRootPath:   dir,
		BundleRootPath: dir,
		SyncRoot:       vfs.MustNew(dir),
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"job": {
						JobSettings: jobs.JobSettings{
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

	diags := bundle.ApplySeq(context.Background(), b, mutator.NormalizePaths(), mutator.TranslatePaths())
	assert.EqualError(t, diags.Error(), "file doesnt_exist.py not found")
}

func TestPipelineNotebookDoesNotExistError(t *testing.T) {
	dir := t.TempDir()

	b := &bundle.Bundle{
		SyncRootPath:   dir,
		BundleRootPath: dir,
		SyncRoot:       vfs.MustNew(dir),
		Config: config.Root{
			Resources: config.Resources{
				Pipelines: map[string]*resources.Pipeline{
					"pipeline": {
						CreatePipeline: pipelines.CreatePipeline{
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

	diags := bundle.ApplySeq(context.Background(), b, mutator.NormalizePaths(), mutator.TranslatePaths())
	assert.EqualError(t, diags.Error(), "notebook doesnt_exist.py not found")
}

func TestPipelineNotebookDoesNotExistErrorWithoutExtension(t *testing.T) {
	for _, ext := range []string{
		".py",
		".r",
		".scala",
		".sql",
		".ipynb",
		"",
	} {
		t.Run("case_"+ext, func(t *testing.T) {
			dir := t.TempDir()

			if ext != "" {
				touchEmptyFile(t, filepath.Join(dir, "foo"+ext))
			}

			b := &bundle.Bundle{
				SyncRootPath:   dir,
				BundleRootPath: dir,
				SyncRoot:       vfs.MustNew(dir),
				Config: config.Root{
					Resources: config.Resources{
						Pipelines: map[string]*resources.Pipeline{
							"pipeline": {
								CreatePipeline: pipelines.CreatePipeline{
									Libraries: []pipelines.PipelineLibrary{
										{
											Notebook: &pipelines.NotebookLibrary{
												Path: "./foo",
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
			diags := bundle.ApplySeq(context.Background(), b, mutator.NormalizePaths(), mutator.TranslatePaths())

			if ext == "" {
				assert.EqualError(t, diags.Error(), `notebook "foo" not found. Local notebook references are expected
to contain one of the following file extensions: [.py, .r, .scala, .sql, .ipynb]`)
			} else {
				assert.EqualError(t, diags.Error(), fmt.Sprintf(`notebook "foo" not found. Did you mean "foo%s"?
Local notebook references are expected to contain one of the following
file extensions: [.py, .r, .scala, .sql, .ipynb]`, ext))
			}
		})
	}
}

func TestPipelineFileDoesNotExistError(t *testing.T) {
	dir := t.TempDir()

	b := &bundle.Bundle{
		SyncRootPath:   dir,
		BundleRootPath: dir,
		SyncRoot:       vfs.MustNew(dir),
		Config: config.Root{
			Resources: config.Resources{
				Pipelines: map[string]*resources.Pipeline{
					"pipeline": {
						CreatePipeline: pipelines.CreatePipeline{
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

	diags := bundle.ApplySeq(context.Background(), b, mutator.NormalizePaths(), mutator.TranslatePaths())
	assert.EqualError(t, diags.Error(), "file doesnt_exist.py not found")
}

func TestJobSparkPythonTaskWithNotebookSourceError(t *testing.T) {
	dir := t.TempDir()
	touchNotebookFile(t, filepath.Join(dir, "my_notebook.py"))

	b := &bundle.Bundle{
		SyncRootPath:   dir,
		BundleRootPath: dir,
		SyncRoot:       vfs.MustNew(dir),
		Config: config.Root{
			Workspace: config.Workspace{
				FilePath: "/bundle",
			},
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"job": {
						JobSettings: jobs.JobSettings{
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

	diags := bundle.ApplySeq(context.Background(), b, mutator.NormalizePaths(), mutator.TranslatePaths())
	assert.ErrorContains(t, diags.Error(), `expected a file for "resources.jobs.job.tasks[0].spark_python_task.python_file" but got a notebook`)
}

func TestJobNotebookTaskWithFileSourceError(t *testing.T) {
	dir := t.TempDir()
	touchEmptyFile(t, filepath.Join(dir, "my_file.py"))

	b := &bundle.Bundle{
		SyncRootPath:   dir,
		BundleRootPath: dir,
		SyncRoot:       vfs.MustNew(dir),
		Config: config.Root{
			Workspace: config.Workspace{
				FilePath: "/bundle",
			},
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"job": {
						JobSettings: jobs.JobSettings{
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

	diags := bundle.ApplySeq(context.Background(), b, mutator.NormalizePaths(), mutator.TranslatePaths())
	assert.ErrorContains(t, diags.Error(), `expected a notebook for "resources.jobs.job.tasks[0].notebook_task.notebook_path" but got a file`)
}

func TestPipelineNotebookLibraryWithFileSourceError(t *testing.T) {
	dir := t.TempDir()
	touchEmptyFile(t, filepath.Join(dir, "my_file.py"))

	b := &bundle.Bundle{
		SyncRootPath:   dir,
		BundleRootPath: dir,
		SyncRoot:       vfs.MustNew(dir),
		Config: config.Root{
			Workspace: config.Workspace{
				FilePath: "/bundle",
			},
			Resources: config.Resources{
				Pipelines: map[string]*resources.Pipeline{
					"pipeline": {
						CreatePipeline: pipelines.CreatePipeline{
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

	diags := bundle.ApplySeq(context.Background(), b, mutator.NormalizePaths(), mutator.TranslatePaths())
	assert.ErrorContains(t, diags.Error(), `expected a notebook for "resources.pipelines.pipeline.libraries[0].notebook.path" but got a file`)
}

func TestPipelineFileLibraryWithNotebookSourceError(t *testing.T) {
	dir := t.TempDir()
	touchNotebookFile(t, filepath.Join(dir, "my_notebook.py"))

	b := &bundle.Bundle{
		SyncRootPath:   dir,
		BundleRootPath: dir,
		SyncRoot:       vfs.MustNew(dir),
		Config: config.Root{
			Workspace: config.Workspace{
				FilePath: "/bundle",
			},
			Resources: config.Resources{
				Pipelines: map[string]*resources.Pipeline{
					"pipeline": {
						CreatePipeline: pipelines.CreatePipeline{
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

	diags := bundle.ApplySeq(context.Background(), b, mutator.NormalizePaths(), mutator.TranslatePaths())
	assert.ErrorContains(t, diags.Error(), `expected a file for "resources.pipelines.pipeline.libraries[0].file.path" but got a notebook`)
}

func TestTranslatePathJobEnvironments(t *testing.T) {
	dir := t.TempDir()
	touchEmptyFile(t, filepath.Join(dir, "env1.py"))
	touchEmptyFile(t, filepath.Join(dir, "env2.py"))

	b := &bundle.Bundle{
		SyncRootPath:   dir,
		BundleRootPath: dir,
		SyncRoot:       vfs.MustNew(dir),
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"job": {
						JobSettings: jobs.JobSettings{
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

	diags := bundle.ApplySeq(context.Background(), b, mutator.NormalizePaths(), mutator.TranslatePaths())
	require.NoError(t, diags.Error())

	assert.Equal(t, "./job/dist/env1.whl", b.Config.Resources.Jobs["job"].JobSettings.Environments[0].Spec.Dependencies[0])
	assert.Equal(t, "./dist/env2.whl", b.Config.Resources.Jobs["job"].JobSettings.Environments[0].Spec.Dependencies[1])
	assert.Equal(t, "simplejson", b.Config.Resources.Jobs["job"].JobSettings.Environments[0].Spec.Dependencies[2])
	assert.Equal(t, "/Workspace/Users/foo@bar.com/test.whl", b.Config.Resources.Jobs["job"].JobSettings.Environments[0].Spec.Dependencies[3])
	assert.Equal(t, "--extra-index-url https://name:token@gitlab.com/api/v4/projects/9876/packages/pypi/simple foobar", b.Config.Resources.Jobs["job"].JobSettings.Environments[0].Spec.Dependencies[4])
	assert.Equal(t, "foobar --extra-index-url https://name:token@gitlab.com/api/v4/projects/9876/packages/pypi/simple", b.Config.Resources.Jobs["job"].JobSettings.Environments[0].Spec.Dependencies[5])
	assert.Equal(t, "https://foo@bar.com/packages/pypi/simple", b.Config.Resources.Jobs["job"].JobSettings.Environments[0].Spec.Dependencies[6])
}

func TestTranslatePathWithComplexVariables(t *testing.T) {
	dir := t.TempDir()
	b := &bundle.Bundle{
		SyncRootPath:   dir,
		BundleRootPath: dir,
		SyncRoot:       vfs.MustNew(dir),
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
						JobSettings: jobs.JobSettings{
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
	bundle.ApplyFuncContext(ctx, b, func(ctx context.Context, b *bundle.Bundle) {
		err := b.Config.Mutate(func(v dyn.Value) (dyn.Value, error) {
			p := dyn.MustPathFromString("resources.jobs.job.tasks[0]")
			return dyn.SetByPath(v, p.Append(dyn.Key("libraries")), dyn.V("${var.cluster_libraries}"))
		})
		require.NoError(t, err)
	})

	diags := bundle.ApplySeq(ctx, b,
		mutator.SetVariables(),
		mutator.ResolveVariableReferencesOnlyResources("variables"),
		mutator.NormalizePaths(),
		mutator.TranslatePaths())
	require.NoError(t, diags.Error())

	assert.Equal(
		t,
		"variables/local/whl.whl",
		b.Config.Resources.Jobs["job"].Tasks[0].Libraries[0].Whl,
	)
}

func TestTranslatePathsWithSourceLinkedDeployment(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("this test is not applicable on Windows because source-linked mode works only in the Databricks Workspace")
	}

	dir := t.TempDir()
	touchNotebookFile(t, filepath.Join(dir, "my_job_notebook.py"))
	touchNotebookFile(t, filepath.Join(dir, "my_pipeline_notebook.py"))
	touchEmptyFile(t, filepath.Join(dir, "my_python_file.py"))
	touchEmptyFile(t, filepath.Join(dir, "dist", "task.jar"))
	touchEmptyFile(t, filepath.Join(dir, "requirements.txt"))

	enabled := true
	b := &bundle.Bundle{
		SyncRootPath:   dir,
		BundleRootPath: dir,
		SyncRoot:       vfs.MustNew(dir),
		Config: config.Root{
			Workspace: config.Workspace{
				FilePath: "/bundle",
			},
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"job": {
						JobSettings: jobs.JobSettings{
							Tasks: []jobs.Task{
								{
									NotebookTask: &jobs.NotebookTask{
										NotebookPath: "my_job_notebook.py",
									},
									Libraries: []compute.Library{
										{Whl: "./dist/task.whl"},
									},
								},
								{
									NotebookTask: &jobs.NotebookTask{
										NotebookPath: "/Users/jane.doe@databricks.com/absolute_remote.py",
									},
								},
								{
									NotebookTask: &jobs.NotebookTask{
										NotebookPath: "my_job_notebook.py",
									},
									Libraries: []compute.Library{
										{Requirements: "requirements.txt"},
									},
								},
								{
									SparkPythonTask: &jobs.SparkPythonTask{
										PythonFile: "my_python_file.py",
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
						CreatePipeline: pipelines.CreatePipeline{
							Libraries: []pipelines.PipelineLibrary{
								{
									Notebook: &pipelines.NotebookLibrary{
										Path: "my_pipeline_notebook.py",
									},
								},
								{
									Notebook: &pipelines.NotebookLibrary{
										Path: "/Users/jane.doe@databricks.com/absolute_remote.py",
									},
								},
								{
									File: &pipelines.FileLibrary{
										Path: "my_python_file.py",
									},
								},
							},
						},
					},
				},
			},
			Presets: config.Presets{
				SourceLinkedDeployment: &enabled,
			},
		},
	}

	bundletest.SetLocation(b, ".", []dyn.Location{{File: filepath.Join(dir, "resource.yml")}})
	diags := bundle.ApplySeq(context.Background(), b, mutator.NormalizePaths(), mutator.TranslatePaths())
	require.NoError(t, diags.Error())

	// updated to source path
	assert.Equal(
		t,
		dir+"/my_job_notebook",
		b.Config.Resources.Jobs["job"].Tasks[0].NotebookTask.NotebookPath,
	)
	assert.Equal(
		t,
		dir+"/requirements.txt",
		b.Config.Resources.Jobs["job"].Tasks[2].Libraries[0].Requirements,
	)
	assert.Equal(
		t,
		dir+"/my_python_file.py",
		b.Config.Resources.Jobs["job"].Tasks[3].SparkPythonTask.PythonFile,
	)
	assert.Equal(
		t,
		dir+"/my_pipeline_notebook",
		b.Config.Resources.Pipelines["pipeline"].Libraries[0].Notebook.Path,
	)
	assert.Equal(
		t,
		dir+"/my_python_file.py",
		b.Config.Resources.Pipelines["pipeline"].Libraries[2].File.Path,
	)

	// left as is
	assert.Equal(
		t,
		"dist/task.whl",
		b.Config.Resources.Jobs["job"].Tasks[0].Libraries[0].Whl,
	)
	assert.Equal(
		t,
		"/Users/jane.doe@databricks.com/absolute_remote.py",
		b.Config.Resources.Jobs["job"].Tasks[1].NotebookTask.NotebookPath,
	)
	assert.Equal(
		t,
		"dist/task.jar",
		b.Config.Resources.Jobs["job"].Tasks[4].Libraries[0].Jar,
	)
	assert.Equal(
		t,
		"dbfs:/bundle/dist/task_remote.jar",
		b.Config.Resources.Jobs["job"].Tasks[5].Libraries[0].Jar,
	)
	assert.Equal(
		t,
		"/Users/jane.doe@databricks.com/absolute_remote.py",
		b.Config.Resources.Pipelines["pipeline"].Libraries[1].Notebook.Path,
	)
}
