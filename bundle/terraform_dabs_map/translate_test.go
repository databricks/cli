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
		group  string
		input  string
		output string
	}{
		// Top-level renames - jobs
		{"jobs", "task", "tasks"},
		{"jobs", "job_cluster", "job_clusters"},
		{"jobs", "parameter", "parameters"},
		{"jobs", "environment", "environments"},

		// Nested renames - jobs
		{"jobs", "task.library", "tasks.libraries"},
		{"jobs", "task.for_each_task.task.library", "tasks.for_each_task.task.libraries"},

		// git_source: segment itself unchanged, children renamed
		{"jobs", "git_source.branch", "git_source.git_branch"},
		{"jobs", "git_source.url", "git_source.git_url"},

		// Unknown fields pass through unchanged
		{"jobs", "name", "name"},
		{"jobs", "format", "format"},

		// After an unrecognised segment, remaining segments pass through as-is
		{"jobs", "task.new_cluster.node_type_id", "tasks.new_cluster.node_type_id"},

		// Array index passes through without advancing the rename tree
		{"jobs", "task[0].library", "tasks[0].libraries"},

		// Pipelines
		{"pipelines", "cluster", "clusters"},
		{"pipelines", "library", "libraries"},
		{"pipelines", "notification", "notifications"},

		// Unknown group: path unchanged
		{"unknown_group", "task", "task"},
	}

	for _, tt := range tests {
		t.Run(tt.group+"/"+tt.input, func(t *testing.T) {
			path, err := structpath.ParsePath(tt.input)
			require.NoError(t, err)

			result := terraform_dabs_map.TerraformPathToDABs(tt.group, path)
			require.NotNil(t, result)
			assert.Equal(t, tt.output, result.String())
		})
	}
}

func TestTerraformPathToDABsNilPath(t *testing.T) {
	assert.Nil(t, terraform_dabs_map.TerraformPathToDABs("jobs", nil))
}
