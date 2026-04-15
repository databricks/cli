package apps

import (
	"path/filepath"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/internal/bundletest"
	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/vfs"
	"github.com/databricks/databricks-sdk-go/service/apps"
	"github.com/stretchr/testify/require"
)

func TestAppsValidateSameSourcePath(t *testing.T) {
	tmpDir := t.TempDir()
	testutil.Touch(t, tmpDir, "app1", "app.py")

	b := &bundle.Bundle{
		BundleRootPath: tmpDir,
		SyncRootPath:   tmpDir,
		SyncRoot:       vfs.MustNew(tmpDir),
		Config: config.Root{
			Workspace: config.Workspace{
				FilePath: "/foo/bar/",
			},
			Resources: config.Resources{
				Apps: map[string]*resources.App{
					"app1": {
						App: apps.App{
							Name: "app1",
						},
						SourceCodePath: "./app1",
					},
					"app2": {
						App: apps.App{
							Name: "app2",
						},
						SourceCodePath: "./app1",
					},
				},
			},
		},
	}

	bundletest.SetLocation(b, ".", []dyn.Location{{File: filepath.Join(tmpDir, "databricks.yml")}})

	diags := bundle.ApplySeq(t.Context(), b, mutator.TranslatePaths(), Validate())
	require.Len(t, diags, 1)
	require.Equal(t, "Duplicate app source code path", diags[0].Summary)
	require.Contains(t, diags[0].Detail, "has the same source code path as app resource")
}

func TestAppsValidateResourcePermissionsWarning(t *testing.T) {
	testCases := []struct {
		name         string
		appResources []apps.AppResource
		resources    config.Resources
		wantWarning  bool
		wantSummary  string
	}{
		{
			name: "job with permissions warns",
			appResources: []apps.AppResource{
				{Name: "my-job", Job: &apps.AppResourceJob{Id: "${resources.jobs.my_job.id}", Permission: "CAN_MANAGE_RUN"}},
			},
			resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"my_job": {Permissions: []resources.JobPermission{{Level: "CAN_MANAGE", UserName: "someone@example.com"}}},
				},
			},
			wantWarning: true,
			wantSummary: `app "my_app" references jobs "my_job"`,
		},
		{
			name: "job with SP already included no warning",
			appResources: []apps.AppResource{
				{Name: "my-job", Job: &apps.AppResourceJob{Id: "${resources.jobs.my_job.id}", Permission: "CAN_MANAGE_RUN"}},
			},
			resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"my_job": {Permissions: []resources.JobPermission{
						{Level: "CAN_MANAGE", UserName: "someone@example.com"},
						{Level: "CAN_MANAGE_RUN", ServicePrincipalName: "${resources.apps.my_app.service_principal_client_id}"},
					}},
				},
			},
			wantWarning: false,
		},
		{
			name: "job without permissions no warning",
			appResources: []apps.AppResource{
				{Name: "my-job", Job: &apps.AppResourceJob{Id: "${resources.jobs.my_job.id}", Permission: "CAN_MANAGE_RUN"}},
			},
			resources: config.Resources{
				Jobs: map[string]*resources.Job{"my_job": {}},
			},
			wantWarning: false,
		},
		{
			name: "sql_warehouse with permissions warns",
			appResources: []apps.AppResource{
				{Name: "my-wh", SqlWarehouse: &apps.AppResourceSqlWarehouse{Id: "${resources.sql_warehouses.my_wh.id}", Permission: "CAN_USE"}},
			},
			resources: config.Resources{
				SqlWarehouses: map[string]*resources.SqlWarehouse{
					"my_wh": {Permissions: []resources.SqlWarehousePermission{{Level: "CAN_MANAGE", UserName: "someone@example.com"}}},
				},
			},
			wantWarning: true,
			wantSummary: `app "my_app" references sql_warehouses "my_wh"`,
		},
		{
			name: "serving_endpoint with permissions warns",
			appResources: []apps.AppResource{
				{Name: "my-ep", ServingEndpoint: &apps.AppResourceServingEndpoint{Name: "${resources.model_serving_endpoints.my_ep.name}", Permission: "CAN_QUERY"}},
			},
			resources: config.Resources{
				ModelServingEndpoints: map[string]*resources.ModelServingEndpoint{
					"my_ep": {Permissions: []resources.ModelServingEndpointPermission{{Level: "CAN_MANAGE", UserName: "someone@example.com"}}},
				},
			},
			wantWarning: true,
			wantSummary: `app "my_app" references model_serving_endpoints "my_ep"`,
		},
		{
			name: "experiment with permissions warns",
			appResources: []apps.AppResource{
				{Name: "my-exp", Experiment: &apps.AppResourceExperiment{ExperimentId: "${resources.experiments.my_exp.experiment_id}", Permission: "CAN_READ"}},
			},
			resources: config.Resources{
				Experiments: map[string]*resources.MlflowExperiment{
					"my_exp": {Permissions: []resources.MlflowExperimentPermission{{Level: "CAN_MANAGE", UserName: "someone@example.com"}}},
				},
			},
			wantWarning: true,
			wantSummary: `app "my_app" references experiments "my_exp"`,
		},
		{
			name: "postgres with permissions warns",
			appResources: []apps.AppResource{
				{Name: "my-pg", Postgres: &apps.AppResourcePostgres{Branch: "${resources.postgres_projects.my_pg.name}", Permission: "CAN_CONNECT_AND_CREATE"}},
			},
			resources: config.Resources{
				PostgresProjects: map[string]*resources.PostgresProject{
					"my_pg": {Permissions: []resources.Permission{{Level: "CAN_MANAGE", UserName: "someone@example.com"}}},
				},
			},
			wantWarning: true,
			wantSummary: `app "my_app" references postgres_projects "my_pg"`,
		},
		{
			name: "postgres without permissions no warning",
			appResources: []apps.AppResource{
				{Name: "my-pg", Postgres: &apps.AppResourcePostgres{Branch: "${resources.postgres_projects.my_pg.name}", Permission: "CAN_CONNECT_AND_CREATE"}},
			},
			resources: config.Resources{
				PostgresProjects: map[string]*resources.PostgresProject{
					"my_pg": {},
				},
			},
			wantWarning: false,
		},
		{
			name: "non-reference id no warning",
			appResources: []apps.AppResource{
				{Name: "my-job", Job: &apps.AppResourceJob{Id: "12345", Permission: "CAN_MANAGE_RUN"}},
			},
			resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"my_job": {Permissions: []resources.JobPermission{{Level: "CAN_MANAGE", UserName: "someone@example.com"}}},
				},
			},
			wantWarning: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			testutil.Touch(t, tmpDir, "app1", "app.py")

			tc.resources.Apps = map[string]*resources.App{
				"my_app": {
					App: apps.App{
						Name:      "my_app",
						Resources: tc.appResources,
					},
					SourceCodePath: "./app1",
				},
			}

			b := &bundle.Bundle{
				BundleRootPath: tmpDir,
				SyncRootPath:   tmpDir,
				SyncRoot:       vfs.MustNew(tmpDir),
				Config: config.Root{
					Workspace: config.Workspace{FilePath: "/foo/bar/"},
					Resources: tc.resources,
				},
			}

			bundletest.SetLocation(b, ".", []dyn.Location{{File: filepath.Join(tmpDir, "databricks.yml")}})

			diags := bundle.ApplySeq(t.Context(), b, Validate())
			warnings := diags.Filter(diag.Warning)
			if tc.wantWarning {
				require.Len(t, warnings, 1)
				require.Contains(t, warnings[0].Summary, tc.wantSummary)
				require.Contains(t, warnings[0].Detail, "service_principal_name: ${resources.apps.my_app.service_principal_client_id}")
				require.Equal(t, dyn.MustPathFromString("resources.apps.my_app"), warnings[0].Paths[0])
			} else {
				require.Empty(t, warnings)
			}
		})
	}
}

