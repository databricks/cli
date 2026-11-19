package direct

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"os"
	"slices"
	"strings"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/direct/dstate"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/logdiag"
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
	// Plan is the plan a deploy after the bind would run; it is the one presented to the user.
	Plan *deployplan.Plan
	// BindPlan records the bound resource in the state without touching the backend. Applying it
	// to StatePath completes the bind.
	BindPlan *deployplan.Plan
	// TempStatePath is the path to the temporary state file BindPlan was applied to
	TempStatePath string
	// StatePath is the path to the final state file
	StatePath string
}

// Bind adds an existing workspace resource to the state and calculates the changes a deploy would
// make to it afterwards. It reuses the regular plan and apply:
//
//  1. Plan against the current state with the resource recorded at resourceID and an empty state.
//     The bound resource's entry is turned into a state-only update: applying it records the
//     resolved config as the resource's state, with no backend call (the bind plan).
//  2. Apply the bind plan to a copy of the state (state file + ".temp-bind") and plan again
//     against it. This is the plan presented to the user.
//
// Call FinalizeBind to apply the bind plan to the state, or Cancel to discard.
func (b *DeploymentBundle) Bind(ctx context.Context, client *databricks.WorkspaceClient, configRoot *config.Root, statePath, resourceKey, resourceID string) (*BindResult, error) {
	err := b.StateDB.Open(ctx, statePath, dstate.ReadingFile, dstate.WithRecovery(true), dstate.OpenDmsArgs{})
	if err != nil {
		// State that cannot be read is not the same as state without a binding:
		// the resource may well be managed already, and binding on top of it
		// would take over a resource whose ownership was never checked. A state
		// file that does not exist yet opens successfully as an empty one.
		return nil, fmt.Errorf("cannot check whether %s is already bound: %w", resourceKey, err)
	}
	existingID := b.StateDB.GetResourceID(resourceKey)
	data := b.StateDB.Data
	if _, err := b.StateDB.Finalize(ctx); err != nil {
		log.Warnf(ctx, "failed to finalize state: %v", err)
	}
	if existingID != "" {
		return nil, ErrResourceAlreadyBound{
			ResourceKey: resourceKey,
			ExistingID:  existingID,
			NewID:       resourceID,
		}
	}

	bindPlan, err := b.calculateBindPlan(ctx, client, configRoot, statePath, data, resourceKey, resourceID)
	if err != nil {
		return nil, err
	}

	tmpStatePath := statePath + ".temp-bind"

	// Copy existing state to temp location if it exists
	if data, err := os.ReadFile(statePath); err == nil {
		if err := os.WriteFile(tmpStatePath, data, 0o600); err != nil {
			return nil, err
		}
	}

	if err := b.applyBindPlan(ctx, client, tmpStatePath, bindPlan); err != nil {
		os.Remove(tmpStatePath)
		return nil, err
	}

	log.Infof(ctx, "Bound %s to id=%s (in temp state)", resourceKey, resourceID)

	// Plan against the temp state: this is the plan to present to the user (change between remote
	// resource and config).
	err = b.StateDB.Open(ctx, tmpStatePath, dstate.ReadingFile, dstate.WithRecovery(true), dstate.OpenDmsArgs{})
	if err != nil {
		os.Remove(tmpStatePath)
		return nil, err
	}
	plan, err := b.CalculatePlan(ctx, client, configRoot)
	if _, ferr := b.StateDB.Finalize(ctx); ferr != nil {
		log.Warnf(ctx, "failed to finalize state: %v", ferr)
	}
	if err != nil {
		os.Remove(tmpStatePath)
		return nil, err
	}

	// Check if the bound resource has changes
	result := &BindResult{
		HasChanges:    false,
		Action:        deployplan.Skip,
		Plan:          plan,
		BindPlan:      bindPlan,
		TempStatePath: tmpStatePath,
		StatePath:     statePath,
	}

	entry := plan.Plan[resourceKey]
	if entry != nil {
		result.Action = entry.Action
		// A state-only entry only records something in the state (such as an internal ID a bind
		// does not write), so it changes nothing remotely.
		result.HasChanges = result.Action != deployplan.Skip && result.Action != deployplan.Undefined && !entry.IsStateOnly()
	}

	return result, nil
}

