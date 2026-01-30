package configsync

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/engine"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/direct"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go/marshal"
)

type OperationType string

const (
	OperationUnknown OperationType = "unknown"
	OperationAdd     OperationType = "add"
	OperationRemove  OperationType = "remove"
	OperationReplace OperationType = "replace"
	OperationSkip    OperationType = "skip"
)

type ConfigChangeDesc struct {
	Operation OperationType `json:"operation"`
	Value     any           `json:"value,omitempty"` // Normalized remote value (nil for remove operations)
}

type ResourceChanges map[string]*ConfigChangeDesc

type Changes map[string]ResourceChanges

// normalizeValue converts values to plain Go types suitable for YAML patching
// by using SDK marshaling which properly handles ForceSendFields and other annotations.
func normalizeValue(v any) (any, error) {
	if v == nil {
		return nil, nil
	}

	switch v.(type) {
	case bool, string, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		return v, nil
	}

	rv := reflect.ValueOf(v)
	rt := rv.Type()

	if rt.Kind() == reflect.Ptr {
		rt = rt.Elem()
	}

	var data []byte
	var err error

	if rt.Kind() == reflect.Struct {
		data, err = marshal.Marshal(v)
	} else {
		data, err = json.Marshal(v)
	}

	if err != nil {
		return v, fmt.Errorf("failed to marshal value of type %T: %w", v, err)
	}

	var normalized any
	err = json.Unmarshal(data, &normalized)
	if err != nil {
		return v, fmt.Errorf("failed to unmarshal value: %w", err)
	}

	return normalized, nil
}

func convertChangeDesc(path string, cd *deployplan.ChangeDesc) (*ConfigChangeDesc, error) {
	hasConfigValue := cd.Old != nil || cd.New != nil

	op := OperationUnknown
	if shouldSkipField(path, cd) {
		return &ConfigChangeDesc{
			Operation: OperationSkip,
		}, nil
	}

	if cd.Remote == nil && hasConfigValue {
		op = OperationRemove
	}
	if cd.Remote != nil && hasConfigValue {
		op = OperationReplace
	}
	if cd.Remote != nil && !hasConfigValue {
		op = OperationAdd
	}

	var normalizedValue any
	var err error
	if op != OperationRemove {
		normalizedValue, err = normalizeValue(cd.Remote)
		if err != nil {
			return nil, fmt.Errorf("failed to normalize remote value: %w", err)
		}
	}

	return &ConfigChangeDesc{
		Operation: op,
		Value:     normalizedValue,
	}, nil
}

// DetectChanges compares current remote state with the last deployed state
// and returns a map of resource changes.
func DetectChanges(ctx context.Context, b *bundle.Bundle, engine engine.EngineType) (Changes, error) {
	changes := make(Changes)

	deployBundle := &direct.DeploymentBundle{}
	var statePath string
	if engine.IsDirect() {
		_, statePath = b.StateFilenameDirect(ctx)
	} else {
		_, statePath = b.StateFilenameConfigSnapshot(ctx)
	}

	plan, err := deployBundle.CalculatePlan(ctx, b.WorkspaceClient(), &b.Config, statePath)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate plan: %w", err)
	}

	for resourceKey, entry := range plan.Plan {
		resourceChanges := make(ResourceChanges)

		if entry.Changes != nil {
			for path, changeDesc := range entry.Changes {
				change, err := convertChangeDesc(path, changeDesc)
				if err != nil {
					return nil, fmt.Errorf("failed to compute config change for path %s: %w", path, err)
				}
				if change.Operation == OperationSkip {
					continue
				}
				resourceChanges[path] = change
			}
		}

		if len(resourceChanges) != 0 {
			changes[resourceKey] = resourceChanges
		}

		log.Debugf(ctx, "Resource %s has %d changes", resourceKey, len(resourceChanges))
	}

	return changes, nil
}

func matchParts(patternParts, pathParts []string) bool {
	if len(patternParts) == 0 && len(pathParts) == 0 {
		return true
	}
	if len(patternParts) == 0 || len(pathParts) == 0 {
		return false
	}

	patternPart := patternParts[0]
	pathPart := pathParts[0]

	if patternPart == "*" {
		return matchParts(patternParts[1:], pathParts[1:])
	}

	if strings.Contains(patternPart, "[*]") {
		prefix := strings.Split(patternPart, "[*]")[0]

		if strings.HasPrefix(pathPart, prefix) && strings.Contains(pathPart, "[") {
			return matchParts(patternParts[1:], pathParts[1:])
		}
		return false
	}

	if patternPart == pathPart {
		return matchParts(patternParts[1:], pathParts[1:])
	}

	return false
}