func TestAppsValidateMultipleGitSourceAppsNoDuplicate(t *testing.T) {
	tmpDir := t.TempDir()

	b := &bundle.Bundle{
		BundleRootPath: tmpDir,
		SyncRootPath:   tmpDir,
		SyncRoot:       vfs.MustNew(tmpDir),
		Config: config.Root{
			Workspace: config.Workspace{
				FilePath: "/foo/bar/",
			},
			Resources: config.Resources{
				Apps: map[string]*resources.App{
					"app1": {
						App: apps.App{
							Name: "app1",
						},
						GitSource: &apps.GitSource{
							Branch: "main",
						},
					},
					"app2": {
						App: apps.App{
							Name: "app2",
						},
						GitSource: &apps.GitSource{
							Branch: "dev",
						},
					},
				},
			},
		},
	}

	bundletest.SetLocation(b, ".", []dyn.Location{{File: filepath.Join(tmpDir, "databricks.yml")}})

	diags := bundle.ApplySeq(t.Context(), b, Validate())
	require.Empty(t, diags)
}

func TestAppsValidateBothSourceCodePathAndGitSource(t *testing.T) {
	tmpDir := t.TempDir()
	testutil.Touch(t, tmpDir, "app1", "app.py")

	b := &bundle.Bundle{
		BundleRootPath: tmpDir,
		SyncRootPath:   tmpDir,
		SyncRoot:       vfs.MustNew(tmpDir),
		Config: config.Root{
			Workspace: config.Workspace{
				FilePath: "/foo/bar/",
			},
			Resources: config.Resources{
				Apps: map[string]*resources.App{
					"app1": {
						App: apps.App{
							Name: "app1",
						},
						SourceCodePath: "./app1",
						GitSource: &apps.GitSource{
							Branch: "main",
						},
					},
				},
			},
		},
	}

	bundletest.SetLocation(b, ".", []dyn.Location{{File: filepath.Join(tmpDir, "databricks.yml")}})

	diags := bundle.ApplySeq(t.Context(), b, mutator.TranslatePaths(), Validate())
	require.Len(t, diags, 1)
	require.Equal(t, "Both source_code_path and git_source fields are set", diags[0].Summary)
	require.Contains(t, diags[0].Detail, "should have either source_code_path or git_source field, not both")
}
