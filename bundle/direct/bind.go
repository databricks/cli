package direct

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/direct/dstate"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/structs/structaccess"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/databricks-sdk-go"
)

// ErrResourceAlreadyBound is returned when attempting to bind a resource
// that is already bound to a different ID in the state.
type ErrResourceAlreadyBound struct {
	ResourceKey string
	ExistingID  string
	NewID       string
}

func (e ErrResourceAlreadyBound) Error() string {
	if e.ExistingID != e.NewID {
		return fmt.Sprintf("Resource already managed\n\n"+
			"The bundle is already managing a resource for %s with ID '%s'.\n"+
			"To bind to a different resource with ID '%s', you must first unbind the existing resource.",
			e.ResourceKey, e.ExistingID, e.NewID)
	}
	return fmt.Sprintf("Resource already managed\n\nThe bundle is already managing resource for %s and it is bound to the requested ID %s.",
		e.ResourceKey, e.ExistingID)
}

// BindResult contains the result of a bind operation including any detected changes.
type BindResult struct {
	// HasChanges is true if deploying after bind would make changes to the resource
	HasChanges bool
	// Action is the planned action for the bound resource (e.g., "skip", "update", "recreate")
	Action deployplan.ActionType
	// Plan contains the full deployment plan for the bound resource
	Plan *deployplan.Plan
	// TempStatePath is the path to the temporary state file
	TempStatePath string
	// StatePath is the path to the final state file
	StatePath string
}

// Bind adds an existing workspace resource to a temporary state and calculates
// if there will be any changes when deploying.
//
// Flow:
// 1. Load current state file
// 2. Update id of the target resource
// 3. Update state path to temp file (state file + ".temp-bind")
// 4. Perform plan + update to populate state with resolved config
// 5. Calculate plan again - this is the plan presented to the user
//
// Call FinalizeBind to commit the state or CancelBind to discard.
func (b *DeploymentBundle) Bind(ctx context.Context, client *databricks.WorkspaceClient, configRoot *config.Root, statePath, resourceKey, resourceID string) (*BindResult, error) {
	// Check if the resource is already managed (bound to a different ID)
	var checkStateDB dstate.DeploymentState
	if err := checkStateDB.Open(statePath); err == nil {
		if existing, ok := checkStateDB.GetResourceEntry(resourceKey); ok {
			return nil, ErrResourceAlreadyBound{
				ResourceKey: resourceKey,
				ExistingID:  existing.ID,
				NewID:       resourceID,
			}
		}
	}

	tmpStatePath := statePath + ".temp-bind"

	// Copy existing state to temp location if it exists
	if data, err := os.ReadFile(statePath); err == nil {
		if err := os.WriteFile(tmpStatePath, data, 0o600); err != nil {
			return nil, err
		}
	}

	// Open temp state
	err := b.StateDB.Open(tmpStatePath)
	if err != nil {
		os.Remove(tmpStatePath)
		return nil, err
	}

	// Save state with ID and empty state (like migrate does)
	err = b.StateDB.SaveState(resourceKey, resourceID, struct{}{}, nil)
	if err != nil {
		os.Remove(tmpStatePath)
		return nil, err
	}

	// Finalize to write temp state to disk so CalculatePlan can read it
	err = b.StateDB.Finalize()
	if err != nil {
		os.Remove(tmpStatePath)
		return nil, err
	}

	log.Infof(ctx, "Bound %s to id=%s (in temp state)", resourceKey, resourceID)

	// First plan + update: populate state with resolved config
	plan, err := b.CalculatePlan(ctx, client, configRoot, tmpStatePath)
	if err != nil {
		os.Remove(tmpStatePath)
		return nil, err
	}

	// Populate the state with the resolved config
	entry := plan.Plan[resourceKey]
	sv, ok := b.StructVarCache.Load(resourceKey)
	if ok && sv != nil {
		var dependsOn []deployplan.DependsOnEntry
		if entry != nil {
			dependsOn = entry.DependsOn
		}

		// Copy etag from remote state for dashboards.
		// Dashboards store "etag" in state which is not provided by user but comes from remote.
		// If we don't store "etag" in state, we won't detect remote drift correctly.
		if strings.Contains(resourceKey, ".dashboards.") && entry != nil && entry.RemoteState != nil {
			etag, err := structaccess.Get(entry.RemoteState, structpath.NewStringKey(nil, "etag"))
			if err == nil && etag != nil {
				if etagStr, ok := etag.(string); ok && etagStr != "" {
					_ = structaccess.Set(sv.Value, structpath.NewStringKey(nil, "etag"), etagStr)
				}
			}
		}

		err = b.StateDB.SaveState(resourceKey, resourceID, sv.Value, dependsOn)
		if err != nil {
			os.Remove(tmpStatePath)
			return nil, err
		}

		err = b.StateDB.Finalize()
		if err != nil {
			os.Remove(tmpStatePath)
			return nil, err
		}
	}

	// Second plan: this is the plan to present to the user (change between remote resource and config)
	plan, err = b.CalculatePlan(ctx, client, configRoot, tmpStatePath)
	if err != nil {
		os.Remove(tmpStatePath)
		return nil, err
	}

	// Check if the bound resource has changes
	result := &BindResult{
		HasChanges:    false,
		Action:        deployplan.Skip,
		Plan:          plan,
		TempStatePath: tmpStatePath,
		StatePath:     statePath,
	}

	entry = plan.Plan[resourceKey]
	if entry != nil {
		result.Action = entry.Action
		result.HasChanges = result.Action != deployplan.Skip && result.Action != deployplan.Undefined
	}

	return result, nil
}

// FinalizeBind completes the bind operation by renaming the temp state to the final location.
func FinalizeBind(result *BindResult) error {
	if result == nil || result.TempStatePath == "" {
		return nil
	}
	return os.Rename(result.TempStatePath, result.StatePath)
}

// CancelBind cleans up temp state without committing changes.
func CancelBind(result *BindResult) {
	if result != nil && result.TempStatePath != "" {
		os.Remove(result.TempStatePath)
	}
}

// Unbind removes a resource from direct engine state without deleting
// the workspace resource. Also removes associated permissions/grants entries.
func (b *DeploymentBundle) Unbind(ctx context.Context, statePath, resourceKey string) error {
	err := b.StateDB.Open(statePath)
	if err != nil {
		return err
	}

	// Delete the main resource
	err = b.StateDB.DeleteState(resourceKey)
	if err != nil {
		return err
	}

	log.Infof(ctx, "Unbound %s", resourceKey)

	// Also delete associated permissions/grants if they exist
	// These are stored as "resources.jobs.foo.permissions" or "resources.schemas.bar.grants"
	permissionsKey := resourceKey + ".permissions"
	grantsKey := resourceKey + ".grants"

	for key := range b.StateDB.Data.State {
		if key == permissionsKey || key == grantsKey || strings.HasPrefix(key, resourceKey+".") {
			err = b.StateDB.DeleteState(key)
			if err != nil {
				return err
			}
			log.Infof(ctx, "Unbound %s", key)
		}
	}

	return b.StateDB.Finalize()
}