func matchPattern(pattern, path string) bool {
	patternParts := strings.Split(pattern, ".")
	pathParts := strings.Split(path, ".")
	return matchParts(patternParts, pathParts)
}

// serverSideDefaults contains all hardcoded server-side defaults.
// This is a temporary solution until the bundle plan issue is resolved.
var serverSideDefaults = map[string]func(*deployplan.ChangeDesc) bool{
	// Job-level fields
	"timeout_seconds": isZero,
	"usage_policy_id": alwaysDefault, // computed field
	"edit_mode":       alwaysDefault, // set by CLI

	// Task-level fields
	"tasks[*].run_if":               isStringEqual("ALL_SUCCESS"),
	"tasks[*].disabled":             isBoolEqual(false),
	"tasks[*].timeout_seconds":      isZero,
	"tasks[*].notebook_task.source": isStringEqual("WORKSPACE"),

	// Cluster fields (tasks)
	"tasks[*].new_cluster.aws_attributes":     alwaysDefault,
	"tasks[*].new_cluster.azure_attributes":   alwaysDefault,
	"tasks[*].new_cluster.gcp_attributes":     alwaysDefault,
	"tasks[*].new_cluster.data_security_mode": isStringEqual("SINGLE_USER"), // TODO this field is computed on some workspaces in integration tests, check why and if we can skip it

	"tasks[*].new_cluster.enable_elastic_disk": alwaysDefault, // deprecated field

	// Cluster fields (job_clusters)
	"job_clusters[*].new_cluster.aws_attributes":     alwaysDefault,
	"job_clusters[*].new_cluster.azure_attributes":   alwaysDefault,
	"job_clusters[*].new_cluster.gcp_attributes":     alwaysDefault,
	"job_clusters[*].new_cluster.data_security_mode": isStringEqual("SINGLE_USER"), // TODO this field is computed on some workspaces in integration tests, check why and if we can skip it

	"job_clusters[*].new_cluster.enable_elastic_disk": alwaysDefault, // deprecated field

	// Terraform defaults
	"run_as": alwaysDefault,

	// Pipeline fields
	"storage": defaultIfNotSpecified, // TODO it is computed if not specified, probably we should not skip it
}

// shouldSkipField checks if a given field path should be skipped as a hardcoded server-side default.
func shouldSkipField(path string, changeDesc *deployplan.ChangeDesc) bool {
	// TODO: as for now in bundle plan all remote-side changes are considered as server-side defaults.
	// Once it is solved - stop skipping server-side defaults in these checks and remove hardcoded default.
	if changeDesc.Action == deployplan.Skip && changeDesc.Reason != deployplan.ReasonServerSideDefault {
		return true
	}

	for pattern, isDefault := range serverSideDefaults {
		if matchPattern(pattern, path) {
			return isDefault(changeDesc)
		}
	}
	return false
}

// alwaysDefault always returns true (for computed fields).
func alwaysDefault(*deployplan.ChangeDesc) bool {
	return true
}

func defaultIfNotSpecified(changeDesc *deployplan.ChangeDesc) bool {
	if changeDesc.Old == nil && changeDesc.New == nil {
		return true
	}
	return false
}

// isStringEqual returns a function that checks if the remote value equals the given string.
func isStringEqual(expected string) func(*deployplan.ChangeDesc) bool {
	return func(changeDesc *deployplan.ChangeDesc) bool {
		if changeDesc.Remote == nil {
			return expected == ""
		}
		// Convert to string to handle SDK enum types
		actual := fmt.Sprintf("%v", changeDesc.Remote)
		return actual == expected
	}
}

// isBoolEqual returns a function that checks if the remote value equals the given bool.
func isBoolEqual(expected bool) func(*deployplan.ChangeDesc) bool {
	return func(changeDesc *deployplan.ChangeDesc) bool {
		if actual, ok := changeDesc.Remote.(bool); ok {
			return actual == expected
		}
		return false
	}
}

// isZero checks if the remote value is zero (0 or 0.0).
func isZero(changeDesc *deployplan.ChangeDesc) bool {
	if changeDesc.Remote == nil {
		return true
	}

	switch v := changeDesc.Remote.(type) {
	case int:
		return v == 0
	case int64:
		return v == 0
	case float64:
		return v == 0.0
	default:
		return false
	}
}
