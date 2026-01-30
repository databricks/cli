package configsync

import (
	"testing"

	"github.com/databricks/cli/bundle/deployplan"
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

func TestFilterFunctions(t *testing.T) {
	tests := []struct {
		name   string
		filter func(*deployplan.ChangeDesc) bool
		desc   *deployplan.ChangeDesc
		want   bool
	}{
		{
			name:   "isBoolEqual(false) with false value",
			filter: isBoolEqual(false),
			desc:   &deployplan.ChangeDesc{Remote: false},
			want:   true,
		},
		{
			name:   "isBoolEqual(false) with true value",
			filter: isBoolEqual(false),
			desc:   &deployplan.ChangeDesc{Remote: true},
			want:   false,
		},
		{
			name:   "isBoolEqual(false) with non-bool value",
			filter: isBoolEqual(false),
			desc:   &deployplan.ChangeDesc{Remote: "false"},
			want:   false,
		},
		{
			name:   "isStringEqual with matching value",
			filter: isStringEqual("PERFORMANCE_OPTIMIZED"),
			desc:   &deployplan.ChangeDesc{Remote: "PERFORMANCE_OPTIMIZED"},
			want:   true,
		},
		{
			name:   "isStringEqual with non-matching value",
			filter: isStringEqual("PERFORMANCE_OPTIMIZED"),
			desc:   &deployplan.ChangeDesc{Remote: "STANDARD"},
			want:   false,
		},
		{
			name:   "isStringEqual with nil remote",
			filter: isStringEqual(""),
			desc:   &deployplan.ChangeDesc{Remote: nil},
			want:   true,
		},
		{
			name:   "defaultIfNotSpecified with nil Old and New",
			filter: defaultIfNotSpecified,
			desc:   &deployplan.ChangeDesc{Old: nil, New: nil, Remote: "something"},
			want:   true,
		},
		{
			name:   "defaultIfNotSpecified with non-nil New",
			filter: defaultIfNotSpecified,
			desc:   &deployplan.ChangeDesc{Old: nil, New: "value", Remote: "something"},
			want:   false,
		},
		{
			name:   "defaultIfNotSpecified with non-nil Old",
			filter: defaultIfNotSpecified,
			desc:   &deployplan.ChangeDesc{Old: "value", New: nil, Remote: "something"},
			want:   false,
		},
		{
			name:   "alwaysDefault always returns true",
			filter: alwaysDefault,
			desc:   &deployplan.ChangeDesc{Remote: "anything"},
			want:   true,
		},
		{
			name:   "isZero with zero int",
			filter: isZero,
			desc:   &deployplan.ChangeDesc{Remote: 0},
			want:   true,
		},
		{
			name:   "isZero with non-zero int",
			filter: isZero,
			desc:   &deployplan.ChangeDesc{Remote: 42},
			want:   false,
		},
		{
			name:   "isZero with nil",
			filter: isZero,
			desc:   &deployplan.ChangeDesc{Remote: nil},
			want:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.filter(tt.desc)
			assert.Equal(t, tt.want, got)
		})
	}
}
