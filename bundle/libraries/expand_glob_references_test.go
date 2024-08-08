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
		RootPath: dir,
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"job": {
						JobSettings: &jobs.JobSettings{
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
									},
								},
							},
						},
					},
				},
			},
		},
	}

	bundletest.SetLocation(b, ".", filepath.Join(dir, "resource.yml"))

	diags := bundle.Apply(context.Background(), b, ExpandGlobReferences())
	require.Empty(t, diags)

	job := b.Config.Resources.Jobs["job"]
	task := job.JobSettings.Tasks[0]
	require.Equal(t, []compute.Library{
		{
			Whl: filepath.Join(dir, "whl", "my1.whl"),
		},
		{
			Whl: filepath.Join(dir, "whl", "my2.whl"),
		},
		{
			Whl: "/Workspace/path/to/whl/my.whl",
		},
		{
			Jar: filepath.Join(dir, "jar", "my1.jar"),
		},
		{
			Jar: filepath.Join(dir, "jar", "my2.jar"),
		},
		{
			Egg: "egg/*.egg",
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
		RootPath: dir,
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"job": {
						JobSettings: &jobs.JobSettings{
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

	bundletest.SetLocation(b, ".", filepath.Join(dir, "resource.yml"))

	diags := bundle.Apply(context.Background(), b, ExpandGlobReferences())
	require.Empty(t, diags)

	job := b.Config.Resources.Jobs["job"]
	env := job.JobSettings.Environments[0]
	require.Equal(t, []string{
		filepath.Join(dir, "whl", "my1.whl"),
		filepath.Join(dir, "whl", "my2.whl"),
		"/Workspace/path/to/whl/my.whl",
		filepath.Join(dir, "jar", "my1.jar"),
		filepath.Join(dir, "jar", "my2.jar"),
	}, env.Spec.Dependencies)

}
