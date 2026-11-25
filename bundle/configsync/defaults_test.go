package configsync

import (
	"testing"

	"github.com/databricks/cli/bundle/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSyncRulesFilesAreNamedAfterResourceTypes(t *testing.T) {
	for _, name := range rulesFileNames() {
		_, ok := config.ResourceTypeAs[config.ConfigResource](rulesResourceType(name))
		assert.True(t, ok, "%s is not named after a resource type", name)
	}
}

func TestParseRulesFileRejectsMalformedFiles(t *testing.T) {
	tests := []struct {
		name string
		data string
		err  string
	}{
		{
			name: "empty file",
			data: "",
			err:  "rules/jobs.yml: file is empty",
		},
		{
			name: "more than one document",
			data: "prefixed_name_fields: [name]\n---\nprefixed_name_fields: [other]\n",
			err:  "rules/jobs.yml: more than one YAML document",
		},
		{
			name: "empty server_side_defaults field",
			data: "server_side_defaults:\n  - field: edit_mode\n    skip: always\n  - skip: always\n",
			err:  "rules/jobs.yml: server_side_defaults entry 2: field is empty",
		},
		{
			name: "empty reset_values field",
			data: "reset_values:\n  - value: {enabled: false}\n",
			err:  "rules/jobs.yml: reset_values entry 1: field is empty",
		},
		{
			name: "empty prefixed_name_fields field",
			data: "prefixed_name_fields: ['']\n",
			err:  "rules/jobs.yml: prefixed_name_fields entry 1: field is empty",
		},
		{
			name: "duplicate server_side_defaults field",
			data: "server_side_defaults:\n  - field: edit_mode\n    skip: always\n  - field: edit_mode\n    value: UI_LOCKED\n",
			err:  `rules/jobs.yml: server_side_defaults entry 2: field "edit_mode" is listed twice`,
		},
		{
			name: "duplicate reset_values field",
			data: "reset_values:\n  - field: queue\n    value: {enabled: false}\n  - field: queue\n    value: {enabled: true}\n",
			err:  `rules/jobs.yml: reset_values entry 2: field "queue" is listed twice`,
		},
		{
			name: "duplicate prefixed_name_fields field",
			data: "prefixed_name_fields: [name, name]\n",
			err:  `rules/jobs.yml: prefixed_name_fields entry 2: field "name" is listed twice`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseRulesFile("rules/jobs.yml", []byte(tt.data))
			assert.EqualError(t, err, tt.err)
		})
	}
}

func TestParseRulesFileAllowsSameFieldInDifferentLists(t *testing.T) {
	data := "server_side_defaults:\n  - field: name\n    skip: always\nprefixed_name_fields: [name]\n"
	rules, err := parseRulesFile("rules/jobs.yml", []byte(data))
	require.NoError(t, err)
	assert.Equal(t, []string{"resources.jobs.*.name"}, rules.prefixedNameFields)
	assert.Len(t, rules.serverSideDefaults, 1)
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
