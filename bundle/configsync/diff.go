package configsync

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/engine"
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

func convertChangeDesc(path string, cd *deployplan.ChangeDesc) (*ConfigChangeDesc, error) {
	hasConfigValue := cd.Old != nil || cd.New != nil
	normalizedValue, err := normalizeValue(cd.Remote)
	if err != nil {
		return nil, fmt.Errorf("failed to normalize remote value: %w", err)
	}

	op := OperationUnknown
	if shouldSkipField(path, normalizedValue) {
		return &ConfigChangeDesc{
			Operation: OperationSkip,
		}, nil
	}

	normalizedValue = resetValueIfNeeded(path, normalizedValue)

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
				// TODO: as for now in bundle plan all remote-side changes are considered as server-side defaults.
				// Once it is solved - stop skipping server-side defaults in these checks and remove hardcoded default.
				if changeDesc.Action == deployplan.Skip && changeDesc.Reason != deployplan.ReasonServerSideDefault {
					continue
				}

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
