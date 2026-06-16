package terraform_dabs_map_test

import (
	"testing"

	"github.com/databricks/cli/bundle/terraform_dabs_map"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTerraformPathToDABs(t *testing.T) {
	tests := []struct {
		group       string
		terrPath    string
		dabsPath    string
		noRoundtrip bool // if true, DABsPathToTerraform(dabsPath) != terrPath (non-invertible path)
		expectErr   bool // if true, translation must return an error (Terraform-only field)
	}{
		// Top-level renames - jobs
		{
			group:    "jobs",
			terrPath: "task",
			dabsPath: "tasks",
		},
		{
			group:    "jobs",
			terrPath: "job_cluster",
			dabsPath: "job_clusters",
		},
		{
			group:    "jobs",
			terrPath: "parameter",
			dabsPath: "parameters",
		},
		{
			group:    "jobs",
			terrPath: "environment",
			dabsPath: "environments",
		},

		// Nested renames - jobs
		{
			group:    "jobs",
			terrPath: "task.library",
			dabsPath: "tasks.libraries",
		},
		{
			group:    "jobs",
			terrPath: "task.for_each_task.task.library",
			dabsPath: "tasks.for_each_task.task.libraries",
		},

		// git_source: segment itself unchanged, children renamed
		{
			group:    "jobs",
			terrPath: "git_source.branch",
			dabsPath: "git_source.git_branch",
		},
		{
			group:    "jobs",
			terrPath: "git_source.url",
			dabsPath: "git_source.git_url",
		},

		// Unknown fields pass through unchanged
		{
			group:    "jobs",
			terrPath: "name",
			dabsPath: "name",
		},
		{
			group:    "jobs",
			terrPath: "format",
			dabsPath: "format",
		},

		// After an unrecognised segment, remaining segments pass through as-is
		{
			group:    "jobs",
			terrPath: "task.new_cluster.node_type_id",
			dabsPath: "tasks.new_cluster.node_type_id",
		},

		// Array index passes through without advancing the rename tree
		{
			group:    "jobs",
			terrPath: "task[0].library",
			dabsPath: "tasks[0].libraries",
		},

		// Pipelines
		{
			group:    "pipelines",
			terrPath: "cluster",
			dabsPath: "clusters",
		},
		{
			group:    "pipelines",
			terrPath: "library",
			dabsPath: "libraries",
		},
		{
			group:    "pipelines",
			terrPath: "notification",
			dabsPath: "notifications",
		},

		// Unknown group: path unchanged
		{
			group:    "unknown_group",
			terrPath: "task",
			dabsPath: "task",
		},

		// postgres resources: spec wrapper is unwrapped
		{
			group:    "postgres_projects",
			terrPath: "spec.display_name",
			dabsPath: "display_name",
		},
		{
			group:    "postgres_projects",
			terrPath: "spec.default_endpoint_settings.no_suspension",
			dabsPath: "default_endpoint_settings.no_suspension",
		},
		{
			group:    "postgres_branches",
			terrPath: "spec.expire_time",
			dabsPath: "expire_time",
		},
		{
			group:    "postgres_catalogs",
			terrPath: "spec.postgres_database",
			dabsPath: "postgres_database",
		},
		{
			group:    "postgres_endpoints",
			terrPath: "spec.endpoint_type",
			dabsPath: "endpoint_type",
		},

		// TF root-level paths (status, timestamps, IDs) pass through unchanged and
		// round-trip correctly: DABsPathToTerraform recognises them as root fields.
		{
			group:    "postgres_projects",
			terrPath: "status.display_name",
			dabsPath: "status.display_name",
		},
		{
			group:    "postgres_projects",
			terrPath: "create_time",
			dabsPath: "create_time",
		},
		{
			group:    "postgres_projects",
			terrPath: "project_id",
			dabsPath: "project_id",
		},
		{
			group:    "postgres_projects",
			terrPath: "name",
			dabsPath: "name",
		},

		// Terraform-only fields: must return an error
		{
			group:     "jobs",
			terrPath:  "always_running",
			expectErr: true,
		},
		{
			group:     "jobs",
			terrPath:  "new_cluster.node_type_id",
			expectErr: true,
		},
		{
			group:     "pipelines",
			terrPath:  "url",
			expectErr: true,
		},
		// Wildcard map fields: any key under a map-typed TF-only field is also TF-only
		{
			group:     "jobs",
			terrPath:  "new_cluster.spark_conf.MY_KEY",
			expectErr: true,
		},
		{
			group:     "jobs",
			terrPath:  "notebook_task.base_parameters.my_param",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.group+"/"+tt.terrPath, func(t *testing.T) {
			path, err := structpath.ParsePath(tt.terrPath)
			require.NoError(t, err)

			result, err := terraform_dabs_map.TerraformPathToDABs(tt.group, path)
			if tt.expectErr {
				require.Error(t, err)
				assert.Nil(t, result)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, result)
			assert.Equal(t, tt.dabsPath, result.String())

			if !tt.noRoundtrip {
				back, err := terraform_dabs_map.DABsPathToTerraform(tt.group, result)
				require.NoError(t, err)
				require.NotNil(t, back)
				assert.Equal(t, tt.terrPath, back.String(), "roundtrip DABsPathToTerraform(TerraformPathToDABs(terrPath))")
			}
		})
	}
}

func TestTerraformPathToDABsNilPath(t *testing.T) {
	result, err := terraform_dabs_map.TerraformPathToDABs("jobs", nil)
	assert.NoError(t, err)
	assert.Nil(t, result)
}

func TestDABsPathToTerraformErrors(t *testing.T) {
	// The non-error cases are already covered by the roundtrip check in
	// TestTerraformPathToDABs.  This test covers only DABs-only fields, which
	// have no TF equivalent and must return an error.
	tests := []struct {
		group    string
		dabsPath string
	}{
		{"jobs", "tasks.new_cluster.autotermination_minutes"},
		{"pipelines", "dry_run"},
	}

	for _, tt := range tests {
		t.Run(tt.group+"/"+tt.dabsPath, func(t *testing.T) {
			path, err := structpath.ParsePath(tt.dabsPath)
			require.NoError(t, err)
			result, err := terraform_dabs_map.DABsPathToTerraform(tt.group, path)
			require.Error(t, err)
			assert.Nil(t, result)
		})
	}
}

func TestDABsPathToTerraformNilPath(t *testing.T) {
	result, err := terraform_dabs_map.DABsPathToTerraform("jobs", nil)
	assert.NoError(t, err)
	assert.Nil(t, result)
}
