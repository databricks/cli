package configsync

import (
	"context"
	"fmt"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/structs/structdiff"
	"github.com/databricks/cli/libs/structs/structvar"
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
	// Reason says why a skipped change was left unapplied.
	Reason string `json:"reason,omitempty"`

	// LocalEdit reports that the local config value for this field differs from
	// the last-deployed state, so applying this change overwrites a not-yet-
	// deployed local edit. Telemetry-only; direct engine only (the terraform
	// sync snapshot has no per-field base). Not part of the command output.
	LocalEdit bool `json:"-"`
}

type ResourceChanges map[string]*ConfigChangeDesc

type Changes map[string]ResourceChanges

func normalizeValue(v any) any {
	return structvar.NewView(v, nil, nil).AsAny()
}

func filterEntityDefaults(basePath string, value any) any {
	if value == nil {
		return nil
	}

	if arr, ok := value.([]any); ok {
		result := make([]any, 0, len(arr))
		for i, elem := range arr {
			elementPath := fmt.Sprintf("%s[%d]", basePath, i)
			result = append(result, filterEntityDefaults(elementPath, elem))
		}
		return result
	}

	m, ok := value.(map[string]any)
	if !ok {
		return value
	}

	result := make(map[string]any)
	for key, val := range m {
		fieldPath := basePath + "." + key

		if shouldSkipField(fieldPath, val, false) {
			continue
		}

		if nestedMap, ok := val.(map[string]any); ok {
			result[key] = filterEntityDefaults(fieldPath, nestedMap)
		} else {
			result[key] = val
		}
	}

	return result
}

func convertChangeDesc(path string, cd *deployplan.ChangeDesc) (*ConfigChangeDesc, error) {
	// Use cd.New (current config) to decide whether the field exists "on the config side".
	// cd.Old (saved state) must not be considered: when the user has already synced a rename
	// locally (cd.New == nil for the old key) but state still holds the prior key, including
	// cd.Old in this check would classify the change as Replace and fail later in
	// resolveSelectors because the old key no longer exists in the YAML.
	hasConfigValue := cd.New != nil
	normalizedValue := normalizeValue(cd.Remote)

	if shouldSkipField(path, normalizedValue, hasConfigValue) {
		return &ConfigChangeDesc{
			Operation: OperationSkip,
		}, nil
	}

	normalizedValue = filterEntityDefaults(path, normalizedValue)
	normalizedValue = resetValueIfNeeded(path, normalizedValue)

	var op OperationType
	if normalizedValue == nil && hasConfigValue {
		op = OperationRemove
	} else if normalizedValue != nil && hasConfigValue {
		op = OperationReplace
	} else if normalizedValue != nil && !hasConfigValue {
		op = OperationAdd
	} else {
		op = OperationSkip
	}

	return &ConfigChangeDesc{
		Operation: op,
		Value:     normalizedValue,
	}, nil
}

// isPermissionsOrGrantsSubResource reports whether a plan resource key is a
// permissions or grants sub-resource ("resources.<type>.<name>.permissions" /
// ".grants"). It classifies the key structurally via config.GetNodeAndType,
// which keys on the path component (index 3), so a resource literally named
// "permissions" ("resources.jobs.permissions") is not misclassified.
func isPermissionsOrGrantsSubResource(resourceKey string) bool {
	nodeType := config.GetResourceTypeFromKey(resourceKey)
	return strings.HasSuffix(nodeType, ".permissions") || strings.HasSuffix(nodeType, ".grants")
}

// ExtractChanges extracts the map of remote-vs-config changes from a deploy
// plan, and the resources whose remote permissions could not be read.
func ExtractChanges(ctx context.Context, b *bundle.Bundle, plan *deployplan.Plan) (Changes, []SkippedChange, error) {
	changes := make(Changes)
	// Read first: a job's ACL names the owner that skipJobRunAs compares run_as to.
	remote, skipped := readRemotePermissions(ctx, b, plan)
	owners := jobOwners(ctx, plan, remote)

	for resourceKey, entry := range plan.Plan {
		// permissions and grants are emitted as their own plan keys
		// ("resources.<type>.<name>.permissions" / ".grants"; see splitResourcePath
		// in bundle/direct/bundle_plan.go). Their fields (e.g. object_id) are
		// server-populated with no source location, so they cannot be written back
		// as is; permissions are added below under their parent resource instead,
		// and grants are skipped.
		if isPermissionsOrGrantsSubResource(resourceKey) {
			continue
		}

		resourceChanges := make(ResourceChanges)

		if entry.Changes != nil {
			for path, changeDesc := range entry.Changes {
				change, err := convertPlanChange(resourceKey, path, changeDesc, owners[resourceKey], b.Config.Presets.NamePrefix)
				if err != nil {
					return nil, nil, fmt.Errorf("failed to compute config change for path %s: %w", path, err)
				}
				if change != nil {
					resourceChanges[path] = change
				}
			}
		}

		if len(resourceChanges) != 0 {
			changes[resourceKey] = resourceChanges
		}

		log.Debugf(ctx, "Resource %s has %d changes", resourceKey, len(resourceChanges))
	}

	if err := addPermissionsChanges(changes, b, plan, remote); err != nil {
		return nil, nil, err
	}

	return changes, skipped, nil
}

// convertPlanChange converts a plan change at path in resourceKey, returning nil when there is
// nothing to write back. owner is the principal the resource runs as by default, if it is a job.
func convertPlanChange(resourceKey, path string, changeDesc *deployplan.ChangeDesc, owner, namePrefix string) (*ConfigChangeDesc, error) {
	fullPath := resourceKey + "." + path
	if changeDesc.Action == deployplan.Skip && changeDesc.Reason != deployplan.ReasonRemoteAddition &&
		!keepSkippedRunAsChange(fullPath, changeDesc) {
		return nil, nil
	}
	// Runs before applyLifecycle, which lets every whole run_as through.
	if skipJobRunAs(fullPath, changeDesc, owner) {
		return nil, nil
	}
	synced, err := applyLifecycle(resourceKey, path, changeDesc)
	if err != nil {
		return nil, fmt.Errorf("failed to apply lifecycle rules: %w", err)
	}
	if synced == nil {
		return nil, nil
	}

	change, err := convertChangeDesc(fullPath, synced)
	if err != nil {
		return nil, err
	}
	if change.Operation == OperationSkip {
		return nil, nil
	}
	// The state snapshot holds real per-field values, so New != Old
	// means the local config diverged from the last deploy and this
	// change overwrites that pending edit.
	if !structdiff.IsEqual(changeDesc.Old, changeDesc.New) {
		change.LocalEdit = true
	}
	change.Value = stripNamePrefix(fullPath, change.Value, namePrefix)
	return change, nil
}
