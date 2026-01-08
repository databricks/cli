package direct

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/direct/dstate"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go"
)

// BindResult contains the result of a bind operation including any detected changes.
type BindResult struct {
	// HasChanges is true if deploying after bind would make changes to the resource
	HasChanges bool
	// Action is the planned action for the bound resource (e.g., "skip", "update", "recreate")
	Action string
	// Plan contains the full deployment plan for the bound resource
	Plan *deployplan.Plan
}

// Bind adds an existing workspace resource to a temporary state and calculates
// if there will be any changes when deploying. This follows the same pattern
// as terraform import: import into temp state, run plan, check for changes.
// Returns a BindResult with the plan. Call FinalizeBind to commit the state to disk.
func (b *DeploymentBundle) Bind(ctx context.Context, client *databricks.WorkspaceClient, configRoot *config.Root, statePath, resourceKey, resourceID string) (*BindResult, error) {
	// Create a temporary directory for the temp state file
	tmpDir, err := os.MkdirTemp("", "state-*")
	if err != nil {
		return nil, err
	}

	tmpStatePath := filepath.Join(tmpDir, filepath.Base(statePath))

	// Copy existing state to temp location if it exists
	if data, err := os.ReadFile(statePath); err == nil {
		if err := os.WriteFile(tmpStatePath, data, 0o600); err != nil {
			os.RemoveAll(tmpDir)
			return nil, err
		}
	}

	// Use a separate StateDB for temp state
	var tmpStateDB dstate.DeploymentState
	err = tmpStateDB.Open(tmpStatePath)
	if err != nil {
		os.RemoveAll(tmpDir)
		return nil, err
	}

	// Save state with ID and empty state (like migrate does)
	err = tmpStateDB.SaveState(resourceKey, resourceID, struct{}{}, nil)
	if err != nil {
		os.RemoveAll(tmpDir)
		return nil, err
	}

	// Finalize to write temp state to disk so CalculatePlan can read it
	err = tmpStateDB.Finalize()
	if err != nil {
		os.RemoveAll(tmpDir)
		return nil, err
	}

	log.Infof(ctx, "Bound %s to id=%s (in temp state)", resourceKey, resourceID)

	// Calculate plan against temp state to detect if there will be changes
	plan, err := b.CalculatePlan(ctx, client, configRoot, tmpStatePath)
	if err != nil {
		os.RemoveAll(tmpDir)
		return nil, err
	}

	// Check if the bound resource has changes
	result := &BindResult{
		HasChanges: false,
		Action:     deployplan.ActionTypeSkipString,
		Plan:       plan,
	}

	entry, ok := plan.Plan[resourceKey]
	if ok && entry != nil {
		action := deployplan.ActionTypeFromString(entry.Action)
		result.Action = entry.Action
		result.HasChanges = action != deployplan.ActionTypeSkip && action != deployplan.ActionTypeUndefined
	}

	// Now populate the state with the resolved config (like migrate does)
	// This ensures the state has proper config for accurate plan comparisons
	// Note: CalculatePlan uses b.StateDB internally, so we save to that
	sv, ok := b.StructVarCache.Load(resourceKey)
	if ok && sv != nil {
		var dependsOn []deployplan.DependsOnEntry
		if entry != nil {
			dependsOn = entry.DependsOn
		}
		// b.StateDB was opened by CalculatePlan with tmpStatePath
		err = b.StateDB.SaveState(resourceKey, resourceID, sv.Value, dependsOn)
		if err != nil {
			os.RemoveAll(tmpDir)
			return nil, err
		}

		// Write updated state to temp file
		err = b.StateDB.Finalize()
		if err != nil {
			os.RemoveAll(tmpDir)
			return nil, err
		}
	}

	// Store temp dir and path for FinalizeBind
	b.tempBindDir = tmpDir
	b.tempBindStatePath = tmpStatePath
	b.finalStatePath = statePath

	return result, nil
}

// FinalizeBind completes the bind operation by moving the temp state to the final location.
func (b *DeploymentBundle) FinalizeBind(ctx context.Context) error {
	if b.tempBindDir == "" {
		return nil
	}

	defer func() {
		os.RemoveAll(b.tempBindDir)
		b.tempBindDir = ""
		b.tempBindStatePath = ""
		b.finalStatePath = ""
	}()

	// Read temp state and write to final location
	data, err := os.ReadFile(b.tempBindStatePath)
	if err != nil {
		return err
	}

	// Create parent directories if they don't exist
	dir := filepath.Dir(b.finalStatePath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	return os.WriteFile(b.finalStatePath, data, 0o600)
}

// CancelBind cleans up temp state without committing changes.
func (b *DeploymentBundle) CancelBind() {
	if b.tempBindDir != "" {
		os.RemoveAll(b.tempBindDir)
		b.tempBindDir = ""
		b.tempBindStatePath = ""
		b.finalStatePath = ""
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