// calculateBindPlan plans against data with resourceKey recorded at resourceID and an empty
// state, and returns the plan that records the resource's resolved config as its state.
//
// The plan holds the bound resource as a state-only update. Its dependencies are kept as skipped
// entries: the graph needs every node an entry depends on, and the dependency edges are saved in
// the state alongside the resource.
func (b *DeploymentBundle) calculateBindPlan(ctx context.Context, client *databricks.WorkspaceClient, configRoot *config.Root, statePath string, data dstate.Database, resourceKey, resourceID string) (*deployplan.Plan, error) {
	data.State = maps.Clone(data.State)
	if data.State == nil {
		data.State = make(map[string]dstate.ResourceEntry)
	}
	// An empty state, like migrate records: the plan fills it in from the config.
	data.State[resourceKey] = dstate.ResourceEntry{ID: resourceID, State: json.RawMessage("{}")}

	b.StateDB.OpenWithData(statePath, data)
	fullPlan, err := b.CalculatePlan(ctx, client, configRoot)
	if _, ferr := b.StateDB.Finalize(ctx); ferr != nil {
		log.Warnf(ctx, "failed to finalize state: %v", ferr)
	}
	if err != nil {
		return nil, err
	}

	entry := fullPlan.Plan[resourceKey]
	// The resolved config, which the plan keeps even when it skips the entry.
	sv, ok := b.StateCache.Load(resourceKey)
	if entry == nil || !ok {
		return nil, fmt.Errorf("internal error: %s: missing from the plan", resourceKey)
	}

	// Copy etag from remote state for resources that use etag-based drift
	// detection (dashboards and genie spaces). The etag is not provided by the
	// user; it comes from remote. If we don't store it in state, we won't
	// detect remote drift correctly and the next plan shows a bogus update.
	if (strings.Contains(resourceKey, ".dashboards.") || strings.Contains(resourceKey, ".genie_spaces.")) && entry.RemoteState != nil {
		etag, err := structaccess.Get(entry.RemoteState, structpath.NewStringKey(nil, "etag"))
		if err == nil && etag != nil {
			if etagStr, ok := etag.(string); ok && etagStr != "" {
				_ = structaccess.Set(sv.Value, structpath.NewStringKey(nil, "etag"), etagStr)
			}
		}
	}

	newState, err := sv.ToJSON()
	if err != nil {
		return nil, fmt.Errorf("%s: cannot serialize state: %w", resourceKey, err)
	}
	// References planning could not resolve point at resources whose values are only known after
	// they are deployed. The state records them unresolved; the next deploy resolves them.
	newState.Refs = nil

	bindPlan := deployplan.NewPlanDirect()
	bindPlan.Lineage = fullPlan.Lineage
	bindPlan.Serial = fullPlan.Serial
	bindPlan.Features = fullPlan.Features
	bindPlan.Plan[resourceKey] = &deployplan.PlanEntry{
		// The state the plan is applied to does not have the resource yet.
		ID:          resourceID,
		DependsOn:   entry.DependsOn,
		Action:      deployplan.Update,
		StateOnly:   true,
		NewState:    newState,
		RemoteState: entry.RemoteState,
	}

	queue := slices.Clone(entry.DependsOn)
	for len(queue) > 0 {
		dep := queue[0].Node
		queue = queue[1:]
		if bindPlan.Plan[dep] != nil {
			continue
		}
		depEntry := fullPlan.Plan[dep]
		if depEntry == nil {
			return nil, fmt.Errorf("internal error: %s: dependency %s not in the plan", resourceKey, dep)
		}
		bindPlan.Plan[dep] = &deployplan.PlanEntry{
			DependsOn: depEntry.DependsOn,
			Action:    deployplan.Skip,
		}
		queue = append(queue, depEntry.DependsOn...)
	}

	return bindPlan, nil
}

// applyBindPlan applies the bind plan to the state at statePath with the regular deploy apply.
func (b *DeploymentBundle) applyBindPlan(ctx context.Context, client *databricks.WorkspaceClient, statePath string, bindPlan *deployplan.Plan) error {
	err := b.StateDB.Open(ctx, statePath, dstate.ReadingFile, dstate.WithRecovery(true), dstate.OpenDmsArgs{})
	if err != nil {
		return err
	}
	if err := b.StateDB.UpgradeToWrite(); err != nil {
		_, _ = b.StateDB.Finalize(ctx)
		return err
	}

	err = b.InitForApply(ctx, client, bindPlan)
	if err == nil {
		b.Apply(ctx, client, bindPlan, false)
	}

	// Finalize even if Apply failed, so the state stays consistent with what was applied.
	_, ferr := b.StateDB.Finalize(ctx)
	if err != nil {
		return err
	}
	if logdiag.HasError(ctx) {
		return errors.New("recording the bound resource in the state failed")
	}
	return ferr
}

// FinalizeBind completes the bind operation by applying the bind plan to the state.
func (b *DeploymentBundle) FinalizeBind(ctx context.Context, client *databricks.WorkspaceClient, result *BindResult) error {
	defer result.Cancel()
	return b.applyBindPlan(ctx, client, result.StatePath, result.BindPlan)
}

// Cancel cleans up temp state without committing changes.
func (result *BindResult) Cancel() {
	if result != nil && result.TempStatePath != "" {
		os.Remove(result.TempStatePath)
	}
}

// Unbind removes a resource from direct engine state without deleting
// the workspace resource. Also removes associated permissions/grants entries.
func (b *DeploymentBundle) Unbind(ctx context.Context, statePath, resourceKey string) error {
	if err := b.StateDB.Open(ctx, statePath, dstate.ReadingFile, dstate.WithRecovery(true), dstate.OpenDmsArgs{}); err != nil {
		return err
	}
	if err := b.StateDB.UpgradeToWrite(); err != nil {
		return err
	}

	// Delete the main resource
	err := b.StateDB.DeleteState(ctx, resourceKey, false)
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
			err = b.StateDB.DeleteState(ctx, key, false)
			if err != nil {
				return err
			}
			log.Infof(ctx, "Unbound %s", key)
		}
	}

	_, err = b.StateDB.Finalize(ctx)
	return err
}
