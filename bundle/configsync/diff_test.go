package configsync

import (
	"testing"

	"github.com/databricks/cli/bundle/deployplan"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertChangeDesc(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		cd      *deployplan.ChangeDesc
		wantOp  OperationType
		wantVal any
	}{
		{
			name:    "add: new in remote only",
			path:    "resources.jobs.my_job.description",
			cd:      &deployplan.ChangeDesc{Old: nil, New: nil, Remote: "remote-desc"},
			wantOp:  OperationAdd,
			wantVal: "remote-desc",
		},
		{
			name:    "remove: in config, missing from remote",
			path:    "resources.jobs.my_job.description",
			cd:      &deployplan.ChangeDesc{Old: "state-desc", New: "config-desc", Remote: nil},
			wantOp:  OperationRemove,
			wantVal: nil,
		},
		{
			name:    "replace: differs between config and remote",
			path:    "resources.jobs.my_job.description",
			cd:      &deployplan.ChangeDesc{Old: "state-desc", New: "config-desc", Remote: "remote-desc"},
			wantOp:  OperationReplace,
			wantVal: "remote-desc",
		},
		{
			name:    "skip: absent everywhere",
			path:    "resources.jobs.my_job.description",
			cd:      &deployplan.ChangeDesc{Old: nil, New: nil, Remote: nil},
			wantOp:  OperationSkip,
			wantVal: nil,
		},
		// Regression: rename-back-and-forth. State holds the old key (user did not
		// redeploy after the first sync), config holds the intermediate key, and
		// remote now matches the original. The element at this path is missing from
		// config, so the change must be Add — Replace would error in resolveSelectors
		// because the keyed element no longer exists in the YAML.
		{
			name: "add: rename-back path, state has it but config does not",
			path: "resources.jobs.my_job.tasks[task_key='new_task']",
			cd: &deployplan.ChangeDesc{
				Old:    map[string]any{"task_key": "new_task"},
				New:    nil,
				Remote: map[string]any{"task_key": "new_task"},
			},
			wantOp:  OperationAdd,
			wantVal: map[string]any{"task_key": "new_task"},
		},
		{
			name: "skip: state has it, config and remote do not",
			path: "resources.jobs.my_job.tasks[task_key='gone']",
			cd: &deployplan.ChangeDesc{
				Old:    map[string]any{"task_key": "gone"},
				New:    nil,
				Remote: nil,
			},
			wantOp:  OperationSkip,
			wantVal: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := convertChangeDesc(tt.path, tt.cd)
			require.NoError(t, err)
			assert.Equal(t, tt.wantOp, got.Operation)
			assert.Equal(t, tt.wantVal, got.Value)
		})
	}
}

func TestShouldSkipFieldDashboardEtag(t *testing.T) {
	// etag is output-only: alwaysSkip must apply regardless of hasConfigValue,
	// otherwise out-of-band etag drift is written back into the user's YAML.
	path := "resources.dashboards.my_dashboard.etag"
	assert.True(t, shouldSkipField(path, "some-etag", false))
	assert.True(t, shouldSkipField(path, "some-etag", true))
}

func TestStripNamePrefix(t *testing.T) {
	tests := []struct {
		name   string
		path   string
		value  any
		prefix string
		want   any
	}{
		{
			name:   "job name with normal prefix",
			path:   "resources.jobs.my_job.name",
			value:  "[dev user] my_job",
			prefix: "[dev user] ",
			want:   "my_job",
		},
		{
			name:   "pipeline name with normal prefix",
			path:   "resources.pipelines.my_pipeline.name",
			value:  "[dev user] my_pipeline",
			prefix: "[dev user] ",
			want:   "my_pipeline",
		},
		{
			name:   "dashboard display_name with prefix",
			path:   "resources.dashboards.my_dash.display_name",
			value:  "[dev user] my_dash",
			prefix: "[dev user] ",
			want:   "my_dash",
		},
		{
			name:   "name does not start with prefix",
			path:   "resources.jobs.my_job.name",
			value:  "my_job",
			prefix: "[dev user] ",
			want:   "my_job",
		},
		{
			name:   "empty prefix is noop",
			path:   "resources.jobs.my_job.name",
			value:  "[dev user] my_job",
			prefix: "",
			want:   "[dev user] my_job",
		},
		{
			name:   "non-name field is not stripped",
			path:   "resources.jobs.my_job.description",
			value:  "[dev user] some description",
			prefix: "[dev user] ",
			want:   "[dev user] some description",
		},
		{
			name:   "non-string value is unchanged",
			path:   "resources.jobs.my_job.name",
			value:  42,
			prefix: "[dev user] ",
			want:   42,
		},
		{
			name:   "nil value is unchanged",
			path:   "resources.jobs.my_job.name",
			value:  nil,
			prefix: "[dev user] ",
			want:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripNamePrefix(tt.path, tt.value, tt.prefix)
			assert.Equal(t, tt.want, got)
		})
	}
}

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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchPattern(tt.pattern, tt.path)
			assert.Equal(t, tt.want, got, "matchPattern(%q, %q)", tt.pattern, tt.path)
		})
	}
}
