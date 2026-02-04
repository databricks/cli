package dresources

import (
	"reflect"
	"slices"
	"strings"
	"testing"

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
			collectTypeIssues(t, stateType.Elem(), remoteType.Elem(), "", &missingFields)

			knownMissing := make(map[string]bool)
			for _, f := range knownMissingInRemoteType[resourceType] {
				knownMissing[f] = true
			}

			// Check that known missing fields are actually missing
			for _, f := range knownMissingInRemoteType[resourceType] {
				if !slices.Contains(missingFields, f) {
					t.Errorf("field %q is listed in knownMissingInRemoteType but exists in RemoteType; remove it from the list", f)
				}
			}

			// Filter out known missing fields
			var unexpectedMissing []string
			for _, f := range missingFields {
				if !knownMissing[f] {
					unexpectedMissing = append(unexpectedMissing, f)
				}
			}

			if len(unexpectedMissing) > 0 {
				t.Errorf("fields in StateType not found in RemoteType: %v", unexpectedMissing)
			}
		})
	}
}

func collectTypeIssues(t *testing.T, stateType, remoteType reflect.Type, prefix string, missingFields *[]string) {
	for i := range stateType.NumField() {
		stateField := stateType.Field(i)

		// Skip fields that are not serialized to JSON
		jsonTag := stateField.Tag.Get("json")
		if jsonTag == "-" {
			continue
		}

		// For embedded structs, check their fields against remoteType directly
		if stateField.Anonymous && stateField.Type.Kind() == reflect.Struct {
			collectTypeIssues(t, stateField.Type, remoteType, prefix, missingFields)
			continue
		}

		jsonName := strings.Split(jsonTag, ",")[0]
		if jsonName == "" {
			jsonName = stateField.Name
		}
		fieldPath := prefix + jsonName

		remoteField, found := remoteType.FieldByName(stateField.Name)
		if !found {
			*missingFields = append(*missingFields, fieldPath)
			continue
		}

		if stateField.Type == remoteField.Type {
			continue
		}

		// For nested structs, recurse
		if stateField.Type.Kind() == reflect.Struct && remoteField.Type.Kind() == reflect.Struct {
			collectTypeIssues(t, stateField.Type, remoteField.Type, fieldPath+".", missingFields)
		} else {
			t.Logf("type mismatch (not an error): %s: StateType has %s, RemoteType has %s",
				fieldPath, stateField.Type.String(), remoteField.Type.String())
		}
	}
}
