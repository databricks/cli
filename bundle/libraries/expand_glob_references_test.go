package libraries

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/internal/bundletest"
	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/databricks/databricks-sdk-go/service/jobs"
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
							Tasks: []jobs.Task{
								{
									TaskKey: "task",
									Libraries: []compute.Library{
										{
											Whl: "whl/*.whl",
										},
										{
											Whl: "/Workspace/path/to/whl/my.whl",
										},
										{
											Jar: "./jar/*.jar",
										},
										{
											Egg: "egg/*.egg",
										},
										{
											Jar: "/Workspace/path/to/jar/*.jar",
										},
										{
											Whl: "/some/full/path/to/whl/*.whl",
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

	bundletest.SetLocation(b, ".", []dyn.Location{{File: filepath.Join(dir, "resource.yml")}})

	diags := bundle.Apply(context.Background(), b, ExpandGlobReferences())
	require.Empty(t, diags)

	job := b.Config.Resources.Jobs["job"]
	task := job.JobSettings.Tasks[0]
	require.Equal(t, []compute.Library{
		{
			Whl: filepath.Join("whl", "my1.whl"),
		},
		{
			Whl: filepath.Join("whl", "my2.whl"),
		},
		{
			Whl: "/Workspace/path/to/whl/my.whl",
		},
		{
			Jar: filepath.Join("jar", "my1.jar"),
		},
		{
			Jar: filepath.Join("jar", "my2.jar"),
		},
		{
			Egg: "egg/*.egg",
		},
		{
			Jar: "/Workspace/path/to/jar/*.jar",
		},
		{
			Whl: "/some/full/path/to/whl/*.whl",
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
							Tasks: []jobs.Task{
								{
									TaskKey: "task",
									ForEachTask: &jobs.ForEachTask{
										Task: jobs.Task{
											Libraries: []compute.Library{
												{
													Whl: "whl/*.whl",
												},
												{
													Whl: "/Workspace/path/to/whl/my.whl",
												},
												{
													Jar: "./jar/*.jar",
												},
												{
													Egg: "egg/*.egg",
												},
												{
													Jar: "/Workspace/path/to/jar/*.jar",
												},
												{
													Whl: "/some/full/path/to/whl/*.whl",
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

	bundletest.SetLocation(b, ".", []dyn.Location{{File: filepath.Join(dir, "resource.yml")}})

	diags := bundle.Apply(context.Background(), b, ExpandGlobReferences())
	require.Empty(t, diags)

	job := b.Config.Resources.Jobs["job"]
	task := job.JobSettings.Tasks[0].ForEachTask.Task
	require.Equal(t, []compute.Library{
		{
			Whl: filepath.Join("whl", "my1.whl"),
		},
		{
			Whl: filepath.Join("whl", "my2.whl"),
		},
		{
			Whl: "/Workspace/path/to/whl/my.whl",
		},
		{
			Jar: filepath.Join("jar", "my1.jar"),
		},
		{
			Jar: filepath.Join("jar", "my2.jar"),
		},
		{
			Egg: "egg/*.egg",
		},
		{
			Jar: "/Workspace/path/to/jar/*.jar",
		},
		{
			Whl: "/some/full/path/to/whl/*.whl",
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
							Tasks: []jobs.Task{
								{
									TaskKey:        "task",
									EnvironmentKey: "env",
								},
							},
							Environments: []jobs.JobEnvironment{
								{
									EnvironmentKey: "env",
									Spec: &compute.Environment{
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

	bundletest.SetLocation(b, ".", []dyn.Location{{File: filepath.Join(dir, "resource.yml")}})

	diags := bundle.Apply(context.Background(), b, ExpandGlobReferences())
	require.Empty(t, diags)

	job := b.Config.Resources.Jobs["job"]
	env := job.JobSettings.Environments[0]
	require.Equal(t, []string{
		filepath.Join("whl", "my1.whl"),
		filepath.Join("whl", "my2.whl"),
		"/Workspace/path/to/whl/my.whl",
		filepath.Join("jar", "my1.jar"),
		filepath.Join("jar", "my2.jar"),
		"/some/local/path/to/whl/*.whl",
	}, env.Spec.Dependencies)
}
