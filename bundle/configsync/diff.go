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
	"github.com/databricks/cli/bundle/config/engine"
	"github.com/databricks/cli/bundle/deploy"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/direct"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/databricks/cli/libs/log"
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

		if shouldSkipField(fieldPath, val) {
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
	hasConfigValue := cd.Old != nil || cd.New != nil
	normalizedValue, err := normalizeValue(cd.Remote)
	if err != nil {
		return nil, fmt.Errorf("failed to normalize remote value: %w", err)
	}

	if shouldSkipField(path, normalizedValue) {
		// If the config has an explicit value for a server-side default field,
		// we should remove it since the remote value is the default.
		if cd.New != nil {
			return &ConfigChangeDesc{
				Operation: OperationRemove,
			}, nil
		}
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

// DetectChanges compares current remote state with the last deployed state
// and returns a map of resource changes.
func DetectChanges(ctx context.Context, b *bundle.Bundle, engine engine.EngineType) (Changes, error) {
	changes := make(Changes)

	err := ensureSnapshotAvailable(ctx, b, engine)
	if err != nil {
		return nil, fmt.Errorf("state snapshot not available: %w", err)
	}

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
				if changeDesc.Action == deployplan.Skip {
					continue
				}

				change, err := convertChangeDesc(resourceKey+"."+path, changeDesc)
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

	f, err := deploy.StateFiler(b)
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
