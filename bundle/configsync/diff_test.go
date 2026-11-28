package configsync

import (
	"errors"
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/direct/dresources"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/yamlloader"
	"github.com/databricks/cli/libs/structs/structvar"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/databricks/databricks-sdk-go/service/serving"
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

// ExtractChanges must not return permissions/grants sub-resources under their own
// plan keys: their fields (e.g. the server-populated object_id) have no source
// location, so resolving them would fail the whole sync. Permissions are written
// under their parent resource instead (see TestExtractChangesDeclaredPermissions),
// and grants are skipped.
func TestExtractChangesSkipsPermissionsAndGrants(t *testing.T) {
	ctx := t.Context()

	plan := &deployplan.Plan{Plan: map[string]*deployplan.PlanEntry{
		"resources.jobs.my_job": {
			Changes: deployplan.Changes{
				"description": {Action: deployplan.Update, New: nil, Remote: "remote-desc"},
			},
		},
		"resources.jobs.my_job.permissions": {
			Changes: deployplan.Changes{
				"object_id": {Action: deployplan.Update, New: nil, Remote: "/jobs/123"},
			},
		},
		"resources.schemas.my_schema.grants": {
			Changes: deployplan.Changes{
				"[0].privileges": {Action: deployplan.Update, New: nil, Remote: []any{"SELECT"}},
			},
		},
		// A resource literally named "permissions" is a regular resource, not a
		// permissions sub-resource, so its changes must be kept (structural
		// classification via config.GetNodeAndType, not a suffix match).
		"resources.jobs.permissions": {
			Changes: deployplan.Changes{
				"description": {Action: deployplan.Update, New: nil, Remote: "kept"},
			},
		},
	}}

	changes, _, err := ExtractChanges(ctx, &bundle.Bundle{}, plan)
	require.NoError(t, err)

	_, hasPerms := changes["resources.jobs.my_job.permissions"]
	assert.False(t, hasPerms, "permissions sub-resource must be skipped")
	_, hasGrants := changes["resources.schemas.my_schema.grants"]
	assert.False(t, hasGrants, "grants sub-resource must be skipped")

	job, hasJob := changes["resources.jobs.my_job"]
	require.True(t, hasJob, "regular resource changes must be kept")
	assert.Contains(t, job, "description")

	namedPerms, hasNamedPerms := changes["resources.jobs.permissions"]
	require.True(t, hasNamedPerms, "a resource named permissions must not be skipped")
	assert.Contains(t, namedPerms, "description")
}

func TestExtractChangesDeclaredPermissions(t *testing.T) {
	value, err := yamlloader.LoadYAML("databricks.yml", strings.NewReader(`
resources:
  jobs:
    my_job:
      permissions:
        - level: CAN_VIEW
          user_name: viewer@example.com
        - level: CAN_VIEW
          user_name: gone@example.com
`))
	require.NoError(t, err)
	// FixPermissions adds the current user's entry, with no location.
	value, err = dyn.Set(value, "resources.jobs.my_job.permissions", dyn.V(append(
		value.Get("resources").Get("jobs").Get("my_job").Get("permissions").MustSequence(),
		dyn.V(map[string]dyn.Value{"level": dyn.V("IS_OWNER"), "user_name": dyn.V("me@example.com")}),
	)))
	require.NoError(t, err)
	b := &bundle.Bundle{}
	require.NoError(t, b.Config.Mutate(func(dyn.Value) (dyn.Value, error) { return value, nil }))

	plan := &deployplan.Plan{Plan: map[string]*deployplan.PlanEntry{
		"resources.jobs.my_job": {},
		"resources.jobs.my_job.permissions": {
			Action:      deployplan.Update,
			RemoteState: &dresources.PermissionsState{ObjectID: "/jobs/123"},
			Changes: deployplan.Changes{
				"[user_name='viewer@example.com'].level": {
					Action: deployplan.Update, Old: "CAN_VIEW", New: "CAN_VIEW", Remote: "CAN_MANAGE",
				},
				"[group_name='data-team']": {
					Action: deployplan.Update,
					Remote: dresources.StatePermission{Level: iam.PermissionLevelCanView, GroupName: "data-team"},
				},
				"[user_name='gone@example.com']": {
					Action: deployplan.Update,
					Old:    dresources.StatePermission{Level: iam.PermissionLevelCanView, UserName: "gone@example.com"},
					New:    dresources.StatePermission{Level: iam.PermissionLevelCanView, UserName: "gone@example.com"},
				},
				"[user_name='me@example.com'].level": {
					Action: deployplan.Update, Old: "IS_OWNER", New: "IS_OWNER", Remote: "CAN_MANAGE",
				},
				"object_id": {Action: deployplan.Update, Remote: "/jobs/456"},
			},
		},
	}}

	changes, _, err := ExtractChanges(t.Context(), b, plan)
	require.NoError(t, err)
	assert.Equal(t, Changes{
		"resources.jobs.my_job": {
			"permissions[user_name='viewer@example.com'].level": {Operation: OperationReplace, Value: "CAN_MANAGE"},
			"permissions[group_name='data-team']": {
				Operation: OperationAdd,
				Value:     map[string]any{"level": "CAN_VIEW", "group_name": "data-team"},
			},
			"permissions[user_name='gone@example.com']": {Operation: OperationRemove},
		},
	}, changes)
}

func TestWithoutCurrentUserDefaults(t *testing.T) {
	permissions := []dresources.StatePermission{
		{Level: iam.PermissionLevelIsOwner, UserName: "me@example.com"},
		{Level: iam.PermissionLevelCanManage, ServicePrincipalName: "me@example.com"},
		{Level: iam.PermissionLevelIsOwner, UserName: "owner@example.com"},
		{Level: iam.PermissionLevelCanManage, UserName: "other@example.com"},
		{Level: iam.PermissionLevelCanView, GroupName: "users"},
	}

	assert.Equal(t, []dresources.StatePermission{
		{Level: iam.PermissionLevelIsOwner, UserName: "owner@example.com"},
		{Level: iam.PermissionLevelCanManage, UserName: "other@example.com"},
		{Level: iam.PermissionLevelCanView, GroupName: "users"},
	}, withoutCurrentUserDefaults(permissions, "me@example.com", "jobs"))
}

func TestWithoutCurrentUserDefaultsClusterPolicy(t *testing.T) {
	permissions := []dresources.StatePermission{
		{Level: iam.PermissionLevelCanUse, ServicePrincipalName: "me@example.com"},
		{Level: iam.PermissionLevelCanUse, UserName: "other@example.com"},
		{Level: iam.PermissionLevelCanUse, GroupName: "users"},
	}

	assert.Equal(t, []dresources.StatePermission{
		{Level: iam.PermissionLevelCanUse, UserName: "other@example.com"},
		{Level: iam.PermissionLevelCanUse, GroupName: "users"},
	}, withoutCurrentUserDefaults(permissions, "me@example.com", "cluster_policies"))

	creatorOnly := []dresources.StatePermission{
		{Level: iam.PermissionLevelCanUse, UserName: "me@example.com"},
	}
	assert.Empty(t, withoutCurrentUserDefaults(creatorOnly, "me@example.com", "cluster_policies"))
}

func TestPermissionsReadSkip(t *testing.T) {
	denied := &apierr.APIError{StatusCode: 403, ErrorCode: "PERMISSION_DENIED", Message: "no access"}
	assert.Equal(t, SkippedChange{
		ResourceKey: "resources.jobs.my_job",
		Reason:      SkipReasonPermissionsPermissionDenied,
		Error:       "no access",
	}, permissionsReadSkip("resources.jobs.my_job", denied))

	assert.Equal(t, SkippedChange{
		ResourceKey: "resources.jobs.my_job",
		Reason:      SkipReasonPermissionsReadFailed,
		Error:       "unexpected object_id",
	}, permissionsReadSkip("resources.jobs.my_job", errors.New("unexpected object_id")))
}

func TestUnreadPermissionsAdapters(t *testing.T) {
	adapters, err := dresources.InitAll(nil)
	require.NoError(t, err)
	b := &bundle.Bundle{}
	b.DeploymentBundle.Adapters = adapters

	remoteRunAs := &deployplan.ChangeDesc{Remote: &jobs.JobRunAs{UserName: "other@example.com"}}
	plan := &deployplan.Plan{Plan: map[string]*deployplan.PlanEntry{
		"resources.jobs.unmanaged":                {Action: deployplan.Skip},
		"resources.jobs.remote_run_as":            {Action: deployplan.Skip, RemoteState: &dresources.JobRemote{}, Changes: deployplan.Changes{"run_as": remoteRunAs}},
		"resources.jobs.declared":                 {Action: deployplan.Skip},
		"resources.jobs.declared.permissions":     {Action: deployplan.Skip},
		"resources.jobs.removed":                  {Action: deployplan.Update},
		"resources.jobs.removed.permissions":      {Action: deployplan.Delete},
		"resources.jobs.undeployed":               {Action: deployplan.Skip},
		"resources.jobs.undeployed.permissions":   {Action: deployplan.Create},
		"resources.jobs.not_deployed":             {Action: deployplan.Create},
		"resources.jobs.not_deployed.permissions": {Action: deployplan.Create},
		"resources.pipelines.unmanaged":           {Action: deployplan.Skip},
		"resources.secret_scopes.scope":           {Action: deployplan.Skip},
		"resources.schemas.schema":                {Action: deployplan.Skip},
		"resources.schemas.schema.grants":         {Action: deployplan.Skip},
	}}

	assert.ElementsMatch(t,
		[]string{"resources.jobs.remote_run_as", "resources.jobs.undeployed"},
		slices.Collect(maps.Keys(unreadPermissionsAdapters(b, plan))),
	)
}

func TestAddUndeployedPermissionsChanges(t *testing.T) {
	value, err := yamlloader.LoadYAML("databricks.yml", strings.NewReader(`
resources:
  jobs:
    my_job:
      permissions:
        - level: CAN_MANAGE
          user_name: viewer@example.com
        - level: CAN_VIEW
          user_name: gone@example.com
        - level: CAN_VIEW
          group_name: users
`))
	require.NoError(t, err)
	// FixPermissions adds the current user's entry, with no location.
	value, err = dyn.Set(value, "resources.jobs.my_job.permissions", dyn.V(append(
		value.Get("resources").Get("jobs").Get("my_job").Get("permissions").MustSequence(),
		dyn.V(map[string]dyn.Value{"level": dyn.V("IS_OWNER"), "user_name": dyn.V("me@example.com")}),
	)))
	require.NoError(t, err)
	adapters, err := dresources.InitAll(nil)
	require.NoError(t, err)
	b := &bundle.Bundle{}
	require.NoError(t, b.Config.Mutate(func(dyn.Value) (dyn.Value, error) { return value, nil }))
	b.DeploymentBundle.Adapters = adapters
	b.DeploymentBundle.StateCache.Store("resources.jobs.my_job.permissions", &structvar.StructVar{
		Value: &dresources.PermissionsState{EmbeddedSlice: []dresources.StatePermission{
			{Level: iam.PermissionLevelCanManage, UserName: "viewer@example.com"},
			{Level: iam.PermissionLevelCanView, UserName: "gone@example.com"},
			{Level: iam.PermissionLevelCanView, GroupName: "users"},
			{Level: iam.PermissionLevelIsOwner, UserName: "me@example.com"},
		}},
	})

	changes := make(Changes)
	err = addUndeployedPermissionsChanges(changes, b, "resources.jobs.my_job", []dresources.StatePermission{
		{Level: iam.PermissionLevelCanManage, UserName: "me@example.com"},
		{Level: iam.PermissionLevelCanView, UserName: "viewer@example.com"},
		{Level: iam.PermissionLevelCanView, GroupName: "users"},
		{Level: iam.PermissionLevelCanManageRun, UserName: "late@example.com"},
	})
	require.NoError(t, err)
	assert.Equal(t, Changes{
		"resources.jobs.my_job": {
			"permissions[user_name='viewer@example.com'].level": {Operation: OperationReplace, Value: "CAN_VIEW", LocalEdit: true},
			"permissions[user_name='gone@example.com']":         {Operation: OperationRemove, LocalEdit: true},
			"permissions[user_name='late@example.com']": {
				Operation: OperationAdd,
				Value:     map[string]any{"level": "CAN_MANAGE_RUN", "user_name": "late@example.com"},
			},
		},
	}, changes)
}

func TestLookupDeployedFieldReadsRemoteState(t *testing.T) {
	b := &bundle.Bundle{}
	b.DeploymentBundle.RemoteStateCache.Store("resources.model_serving_endpoints.my_endpoint",
		&dresources.ModelServingEndpointRemote{EndpointId: "abc123"})

	value, err := lookupDeployedField(b, "resources.model_serving_endpoints.my_endpoint", "endpoint_id")
	require.NoError(t, err)
	assert.Equal(t, "abc123", value)
}

func TestExtractChangesEmptyRemotePermissions(t *testing.T) {
	value, err := yamlloader.LoadYAML("databricks.yml", strings.NewReader(`
resources:
  apps:
    my_app:
      permissions:
        - level: CAN_USE
          group_name: users
`))
	require.NoError(t, err)
	b := &bundle.Bundle{}
	require.NoError(t, b.Config.Mutate(func(dyn.Value) (dyn.Value, error) { return value, nil }))

	declared := []dresources.StatePermission{{Level: "CAN_USE", GroupName: "users"}}
	plan := &deployplan.Plan{Plan: map[string]*deployplan.PlanEntry{
		"resources.apps.my_app": {},
		"resources.apps.my_app.permissions": {
			Action:      deployplan.Update,
			RemoteState: &dresources.PermissionsState{ObjectID: "/apps/my_app"},
			Changes: deployplan.Changes{
				"": {Action: deployplan.Update, Old: declared, New: declared},
			},
		},
	}}

	changes, _, err := ExtractChanges(t.Context(), b, plan)
	require.NoError(t, err)
	assert.Equal(t, Changes{
		"resources.apps.my_app": {
			"permissions[group_name='users']": {Operation: OperationRemove},
		},
	}, changes)
}

func TestExtractChangesSkipsLifecycleFields(t *testing.T) {
	plan := &deployplan.Plan{Plan: map[string]*deployplan.PlanEntry{
		"resources.genie_spaces.my_space": {
			Changes: deployplan.Changes{
				// OverrideChangeDesc promotes etag drift to an update for deploy.
				"etag":  {Action: deployplan.Update, Old: "1", Remote: "2"},
				"title": {Action: deployplan.Update, Old: "a", New: "a", Remote: "b"},
			},
		},
		"resources.pipelines.my_pipeline": {
			Changes: deployplan.Changes{
				// Not one of the backend default's allowed values, so it is a real edit.
				"configuration['spark.sql.ansi.enabled']": {Action: deployplan.Update, Old: "true", Remote: "false"},
			},
		},
		"resources.pipelines.other_pipeline": {
			Changes: deployplan.Changes{
				// A remote-only map holding a backend default next to a real addition.
				"configuration": {Action: deployplan.Update, Remote: map[string]string{
					"spark.sql.ansi.enabled": "true",
					"custom.key":             "value",
				}},
			},
		},
		"resources.model_serving_endpoints.my_endpoint": {
			Changes: deployplan.Changes{
				// GET returns no telemetry_config, so the plan reports the whole
				// object, and table_names under it is input_only.
				"telemetry_config": {
					Action: deployplan.Update,
					Old:    &serving.TelemetryConfig{TableNames: &serving.UnityCatalogTableNames{LogsTable: "main.default.logs"}},
					New:    &serving.TelemetryConfig{TableNames: &serving.UnityCatalogTableNames{LogsTable: "main.default.logs"}},
				},
			},
		},
		"resources.jobs.my_job": {
			Changes: deployplan.Changes{
				// A task added remotely keeps its fields, including node_type_id, minus the
				// backend-managed ones and defaults pinned to a value.
				"tasks[task_key='loop']": {Action: deployplan.Update, Remote: map[string]any{
					"task_key": "loop",
					"for_each_task": map[string]any{"task": map[string]any{"new_cluster": map[string]any{
						"spark_version":  "16.4.x-scala2.12",
						"node_type_id":   "i3.xlarge",
						"aws_attributes": map[string]any{"availability": "SPOT"},
					}}},
					"run_if": "ALL_SUCCESS",
				}},
				// input_only: never returned by GET, so a local edit would be removed.
				"trigger.table_update.condition": {Action: deployplan.Update, Old: "ANY_UPDATED", New: "ALL_UPDATED"},
				// Backend default the config no longer sets.
				"format": {Action: deployplan.Update, Old: "MULTI_TASK", Remote: "MULTI_TASK"},
				// A principal kind switch must still be synced.
				"run_as.service_principal_name": {Action: deployplan.Skip, Reason: deployplan.ReasonBackendDefault, Remote: "sp"},
			},
		},
	}}

	changes, _, err := ExtractChanges(t.Context(), &bundle.Bundle{}, plan)
	require.NoError(t, err)

	assert.Equal(t, ResourceChanges{
		"title": {Operation: OperationReplace, Value: "b"},
	}, changes["resources.genie_spaces.my_space"])
	assert.Equal(t, ResourceChanges{
		"configuration['spark.sql.ansi.enabled']": {Operation: OperationAdd, Value: "false", LocalEdit: true},
	}, changes["resources.pipelines.my_pipeline"])
	assert.Equal(t, ResourceChanges{
		"configuration": {Operation: OperationAdd, Value: map[string]any{"custom.key": "value"}},
	}, changes["resources.pipelines.other_pipeline"])
	assert.NotContains(t, changes, "resources.model_serving_endpoints.my_endpoint")
	assert.Equal(t, ResourceChanges{
		"tasks[task_key='loop']": {Operation: OperationAdd, Value: map[string]any{
			"task_key": "loop",
			"for_each_task": map[string]any{"task": map[string]any{"new_cluster": map[string]any{
				"spark_version": "16.4.x-scala2.12",
				"node_type_id":  "i3.xlarge",
			}}},
		}},
		"run_as.service_principal_name": {Operation: OperationAdd, Value: "sp"},
	}, changes["resources.jobs.my_job"])
}
