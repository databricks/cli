package configsync

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMatchPattern(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		path    string
		want    bool
	}{
		{
			name:    "exact match",
			pattern: "timeout_seconds",
			path:    "timeout_seconds",
			want:    true,
		},
		{
			name:    "exact match no match",
			pattern: "timeout_seconds",
			path:    "other_field",
			want:    false,
		},
		{
			name:    "array wildcard match",
			pattern: "tasks[*].run_if",
			path:    "tasks[task_key='my_task'].run_if",
			want:    true,
		},
		{
			name:    "array wildcard no match",
			pattern: "tasks[*].run_if",
			path:    "tasks[task_key='my_task'].disabled",
			want:    false,
		},
		{
			name:    "nested array wildcard match",
			pattern: "tasks[*].new_cluster.azure_attributes.availability",
			path:    "tasks[task_key='task1'].new_cluster.azure_attributes.availability",
			want:    true,
		},
		{
			name:    "job_clusters array wildcard match",
			pattern: "job_clusters[*].new_cluster.aws_attributes.availability",
			path:    "job_clusters[job_cluster_key='cluster1'].new_cluster.aws_attributes.availability",
			want:    true,
		},
		{
			name:    "wildcard segment match",
			pattern: "*.timeout_seconds",
			path:    "timeout_seconds",
			want:    false,
		},
		{
			name:    "different array prefix no match",
			pattern: "tasks[*].run_if",
			path:    "jobs[task_key='my_task'].run_if",
			want:    false,
		},
		{
			name:    "nested path match",
			pattern: "tasks[*].notebook_task.source",
			path:    "tasks[task_key='notebook'].notebook_task.source",
			want:    true,
		},
		{
			name:    "path shorter than pattern",
			pattern: "tasks[*].new_cluster.azure_attributes",
			path:    "tasks[task_key='task1'].new_cluster",
			want:    false,
		},
		{
			name:    "path longer than pattern",
			pattern: "tasks[*].new_cluster",
			path:    "tasks[task_key='task1'].new_cluster.azure_attributes",
			want:    false,
		},
		{
			name:    "pipeline channel match",
			pattern: "channel",
			path:    "channel",
			want:    true,
		},
		{
			name:    "pipeline continuous match",
			pattern: "continuous",
			path:    "continuous",
			want:    true,
		},
		{
			name:    "job email_notifications.no_alert_for_skipped_runs match",
			pattern: "email_notifications.no_alert_for_skipped_runs",
			path:    "email_notifications.no_alert_for_skipped_runs",
			want:    true,
		},
		{
			name:    "job environments match",
			pattern: "environments",
			path:    "environments",
			want:    true,
		},
		{
			name:    "job performance_target match",
			pattern: "performance_target",
			path:    "performance_target",
			want:    true,
		},
		{
			name:    "task email_notifications match",
			pattern: "tasks[*].email_notifications",
			path:    "tasks[task_key='my_task'].email_notifications",
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchPattern(tt.pattern, tt.path)
			assert.Equal(t, tt.want, got, "matchPattern(%q, %q)", tt.pattern, tt.path)
		})
	}
}

func TestShouldSkipField(t *testing.T) {
	tests := []struct {
		name  string
		path  string
		value any
		want  bool
	}{
		{
			name:  "timeout_seconds with nil",
			path:  "timeout_seconds",
			value: nil,
			want:  true,
		},
		{
			name:  "timeout_seconds with non-zero",
			path:  "timeout_seconds",
			value: 42,
			want:  false,
		},
		{
			name:  "tasks timeout_seconds with nil",
			path:  "tasks[task_key='my_task'].timeout_seconds",
			value: nil,
			want:  true,
		},
		{
			name:  "performance_target with matching value",
			path:  "performance_target",
			value: "PERFORMANCE_OPTIMIZED",
			want:  true,
		},
		{
			name:  "performance_target with non-matching value",
			path:  "performance_target",
			value: "STANDARD",
			want:  false,
		},
		{
			name:  "tasks run_if with matching value",
			path:  "tasks[task_key='my_task'].run_if",
			value: "ALL_SUCCESS",
			want:  true,
		},
		{
			name:  "tasks run_if with non-matching value",
			path:  "tasks[task_key='my_task'].run_if",
			value: "ALL_DONE",
			want:  false,
		},
		{
			name:  "tasks disabled with false",
			path:  "tasks[task_key='my_task'].disabled",
			value: false,
			want:  true,
		},
		{
			name:  "tasks disabled with true",
			path:  "tasks[task_key='my_task'].disabled",
			value: true,
			want:  false,
		},
		{
			name:  "email_notifications with default value",
			path:  "email_notifications",
			value: map[string]any{"no_alert_for_skipped_runs": false},
			want:  true,
		},
		{
			name:  "email_notifications with non-default value",
			path:  "email_notifications",
			value: map[string]any{"no_alert_for_skipped_runs": true},
			want:  false,
		},
		{
			name:  "edit_mode always skipped",
			path:  "edit_mode",
			value: "anything",
			want:  true,
		},
		{
			name:  "run_as always skipped",
			path:  "run_as",
			value: map[string]any{"user_name": "someone"},
			want:  true,
		},
		{
			name:  "aws_attributes always skipped",
			path:  "tasks[task_key='my_task'].new_cluster.aws_attributes",
			value: map[string]any{"availability": "SPOT"},
			want:  true,
		},
		{
			name:  "continuous with false",
			path:  "continuous",
			value: false,
			want:  true,
		},
		{
			name:  "continuous with true",
			path:  "continuous",
			value: true,
			want:  false,
		},
		{
			name:  "unmatched field not skipped",
			path:  "some_other_field",
			value: "anything",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shouldSkipField(tt.path, tt.value)
			assert.Equal(t, tt.want, got)
		})
	}
}
