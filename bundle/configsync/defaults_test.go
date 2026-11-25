package configsync

import (
	"testing"

	"github.com/databricks/cli/bundle/config"
	"github.com/stretchr/testify/assert"
)

func TestSyncRulesFilesAreNamedAfterResourceTypes(t *testing.T) {
	for _, name := range rulesFileNames() {
		_, ok := config.ResourceTypeAs[config.ConfigResource](rulesResourceType(name))
		assert.True(t, ok, "%s is not named after a resource type", name)
	}
}

// The values in the rules files are decoded from YAML but compared with normalized
// remote values, so their types must match: an int64 for an integer, an empty map for {}.
func TestShouldSkipFieldMatchesRuleValues(t *testing.T) {
	tests := []struct {
		path  string
		value any
		want  bool
	}{
		{"resources.sql_warehouses.my_warehouse.min_num_clusters", int64(1), true},
		{"resources.sql_warehouses.my_warehouse.min_num_clusters", int64(2), false},
		{"resources.jobs.my_job.webhook_notifications", map[string]any{}, true},
		{"resources.jobs.my_job.email_notifications", map[string]any{"no_alert_for_skipped_runs": false}, true},
		{"resources.jobs.my_job.email_notifications", map[string]any{"on_failure": []any{"a@example.com"}}, false},
		{"resources.jobs.my_job.tasks[task_key='main'].timeout_seconds", int64(0), true},
		{"resources.jobs.my_job.tasks[task_key='main'].disabled", false, true},
		{"resources.pipelines.my_pipeline.continuous", true, false},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, shouldSkipField(tt.path, tt.value, false), "%s: %v", tt.path, tt.value)
	}
}
