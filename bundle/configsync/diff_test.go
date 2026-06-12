package configsync

import (
	"testing"

	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertChangeDesc(t *testing.T) {
	tests := []struct {
		name         string
		resourceType string
		path         string
		cd           *deployplan.ChangeDesc
		wantOp       OperationType
		wantVal      any
	}{
		{
			name:         "add: new in remote only",
			resourceType: "jobs",
			path:         "description",
			cd:           &deployplan.ChangeDesc{Old: nil, New: nil, Remote: "remote-desc"},
			wantOp:       OperationAdd,
			wantVal:      "remote-desc",
		},
		{
			name:         "remove: in config, missing from remote",
			resourceType: "jobs",
			path:         "description",
			cd:           &deployplan.ChangeDesc{Old: "state-desc", New: "config-desc", Remote: nil},
			wantOp:       OperationRemove,
			wantVal:      nil,
		},
		{
			name:         "replace: differs between config and remote",
			resourceType: "jobs",
			path:         "description",
			cd:           &deployplan.ChangeDesc{Old: "state-desc", New: "config-desc", Remote: "remote-desc"},
			wantOp:       OperationReplace,
			wantVal:      "remote-desc",
		},
		{
			name:         "skip: absent everywhere",
			resourceType: "jobs",
			path:         "description",
			cd:           &deployplan.ChangeDesc{Old: nil, New: nil, Remote: nil},
			wantOp:       OperationSkip,
			wantVal:      nil,
		},
		// Regression: rename-back-and-forth. State holds the old key (user did not
		// redeploy after the first sync), config holds the intermediate key, and
		// remote now matches the original. The element at this path is missing from
		// config, so the change must be Add — Replace would error in resolveSelectors
		// because the keyed element no longer exists in the YAML.
		{
			name:         "add: rename-back path, state has it but config does not",
			resourceType: "jobs",
			path:         "tasks[task_key='new_task']",
			cd: &deployplan.ChangeDesc{
				Old:    map[string]any{"task_key": "new_task"},
				New:    nil,
				Remote: map[string]any{"task_key": "new_task"},
			},
			wantOp:  OperationAdd,
			wantVal: map[string]any{"task_key": "new_task"},
		},
		{
			name:         "skip: state has it, config and remote do not",
			resourceType: "jobs",
			path:         "tasks[task_key='gone']",
			cd: &deployplan.ChangeDesc{
				Old:    map[string]any{"task_key": "gone"},
				New:    nil,
				Remote: nil,
			},
			wantOp:  OperationSkip,
			wantVal: nil,
		},
		// The plan re-promotes etag drift to Update via
		// ResourceDashboard.OverrideChangeDesc, so sync must skip it based on
		// the ignore_remote_changes metadata, not the plan action.
		{
			name:         "skip: dashboard etag is output-only",
			resourceType: "dashboards",
			path:         "etag",
			cd:           &deployplan.ChangeDesc{Old: "etag-1", New: nil, Remote: "etag-2"},
			wantOp:       OperationSkip,
			wantVal:      nil,
		},
		// create_time comes from resources.generated.yml (spec:output_only).
		{
			name:         "skip: dashboard create_time is output-only (generated rule)",
			resourceType: "dashboards",
			path:         "create_time",
			cd:           &deployplan.ChangeDesc{Old: nil, New: nil, Remote: "2025-01-01T00:00:00Z"},
			wantOp:       OperationSkip,
			wantVal:      nil,
		},
		{
			name:         "skip: backend default not in config",
			resourceType: "jobs",
			path:         "performance_target",
			cd:           &deployplan.ChangeDesc{Old: nil, New: nil, Remote: "PERFORMANCE_OPTIMIZED"},
			wantOp:       OperationSkip,
			wantVal:      nil,
		},
		{
			name:         "replace: backend default value but field is set in config",
			resourceType: "jobs",
			path:         "performance_target",
			cd:           &deployplan.ChangeDesc{Old: "STANDARD", New: "STANDARD", Remote: "PERFORMANCE_OPTIMIZED"},
			wantOp:       OperationReplace,
			wantVal:      "PERFORMANCE_OPTIMIZED",
		},
		{
			name:         "add: non-default value of backend default field",
			resourceType: "jobs",
			path:         "performance_target",
			cd:           &deployplan.ChangeDesc{Old: nil, New: nil, Remote: "STANDARD"},
			wantOp:       OperationAdd,
			wantVal:      "STANDARD",
		},
		{
			name:         "skip: edit_mode is CLI-managed",
			resourceType: "jobs",
			path:         "edit_mode",
			cd:           &deployplan.ChangeDesc{Old: nil, New: nil, Remote: "UI_LOCKED"},
			wantOp:       OperationSkip,
			wantVal:      nil,
		},
		// node_type_id drift on a cluster that omits it in config comes from a
		// cluster policy (jobs/get returns the policy-resolved value in stored
		// settings) or, for standalone clusters, from clusters/get reporting the
		// pool's node type; it must not be written to YAML.
		{
			name:         "skip: node_type_id materialized remotely for cluster omitting it",
			resourceType: "jobs",
			path:         "tasks[task_key='t1'].new_cluster.node_type_id",
			cd:           &deployplan.ChangeDesc{Old: nil, New: nil, Remote: "i3.xlarge"},
			wantOp:       OperationSkip,
			wantVal:      nil,
		},
		{
			name:         "skip: standalone cluster node_type_id materialized for pool-backed cluster",
			resourceType: "clusters",
			path:         "node_type_id",
			cd:           &deployplan.ChangeDesc{Old: nil, New: nil, Remote: "c5d.xlarge"},
			wantOp:       OperationSkip,
			wantVal:      nil,
		},
		{
			name:         "replace: queue removed remotely resets to disabled",
			resourceType: "jobs",
			path:         "queue",
			cd:           &deployplan.ChangeDesc{Old: map[string]any{"enabled": true}, New: map[string]any{"enabled": true}, Remote: nil},
			wantOp:       OperationReplace,
			wantVal:      map[string]any{"enabled": false},
		},
		// A task added remotely as a whole: backend defaults (run_if,
		// timeout_seconds, data_security_mode, the deprecated
		// no_alert_for_skipped_runs the backend populates in
		// email_notifications) and ignored fields (aws_attributes) are
		// stripped; maps that become empty are pruned (email_notifications,
		// webhook_notifications); node_type_id must be kept or the synced
		// config would be undeployable.
		{
			name:         "add: remotely added task is filtered by metadata",
			resourceType: "jobs",
			path:         "tasks[task_key='new_task']",
			cd: &deployplan.ChangeDesc{
				Old: nil,
				New: nil,
				Remote: map[string]any{
					"task_key":              "new_task",
					"run_if":                "ALL_SUCCESS",
					"timeout_seconds":       0,
					"email_notifications":   map[string]any{"no_alert_for_skipped_runs": false},
					"webhook_notifications": map[string]any{},
					"new_cluster": map[string]any{
						"spark_version":      "13.3.x-scala2.12",
						"node_type_id":       "i3.xlarge",
						"num_workers":        1,
						"data_security_mode": "SINGLE_USER",
						"aws_attributes":     map[string]any{"availability": "SPOT_WITH_FALLBACK"},
					},
				},
			},
			wantOp: OperationAdd,
			wantVal: map[string]any{
				"task_key": "new_task",
				"new_cluster": map[string]any{
					"spark_version": "13.3.x-scala2.12",
					"node_type_id":  "i3.xlarge",
					"num_workers":   int64(1),
				},
			},
		},
		// When a field set in config filters entirely to backend defaults on
		// the remote side, the operation is Remove (drop the field from YAML),
		// not Replace-with-{}.
		{
			name:         "remove: config field whose remote value filters to empty",
			resourceType: "jobs",
			path:         "email_notifications",
			cd: &deployplan.ChangeDesc{
				Old:    map[string]any{"on_failure": []any{"someone@example.com"}},
				New:    map[string]any{"on_failure": []any{"someone@example.com"}},
				Remote: map[string]any{"no_alert_for_skipped_runs": false},
			},
			wantOp:  OperationRemove,
			wantVal: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, err := structpath.ParsePath(tt.path)
			require.NoError(t, err)
			got, err := convertChangeDesc(tt.resourceType, path, tt.cd)
			require.NoError(t, err)
			assert.Equal(t, tt.wantOp, got.Operation)
			assert.Equal(t, tt.wantVal, got.Value)
		})
	}
}

