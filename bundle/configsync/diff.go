package configsync

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/engine"
	"github.com/databricks/cli/bundle/deploy"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/direct"
	"github.com/databricks/cli/bundle/direct/dstate"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/structs/structpath"
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

func normalizeValue(v any) (any, error) {
	dynValue, err := convert.FromTyped(v, dyn.NilValue)
	if err != nil {
		return nil, fmt.Errorf("failed to convert value of type %T: %w", v, err)
	}

	return dynValue.AsAny(), nil
}

// filterEntityDefaults removes server-side default fields from an entity that
// is added remotely as a whole (e.g. a new task object). Each nested field is
// matched against the same lifecycle metadata as top-level changes. Maps that
// become empty after filtering are pruned to nil at the top level and in map
// fields, but kept as {} inside arrays: pruning an element would change the
// array arity. Element identity keys like task_key normally prevent full
// pruning there anyway.
func filterEntityDefaults(resourceType string, basePath *structpath.PathNode, value any) any {
	switch v := value.(type) {
	case map[string]any:
		filtered := filterEntityMap(resourceType, basePath, v)
		if len(filtered) == 0 {
			return nil
		}
		return filtered
	case []any:
		result := make([]any, 0, len(v))
		for i, elem := range v {
			elementPath := structpath.NewIndex(basePath, i)
			if m, ok := elem.(map[string]any); ok {
				result = append(result, filterEntityMap(resourceType, elementPath, m))
			} else {
				result = append(result, filterEntityDefaults(resourceType, elementPath, elem))
			}
		}
		return result
	default:
		return value
	}
}

// filterEntityMap drops map fields matched by the lifecycle metadata and prunes
// nested map fields that become empty after filtering. The result may itself be
// empty; pruning it is the caller's decision (see filterEntityDefaults).
func filterEntityMap(resourceType string, basePath *structpath.PathNode, m map[string]any) map[string]any {
	result := make(map[string]any)
	for key, val := range m {
		fieldPath := structpath.NewStringKey(basePath, key)

		if shouldSkipForSync(resourceType, fieldPath, val, false) && !isKeptForSync(resourceType, fieldPath) {
			continue
		}

		if nestedMap, ok := val.(map[string]any); ok {
			filtered := filterEntityMap(resourceType, fieldPath, nestedMap)
			if len(filtered) > 0 {
				result[key] = filtered
			}
			continue
		}
		result[key] = val
	}
	return result
}

func convertChangeDesc(resourceType string, path *structpath.PathNode, cd *deployplan.ChangeDesc) (*ConfigChangeDesc, error) {
	if isIgnoredForSync(resourceType, path) {
		return &ConfigChangeDesc{
			Operation: OperationSkip,
		}, nil
	}

	// Use cd.New (current config) to decide whether the field exists "on the config side".
	// cd.Old (saved state) must not be considered: when the user has already synced a rename
	// locally (cd.New == nil for the old key) but state still holds the prior key, including
	// cd.Old in this check would classify the change as Replace and fail later in
	// resolveSelectors because the old key no longer exists in the YAML.
	hasConfigValue := cd.New != nil
	normalizedValue, err := normalizeValue(cd.Remote)
	if err != nil {
		return nil, fmt.Errorf("failed to normalize remote value: %w", err)
	}

	if shouldSkipForSync(resourceType, path, normalizedValue, hasConfigValue) {
		return &ConfigChangeDesc{
			Operation: OperationSkip,
		}, nil
	}

	normalizedValue = filterEntityDefaults(resourceType, path, normalizedValue)
	normalizedValue = resetValueIfNeeded(resourceType, path, normalizedValue)

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

// DetectChanges compares current remote state with the last deployed state
// and returns a map of resource changes.
func DetectChanges(ctx context.Context, b *bundle.Bundle, engine engine.EngineType) (Changes, error) {
	changes := make(Changes)

	err := ensureSnapshotAvailable(ctx, b, engine)
	if err != nil {
		return nil, fmt.Errorf("state snapshot not available: %w", err)
	}

	var deployBundle *direct.DeploymentBundle
	if engine.IsDirect() {
		// For direct engine, state is already opened by the caller (process.go).
		deployBundle = &b.DeploymentBundle
	} else {
		deployBundle = &direct.DeploymentBundle{}
		_, statePath := b.StateFilenameConfigSnapshot(ctx)
		if err := deployBundle.StateDB.Open(ctx, statePath, dstate.WithRecovery(true), dstate.WithWrite(false)); err != nil {
			return nil, fmt.Errorf("failed to open state: %w", err)
		}
	}

	plan, err := deployBundle.CalculatePlan(ctx, b.WorkspaceClient(ctx), &b.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate plan: %w", err)
	}

	for resourceKey, entry := range plan.Plan {
		resourceChanges := make(ResourceChanges)
		resourceType := config.GetResourceTypeFromKey(resourceKey)

		if entry.Changes != nil {
			for path, changeDesc := range entry.Changes {
				if changeDesc.Action == deployplan.Skip {
					continue
				}

				fieldPath, err := structpath.ParsePath(path)
				if err != nil {
					return nil, fmt.Errorf("failed to parse path %s: %w", path, err)
				}

				change, err := convertChangeDesc(resourceType, fieldPath, changeDesc)
				if err != nil {
					return nil, fmt.Errorf("failed to compute config change for path %s: %w", path, err)
				}
				if change.Operation == OperationSkip {
					continue
				}
				change.Value = stripNamePrefix(resourceType, fieldPath, change.Value, b.Config.Presets.NamePrefix)
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

func ensureSnapshotAvailable(ctx context.Context, b *bundle.Bundle, engine engine.EngineType) error {
	if engine.IsDirect() {
		return nil
	}

	remotePathSnapshot, localPathSnapshot := b.StateFilenameConfigSnapshot(ctx)

	if _, err := os.Stat(localPathSnapshot); err == nil {
		return nil
	} else if !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("checking snapshot file: %w", err)
	}

	log.Debugf(ctx, "Resources state snapshot not found locally, pulling from remote")

	f, err := deploy.StateFiler(ctx, b)
	if err != nil {
		return fmt.Errorf("getting state filer: %w", err)
	}

	r, err := f.Read(ctx, remotePathSnapshot)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("resources state snapshot not found remotely at %s", remotePathSnapshot)
		}
		return fmt.Errorf("reading remote snapshot: %w", err)
	}
	defer r.Close()

	content, err := io.ReadAll(r)
	if err != nil {
		return fmt.Errorf("reading snapshot content: %w", err)
	}

	localStateDir := filepath.Dir(localPathSnapshot)
	err = os.MkdirAll(localStateDir, 0o700)
	if err != nil {
		return fmt.Errorf("creating snapshot directory: %w", err)
	}

	err = os.WriteFile(localPathSnapshot, content, 0o600)
	if err != nil {
		return fmt.Errorf("writing snapshot file: %w", err)
	}

	log.Debugf(ctx, "Pulled config snapshot from remote to %s", localPathSnapshot)
	return nil
}
