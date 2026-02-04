package dresources

import (
	"reflect"
	"slices"
	"testing"

	"github.com/databricks/cli/libs/structs/structaccess"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/structs/structtag"
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

// commonMissingInStateType lists fields that are commonly missing across all resource types.
// These are bundle-specific fields that exist in InputType but not in StateType.
var commonMissingInStateType = []string{
	"grants",
	"lifecycle",
	"permissions",
}

// knownMissingInStateType lists resource-specific fields that exist in InputType but not in StateType.
// Fields in commonMissingInStateType are automatically included for all types.
// Note: Fields with bundle:"internal" or bundle:"readonly" tags are automatically skipped.
var knownMissingInStateType = map[string][]string{
	"alerts": {
		"file_path",
	},
	"apps": {
		"config",
		"source_code_path",
	},
	"dashboards": {
		"file_path",
	},
	"secret_scopes": {
		"backend_type",
		"keyvault_metadata",
		"name",
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
				// Skip fields marked as internal or readonly in bundle tags
				if field != nil {
					btag := structtag.BundleTag(field.Tag.Get("bundle"))
					if btag.Internal() || btag.ReadOnly() {
						return false // don't recurse into internal/readonly fields
					}
				}
				if structaccess.Validate(stateType, path) != nil {
					missingFields = append(missingFields, path.String())
					return false // don't recurse into missing field
				}
				return true
			})
			require.NoError(t, err)

			known := append(commonMissingInStateType, knownMissingInStateType[resourceType]...)

			// Check that resource-specific known missing fields are actually missing
			for _, f := range knownMissingInStateType[resourceType] {
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
