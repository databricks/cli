package libraries

import (
	"path/filepath"
	"testing"

	"github.com/databricks/cli/libs/sdktypes/jobs/v2"
	"github.com/databricks/cli/libs/sdktypes/pipelines/v2"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/internal/bundletest"
	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGlobReferencesExpandedForTaskLibraries(t *testing.T) {
	dir := t.TempDir()
	testutil.Touch(t, dir, "whl", "my1.whl")
	testutil.Touch(t, dir, "whl", "my2.whl")
	testutil.Touch(t, dir, "jar", "my1.jar")
	testutil.Touch(t, dir, "jar", "my2.jar")

	b := &bundle.Bundle{
		SyncRootPath: dir,
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"job": {
						JobSettings: jobs.JobSettings{
							Tasks: []jobs.TaskSettings{
								{
									TaskKey: new("task"),
									Libraries: []jobs.Library{
										{
											Whl: new("whl/*.whl"),
										},
										{
											Whl: new("/Workspace/path/to/whl/my.whl"),
										},
										{
											Jar: new("./jar/*.jar"),
										},
										{
											Egg: new("egg/*.egg"),
										},
										{
											Jar: new("/Workspace/path/to/jar/*.jar"),
										},
										{
											Whl: new("/some/full/path/to/whl/*.whl"),
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

	bundletest.SetLocation(b, ".", []diag.Location{{File: filepath.Join(dir, "resource.yml")}})

	diags := bundle.Apply(t.Context(), b, ExpandGlobReferences())
	require.Empty(t, diags)

	job := b.Config.Resources.Jobs["job"]
	task := job.Tasks[0]
	require.Equal(t, []jobs.Library{
		{
			Whl: new(filepath.Join("whl", "my1.whl")),
		},
		{
			Whl: new(filepath.Join("whl", "my2.whl")),
		},
		{
			Whl: new("/Workspace/path/to/whl/my.whl"),
		},
		{
			Jar: new(filepath.Join("jar", "my1.jar")),
		},
		{
			Jar: new(filepath.Join("jar", "my2.jar")),
		},
		{
			Egg: new("egg/*.egg"),
		},
		{
			Jar: new("/Workspace/path/to/jar/*.jar"),
		},
		{
			Whl: new("/some/full/path/to/whl/*.whl"),
		},
	}, task.Libraries)
}

func TestGlobReferencesExpandedForForeachTaskLibraries(t *testing.T) {
	dir := t.TempDir()
	testutil.Touch(t, dir, "whl", "my1.whl")
	testutil.Touch(t, dir, "whl", "my2.whl")
	testutil.Touch(t, dir, "jar", "my1.jar")
	testutil.Touch(t, dir, "jar", "my2.jar")

	b := &bundle.Bundle{
		SyncRootPath: dir,
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"job": {
						JobSettings: jobs.JobSettings{
							Tasks: []jobs.TaskSettings{
								{
									TaskKey: new("task"),
									ForEachTask: &jobs.ForEachTask{
										Task: &jobs.TaskSettings{
											Libraries: []jobs.Library{
												{
													Whl: new("whl/*.whl"),
												},
												{
													Whl: new("/Workspace/path/to/whl/my.whl"),
												},
												{
													Jar: new("./jar/*.jar"),
												},
												{
													Egg: new("egg/*.egg"),
												},
												{
													Jar: new("/Workspace/path/to/jar/*.jar"),
												},
												{
													Whl: new("/some/full/path/to/whl/*.whl"),
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
		},
	}

	bundletest.SetLocation(b, ".", []diag.Location{{File: filepath.Join(dir, "resource.yml")}})

	diags := bundle.Apply(t.Context(), b, ExpandGlobReferences())
	require.Empty(t, diags)

	job := b.Config.Resources.Jobs["job"]
	task := job.Tasks[0].ForEachTask.Task
	require.Equal(t, []jobs.Library{
		{
			Whl: new(filepath.Join("whl", "my1.whl")),
		},
		{
			Whl: new(filepath.Join("whl", "my2.whl")),
		},
		{
			Whl: new("/Workspace/path/to/whl/my.whl"),
		},
		{
			Jar: new(filepath.Join("jar", "my1.jar")),
		},
		{
			Jar: new(filepath.Join("jar", "my2.jar")),
		},
		{
			Egg: new("egg/*.egg"),
		},
		{
			Jar: new("/Workspace/path/to/jar/*.jar"),
		},
		{
			Whl: new("/some/full/path/to/whl/*.whl"),
		},
	}, task.Libraries)
}

func TestGlobReferencesExpandedForEnvironmentsDeps(t *testing.T) {
	dir := t.TempDir()
	testutil.Touch(t, dir, "whl", "my1.whl")
	testutil.Touch(t, dir, "whl", "my2.whl")
	testutil.Touch(t, dir, "jar", "my1.jar")
	testutil.Touch(t, dir, "jar", "my2.jar")

	b := &bundle.Bundle{
		SyncRootPath: dir,
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"job": {
						JobSettings: jobs.JobSettings{
							Tasks: []jobs.TaskSettings{
								{
									TaskKey:        new("task"),
									EnvironmentKey: new("env"),
								},
							},
							Environments: []jobs.JobEnvironment{
								{
									EnvironmentKey: new("env"),
									Spec: &jobs.Environment{
										Dependencies: []string{
											"./whl/*.whl",
											"/Workspace/path/to/whl/my.whl",
											"./jar/*.jar",
											"/some/local/path/to/whl/*.whl",
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

	bundletest.SetLocation(b, ".", []diag.Location{{File: filepath.Join(dir, "resource.yml")}})

	diags := bundle.Apply(t.Context(), b, ExpandGlobReferences())
	require.Empty(t, diags)

	job := b.Config.Resources.Jobs["job"]
	env := job.Environments[0]
	require.Equal(t, []string{
		filepath.Join("whl", "my1.whl"),
		filepath.Join("whl", "my2.whl"),
		"/Workspace/path/to/whl/my.whl",
		filepath.Join("jar", "my1.jar"),
		filepath.Join("jar", "my2.jar"),
		"/some/local/path/to/whl/*.whl",
	}, env.Spec.Dependencies)
}

func TestExpandGlobReferencesPreservesLocations(t *testing.T) {
	dir := t.TempDir()

	b := &bundle.Bundle{
		SyncRootPath: dir,
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"job": {
						JobSettings: jobs.JobSettings{
							Tasks: []jobs.TaskSettings{
								{
									TaskKey: new("task"),
									Libraries: []jobs.Library{
										{Whl: new("/Workspace/remote.whl")},
									},
								},
							},
						},
					},
				},
				Pipelines: map[string]*resources.Pipeline{
					"pipeline": {
						CreatePipelineRequest: pipelines.CreatePipelineRequest{
							Environment: &pipelines.PipelinesEnvironment{
								Dependencies: []string{
									"--editable /Workspace/foo",
								},
							},
						},
					},
				},
			},
		},
	}

	loc := diag.Location{File: filepath.Join(dir, "resource.yml"), Line: 10, Column: 5}
	bundletest.SetLocation(b, ".", []diag.Location{loc})

	diags := bundle.Apply(t.Context(), b, ExpandGlobReferences())
	require.Empty(t, diags)

	libs, err := dyn.GetByPath(b.Config.Value(), dyn.MustPathFromString("resources.jobs.job.tasks[0].libraries"))
	require.NoError(t, err)
	assert.Equal(t, loc.File, libs.Location().File)

	deps, err := dyn.GetByPath(b.Config.Value(), dyn.MustPathFromString("resources.pipelines.pipeline.environment.dependencies"))
	require.NoError(t, err)
	assert.Equal(t, loc.File, deps.Location().File)
}
