package dresources

import (
	"reflect"
	"slices"
	"testing"

	"github.com/databricks/cli/libs/structs/structaccess"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/structs/structwalk"
	"github.com/stretchr/testify/require"
)

// knownMissingInRemoteType lists fields that exist in StateType but not in RemoteType.
// These are known issues that should be fixed. If a field listed here is found in RemoteType,
// the test fails to ensure the entry is removed from this map.
var knownMissingInRemoteType = map[string][]string{
	"clusters": {
		"apply_policy_default_values",
	},
	"jobs": {
		"budget_policy_id",
		"continuous",
		"deployment",
		"description",
		"edit_mode",
		"email_notifications",
		"environments",
		"format",
		"git_source",
		"health",
		"job_clusters",
		"max_concurrent_runs",
		"name",
		"notification_settings",
		"parameters",
		"performance_target",
		"queue",
		"run_as",
		"schedule",
		"tags",
		"tasks",
		"timeout_seconds",
		"trigger",
		"usage_policy_id",
		"webhook_notifications",
	},
	"model_serving_endpoints": {
		"ai_gateway",
		"budget_policy_id",
		"config",
		"description",
		"email_notifications",
		"name",
		"rate_limits",
		"route_optimized",
		"tags",
	},
	"pipelines": {
		"allow_duplicate_names",
		"budget_policy_id",
		"catalog",
		"channel",
		"clusters",
		"configuration",
		"continuous",
		"deployment",
		"development",
		"dry_run",
		"edition",
		"environment",
		"event_log",
		"filters",
		"gateway_definition",
		"id",
		"ingestion_definition",
		"libraries",
		"notifications",
		"photon",
		"restart_window",
		"root_path",
		"schema",
		"serverless",
		"storage",
		"tags",
		"target",
		"trigger",
		"usage_policy_id",
	},
	"quality_monitors": {
		"skip_builtin_dashboard",
		"warehouse_id",
	},
	"secret_scopes": {
		"backend_azure_keyvault",
		"initial_manage_principal",
		"scope",
		"scope_backend_type",
	},
}

// knownMissingInStateType lists fields that exist in InputType but not in StateType.
// These are known issues that should be fixed. If a field listed here is found in StateType,
// the test fails to ensure the entry is removed from this map.
var knownMissingInStateType = map[string][]string{
	"alerts": {
		"file_path",
		"lifecycle",
		"modified_status",
		"permissions",
		"url",
	},
	"apps": {
		"config",
		"lifecycle",
		"modified_status",
		"permissions",
		"source_code_path",
	},
	"catalogs": {
		"grants",
		"id",
		"lifecycle",
		"modified_status",
		"url",
	},
	"clusters": {
		"id",
		"lifecycle",
		"modified_status",
		"permissions",
		"url",
	},
	"dashboards": {
		"file_path",
		"id",
		"lifecycle",
		"modified_status",
		"permissions",
		"url",
	},
	"database_catalogs": {
		"id",
		"lifecycle",
		"modified_status",
		"url",
	},
	"database_instances": {
		"id",
		"lifecycle",
		"modified_status",
		"permissions",
		"url",
	},
	"experiments": {
		"id",
		"lifecycle",
		"modified_status",
		"permissions",
		"url",
	},
	"jobs": {
		"id",
		"lifecycle",
		"modified_status",
		"permissions",
		"url",
	},
	"model_serving_endpoints": {
		"id",
		"lifecycle",
		"modified_status",
		"permissions",
		"url",
	},
	"models": {
		"id",
		"lifecycle",
		"modified_status",
		"permissions",
		"url",
	},
	"pipelines": {
		"lifecycle",
		"modified_status",
		"permissions",
		"url",
	},
	"quality_monitors": {
		"id",
		"lifecycle",
		"modified_status",
		"url",
	},
	"registered_models": {
		"grants",
		"id",
		"lifecycle",
		"modified_status",
		"url",
	},
	"schemas": {
		"grants",
		"id",
		"lifecycle",
		"modified_status",
		"url",
	},
	"secret_scopes": {
		"backend_type",
		"id",
		"keyvault_metadata",
		"lifecycle",
		"modified_status",
		"name",
		"permissions",
		"url",
	},
	"sql_warehouses": {
		"id",
		"lifecycle",
		"modified_status",
		"permissions",
		"url",
	},
	"synced_database_tables": {
		"id",
		"lifecycle",
		"modified_status",
		"url",
	},
	"volumes": {
		"grants",
		"id",
		"lifecycle",
		"modified_status",
		"url",
	},
}

// TestInputSubset validates that all fields in InputType
// exist in StateType for each resource. StateType may have extra fields.
func TestInputSubset(t *testing.T) {
	for resourceType, resource := range SupportedResources {
		adapter, err := NewAdapter(resource, resourceType, nil)
		require.NoError(t, err)

		t.Run(resourceType, func(t *testing.T) {
			inputType := adapter.InputConfigType()
			stateType := adapter.StateType()

			// Validate that all fields in InputType exist in StateType
			var missingFields []string
			err := structwalk.WalkType(inputType, func(path *structpath.PathNode, typ reflect.Type, field *reflect.StructField) bool {
				if path.IsRoot() {
					return true
				}
				if structaccess.Validate(stateType, path) != nil {
					missingFields = append(missingFields, path.String())
					return false // don't recurse into missing field
				}
				return true
			})
			require.NoError(t, err)

			known := knownMissingInStateType[resourceType]

			// Check that known missing fields are actually missing
			for _, f := range known {
				if !slices.Contains(missingFields, f) {
					t.Errorf("field %q is listed in knownMissingInStateType but exists in StateType; remove it from the list", f)
				}
			}

			// Filter out known missing fields
			var unexpectedMissing []string
			for _, f := range missingFields {
				if !slices.Contains(known, f) {
					unexpectedMissing = append(unexpectedMissing, f)
				}
			}

			if len(unexpectedMissing) > 0 {
				t.Errorf("fields in InputType not found in StateType: %v", unexpectedMissing)
			}
		})
	}
}

// TestRemoteSuperset validates that all fields in StateType
// exist in RemoteType for each resource. RemoteType may have extra fields.
func TestRemoteSuperset(t *testing.T) {
	for resourceType, resource := range SupportedResources {
		adapter, err := NewAdapter(resource, resourceType, nil)
		require.NoError(t, err)

		t.Run(resourceType, func(t *testing.T) {
			stateType := adapter.StateType()
			remoteType := adapter.RemoteType()

			// Validate that all fields in StateType exist in RemoteType
			var missingFields []string
			err := structwalk.WalkType(stateType, func(path *structpath.PathNode, typ reflect.Type, field *reflect.StructField) bool {
				if path.IsRoot() {
					return true
				}
				if structaccess.Validate(remoteType, path) != nil {
					missingFields = append(missingFields, path.String())
					return false // don't recurse into missing field
				}
				return true
			})
			require.NoError(t, err)

			known := knownMissingInRemoteType[resourceType]

			// Check that known missing fields are actually missing
			for _, f := range known {
				if !slices.Contains(missingFields, f) {
					t.Errorf("field %q is listed in knownMissingInRemoteType but exists in RemoteType; remove it from the list", f)
				}
			}

			// Filter out known missing fields
			var unexpectedMissing []string
			for _, f := range missingFields {
				if !slices.Contains(known, f) {
					unexpectedMissing = append(unexpectedMissing, f)
				}
			}

			if len(unexpectedMissing) > 0 {
				t.Errorf("fields in StateType not found in RemoteType: %v", unexpectedMissing)
			}
		})
	}
}