func TestFilterEntityDefaults(t *testing.T) {
	tests := []struct {
		name     string
		basePath string
		value    any
		want     any
	}{
		{
			name:     "array element filtering to empty is kept to preserve arity",
			basePath: "tasks",
			value: []any{
				map[string]any{"task_key": "t1", "run_if": "ALL_SUCCESS"},
				map[string]any{"run_if": "ALL_SUCCESS"},
			},
			want: []any{
				map[string]any{"task_key": "t1"},
				map[string]any{},
			},
		},
		{
			name:     "map field filtering to empty is pruned",
			basePath: "tasks[task_key='t1']",
			value: map[string]any{
				"task_key":            "t1",
				"email_notifications": map[string]any{"no_alert_for_skipped_runs": false},
			},
			want: map[string]any{"task_key": "t1"},
		},
		{
			name:     "whole value filtering to empty becomes nil",
			basePath: "email_notifications",
			value:    map[string]any{"no_alert_for_skipped_runs": false},
			want:     nil,
		},
		{
			name:     "scalar is returned unchanged",
			basePath: "description",
			value:    "desc",
			want:     "desc",
		},
		// node_type_id matches backend_defaults (policy-backed clusters
		// materialize it remotely) but is in fieldsKeptForSync: a remotely
		// added cluster carrying it explicitly, even alongside policy_id,
		// must keep it or the synced config would be undeployable.
		{
			name:     "remotely added policy-backed cluster keeps node_type_id",
			basePath: "job_clusters[0].new_cluster",
			value: map[string]any{
				"policy_id":          "ABC123",
				"node_type_id":       "i3.xlarge",
				"spark_version":      "13.3.x-scala2.12",
				"num_workers":        1,
				"data_security_mode": "SINGLE_USER",
			},
			want: map[string]any{
				"policy_id":     "ABC123",
				"node_type_id":  "i3.xlarge",
				"spark_version": "13.3.x-scala2.12",
				"num_workers":   1,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			basePath, err := structpath.ParsePath(tt.basePath)
			require.NoError(t, err)
			got := filterEntityDefaults("jobs", basePath, tt.value)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestShouldSkipForSync(t *testing.T) {
	tests := []struct {
		name           string
		resourceType   string
		path           string
		value          any
		hasConfigValue bool
		want           bool
	}{
		{
			name:         "ignore_remote_changes skips regardless of value",
			resourceType: "dashboards",
			path:         "etag",
			value:        "etag-2",
			want:         true,
		},
		{
			name:           "ignore_remote_changes skips even when field is in config",
			resourceType:   "dashboards",
			path:           "serialized_dashboard",
			value:          "{}",
			hasConfigValue: true,
			want:           true,
		},
		{
			name:         "generated ignore_remote_changes rule",
			resourceType: "dashboards",
			path:         "lifecycle_state",
			value:        "ACTIVE",
			want:         true,
		},
		{
			name:         "ignore_remote_changes matches nested path by prefix",
			resourceType: "jobs",
			path:         "tasks[task_key='t1'].new_cluster.aws_attributes.availability",
			value:        "SPOT",
			want:         true,
		},
		{
			name:         "backend default with matching value",
			resourceType: "jobs",
			path:         "performance_target",
			value:        "PERFORMANCE_OPTIMIZED",
			want:         true,
		},
		{
			name:         "backend default with non-matching value",
			resourceType: "jobs",
			path:         "performance_target",
			value:        "STANDARD",
			want:         false,
		},
		{
			name:           "backend default not applied when field is in config",
			resourceType:   "jobs",
			path:           "performance_target",
			value:          "PERFORMANCE_OPTIMIZED",
			hasConfigValue: true,
			want:           false,
		},
		{
			name:         "backend default without values matches any value",
			resourceType: "jobs",
			path:         "run_as",
			value:        map[string]any{"user_name": "someone@example.com"},
			want:         true,
		},
		{
			name:         "backend default with wildcard pattern",
			resourceType: "jobs",
			path:         "tasks[task_key='t1'].run_if",
			value:        "ALL_SUCCESS",
			want:         true,
		},
		// node_type_id matches backend_defaults; nested filtering overrides
		// this via fieldsKeptForSync (see TestFilterEntityDefaults).
		{
			name:         "backend default node_type_id",
			resourceType: "jobs",
			path:         "tasks[task_key='t1'].new_cluster.node_type_id",
			value:        "i3.xlarge",
			want:         true,
		},
		{
			name:         "regular field is not skipped",
			resourceType: "jobs",
			path:         "description",
			value:        "some description",
			want:         false,
		},
		{
			name:         "unknown resource type is not skipped",
			resourceType: "unknown",
			path:         "anything",
			value:        "value",
			want:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, err := structpath.ParsePath(tt.path)
			require.NoError(t, err)
			got := shouldSkipForSync(tt.resourceType, path, tt.value, tt.hasConfigValue)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestResetValueIfNeeded(t *testing.T) {
	tests := []struct {
		name         string
		resourceType string
		path         string
		value        any
		want         any
	}{
		{
			name:         "whole queue field is reset",
			resourceType: "jobs",
			path:         "queue",
			value:        nil,
			want:         map[string]any{"enabled": false},
		},
		{
			name:         "queue subfield keeps its own value",
			resourceType: "jobs",
			path:         "queue.enabled",
			value:        false,
			want:         false,
		},
		{
			name:         "unrelated field is unchanged",
			resourceType: "jobs",
			path:         "description",
			value:        "desc",
			want:         "desc",
		},
		{
			name:         "other resource type is unchanged",
			resourceType: "pipelines",
			path:         "queue",
			value:        nil,
			want:         nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, err := structpath.ParsePath(tt.path)
			require.NoError(t, err)
			got := resetValueIfNeeded(tt.resourceType, path, tt.value)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestStripNamePrefix(t *testing.T) {
	tests := []struct {
		name         string
		resourceType string
		path         string
		value        any
		prefix       string
		want         any
	}{
		{
			name:         "job name with normal prefix",
			resourceType: "jobs",
			path:         "name",
			value:        "[dev user] my_job",
			prefix:       "[dev user] ",
			want:         "my_job",
		},
		{
			name:         "pipeline name with normal prefix",
			resourceType: "pipelines",
			path:         "name",
			value:        "[dev user] my_pipeline",
			prefix:       "[dev user] ",
			want:         "my_pipeline",
		},
		{
			name:         "dashboard display_name with prefix",
			resourceType: "dashboards",
			path:         "display_name",
			value:        "[dev user] my_dash",
			prefix:       "[dev user] ",
			want:         "my_dash",
		},
		{
			name:         "name does not start with prefix",
			resourceType: "jobs",
			path:         "name",
			value:        "my_job",
			prefix:       "[dev user] ",
			want:         "my_job",
		},
		{
			name:         "empty prefix is noop",
			resourceType: "jobs",
			path:         "name",
			value:        "[dev user] my_job",
			prefix:       "",
			want:         "[dev user] my_job",
		},
		{
			name:         "non-name field is not stripped",
			resourceType: "jobs",
			path:         "description",
			value:        "[dev user] some description",
			prefix:       "[dev user] ",
			want:         "[dev user] some description",
		},
		{
			name:         "name of resource type without prefix support",
			resourceType: "experiments",
			path:         "name",
			value:        "[dev user] my_experiment",
			prefix:       "[dev user] ",
			want:         "[dev user] my_experiment",
		},
		{
			name:         "non-string value is unchanged",
			resourceType: "jobs",
			path:         "name",
			value:        42,
			prefix:       "[dev user] ",
			want:         42,
		},
		{
			name:         "nil value is unchanged",
			resourceType: "jobs",
			path:         "name",
			value:        nil,
			prefix:       "[dev user] ",
			want:         nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, err := structpath.ParsePath(tt.path)
			require.NoError(t, err)
			got := stripNamePrefix(tt.resourceType, path, tt.value, tt.prefix)
			assert.Equal(t, tt.want, got)
		})
	}
}
