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
		group     string
		input     string // TF path
		output    string // DABs path
		roundtrip bool   // if true, DABsPathToTerraform(output) must equal input
	}{
		// Top-level renames - jobs
		{"jobs", "task", "tasks", true},
		{"jobs", "job_cluster", "job_clusters", true},
		{"jobs", "parameter", "parameters", true},
		{"jobs", "environment", "environments", true},

		// Nested renames - jobs
		{"jobs", "task.library", "tasks.libraries", true},
		{"jobs", "task.for_each_task.task.library", "tasks.for_each_task.task.libraries", true},

		// git_source: segment itself unchanged, children renamed
		{"jobs", "git_source.branch", "git_source.git_branch", true},
		{"jobs", "git_source.url", "git_source.git_url", true},

		// Unknown fields pass through unchanged
		{"jobs", "name", "name", true},
		{"jobs", "format", "format", true},

		// After an unrecognised segment, remaining segments pass through as-is
		{"jobs", "task.new_cluster.node_type_id", "tasks.new_cluster.node_type_id", true},

		// Array index passes through without advancing the rename tree
		{"jobs", "task[0].library", "tasks[0].libraries", true},

		// Pipelines
		{"pipelines", "cluster", "clusters", true},
		{"pipelines", "library", "libraries", true},
		{"pipelines", "notification", "notifications", true},

		// Unknown group: path unchanged
		{"unknown_group", "task", "task", true},

		// postgres resources: spec wrapper is unwrapped
		{"postgres_projects", "spec.display_name", "display_name", true},
		{"postgres_projects", "spec.default_endpoint_settings.no_suspension", "default_endpoint_settings.no_suspension", true},
		{"postgres_branches", "spec.expire_time", "expire_time", true},
		{"postgres_catalogs", "spec.postgres_database", "postgres_database", true},
		{"postgres_endpoints", "spec.endpoint_type", "endpoint_type", true},

		// TF-computed paths (status, timestamps) pass through unchanged but are not
		// roundtrippable: DABsPathToTerraform would incorrectly prepend the spec wrapper.
		{"postgres_projects", "status.display_name", "status.display_name", false},
		{"postgres_projects", "create_time", "create_time", false},
	}

	for _, tt := range tests {
		t.Run(tt.group+"/"+tt.input, func(t *testing.T) {
			path, err := structpath.ParsePath(tt.input)
			require.NoError(t, err)

			result := terraform_dabs_map.TerraformPathToDABs(tt.group, path)
			require.NotNil(t, result)
			assert.Equal(t, tt.output, result.String())

			if tt.roundtrip {
				back := terraform_dabs_map.DABsPathToTerraform(tt.group, result)
				require.NotNil(t, back)
				assert.Equal(t, tt.input, back.String(), "roundtrip DABsPathToTerraform(TerraformPathToDABs(input))")
			}
		})
	}
}

func TestTerraformPathToDABsNilPath(t *testing.T) {
	assert.Nil(t, terraform_dabs_map.TerraformPathToDABs("jobs", nil))
}

func TestDABsPathToTerraform(t *testing.T) {
	tests := []struct {
		group  string
		input  string // DABs path
		output string // TF path
	}{
		// Jobs renames (inverse)
		{"jobs", "tasks", "task"},
		{"jobs", "tasks.libraries", "task.library"},
		{"jobs", "tasks.for_each_task.task.libraries", "task.for_each_task.task.library"},
		{"jobs", "job_clusters", "job_cluster"},
		{"jobs", "parameters", "parameter"},
		{"jobs", "environments", "environment"},
		{"jobs", "git_source.git_branch", "git_source.branch"},
		{"jobs", "git_source.git_url", "git_source.url"},

		// After unrecognised segment, remaining pass through
		{"jobs", "tasks.new_cluster.node_type_id", "task.new_cluster.node_type_id"},

		// Array index preserved
		{"jobs", "tasks[0].libraries", "task[0].library"},

		// Pipelines
		{"pipelines", "clusters", "cluster"},
		{"pipelines", "libraries", "library"},
		{"pipelines", "notifications", "notification"},

		// Unknown fields pass through
		{"jobs", "name", "name"},

		// Unknown group: path unchanged
		{"unknown_group", "tasks", "tasks"},

		// postgres resources: prepend spec wrapper
		{"postgres_projects", "display_name", "spec.display_name"},
		{"postgres_projects", "default_endpoint_settings.no_suspension", "spec.default_endpoint_settings.no_suspension"},
		{"postgres_branches", "expire_time", "spec.expire_time"},
		{"postgres_catalogs", "postgres_database", "spec.postgres_database"},
		{"postgres_endpoints", "endpoint_type", "spec.endpoint_type"},
	}

	for _, tt := range tests {
		t.Run(tt.group+"/"+tt.input, func(t *testing.T) {
			path, err := structpath.ParsePath(tt.input)
			require.NoError(t, err)

			result := terraform_dabs_map.DABsPathToTerraform(tt.group, path)
			require.NotNil(t, result)
			assert.Equal(t, tt.output, result.String())
		})
	}
}

func TestDABsPathToTerraformNilPath(t *testing.T) {
	assert.Nil(t, terraform_dabs_map.DABsPathToTerraform("jobs", nil))
}
