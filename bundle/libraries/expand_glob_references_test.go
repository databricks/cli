package libraries

import (
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
	"github.com/databricks/databricks-sdk-go/service/pipelines"
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

	diags := bundle.Apply(t.Context(), b, ExpandGlobReferences())
	require.Empty(t, diags)

	job := b.Config.Resources.Jobs["job"]
	task := job.Tasks[0]
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

	diags := bundle.Apply(t.Context(), b, ExpandGlobReferences())
	require.Empty(t, diags)

	job := b.Config.Resources.Jobs["job"]
	task := job.Tasks[0].ForEachTask.Task
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

func TestSplitWheelExtras(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantPath   string
		wantExtras string
	}{
		{
			name:       "basic extras",
			input:      "./dist/foo.whl[train]",
			wantPath:   "./dist/foo.whl",
			wantExtras: "[train]",
		},
		{
			name:       "multiple extras",
			input:      "./dist/foo.whl[train,test]",
			wantPath:   "./dist/foo.whl",
			wantExtras: "[train,test]",
		},
		{
			name:       "brackets in the middle are not extras",
			input:      "./dist/foo[12].whl",
			wantPath:   "./dist/foo[12].whl",
			wantExtras: "",
		},
		{
			name:       "brackets in the middle with trailing extras",
			input:      "./dist/foo[12].whl[train]",
			wantPath:   "./dist/foo[12].whl",
			wantExtras: "[train]",
		},
		{
			name:       "non-bracket suffix is not extras",
			input:      "./dist/foo.whl.bak",
			wantPath:   "./dist/foo.whl.bak",
			wantExtras: "",
		},
		{
			name:       "glob characters with extras",
			input:      "./dist/*.whl[train]",
			wantPath:   "./dist/*.whl",
			wantExtras: "[train]",
		},
		{
			name:       "no extras",
			input:      "./dist/foo.whl",
			wantPath:   "./dist/foo.whl",
			wantExtras: "",
		},
		{
			name:       "case insensitive extension",
			input:      "./dist/foo.WHL[train]",
			wantPath:   "./dist/foo.WHL",
			wantExtras: "[train]",
		},
		{
			name:       "empty extras",
			input:      "./dist/foo.whl[]",
			wantPath:   "./dist/foo.whl",
			wantExtras: "[]",
		},
		{
			name:       "unclosed bracket is not extras",
			input:      "./dist/foo.whl[train",
			wantPath:   "./dist/foo.whl[train",
			wantExtras: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, extras := splitWheelExtras(tt.input)
			assert.Equal(t, tt.wantPath, path)
			assert.Equal(t, tt.wantExtras, extras)
		})
	}
}

func TestGlobReferencesExpandedForEnvironmentsDepsWithExtras(t *testing.T) {
	dir := t.TempDir()
	testutil.Touch(t, dir, "whl", "my1.whl")
	testutil.Touch(t, dir, "whl", "my2.whl")

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
											"./whl/*.whl[train]",
											"./whl/my1.whl[train,test]",
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

	diags := bundle.Apply(t.Context(), b, ExpandGlobReferences())
	require.Empty(t, diags)

	job := b.Config.Resources.Jobs["job"]
	env := job.Environments[0]
	require.Equal(t, []string{
		filepath.Join("whl", "my1.whl") + "[train]",
		filepath.Join("whl", "my2.whl") + "[train]",
		filepath.Join("whl", "my1.whl") + "[train,test]",
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
							Tasks: []jobs.Task{
								{
									TaskKey: "task",
									Libraries: []compute.Library{
										{Whl: "/Workspace/remote.whl"},
									},
								},
							},
						},
					},
				},
				Pipelines: map[string]*resources.Pipeline{
					"pipeline": {
						CreatePipeline: pipelines.CreatePipeline{
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

	loc := dyn.Location{File: filepath.Join(dir, "resource.yml"), Line: 10, Column: 5}
	bundletest.SetLocation(b, ".", []dyn.Location{loc})

	diags := bundle.Apply(t.Context(), b, ExpandGlobReferences())
	require.Empty(t, diags)

	libs, err := dyn.GetByPath(b.Config.Value(), dyn.MustPathFromString("resources.jobs.job.tasks[0].libraries"))
	require.NoError(t, err)
	assert.Equal(t, loc.File, libs.Location().File)

	deps, err := dyn.GetByPath(b.Config.Value(), dyn.MustPathFromString("resources.pipelines.pipeline.environment.dependencies"))
	require.NoError(t, err)
	assert.Equal(t, loc.File, deps.Location().File)
}
