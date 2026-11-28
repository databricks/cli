package direct

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/direct/dresources"
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

// PreparedBind holds the state to save after approval and the preview of the next deploy.
type PreparedBind struct {
	Resource dstate.ResourceEntry
	// HasChanges is true if deploying after bind would make changes to the resource.
	HasChanges bool
	// Action is the planned action for the bound resource (e.g., "skip", "update", "recreate").
	Action deployplan.ActionType
	// Plan contains the full deployment plan for the bound resource.
	Plan *deployplan.Plan
}

// Bind prepares a binding in memory and previews the next deploy. The actual state stays
// open read-only; the caller must finalize it after committing the prepared resource or aborting.
func (b *DeploymentBundle) Bind(ctx context.Context, client *databricks.WorkspaceClient, configRoot *config.Root, statePath, resourceKey, resourceID string) (*PreparedBind, error) {
	// Check if the resource is already managed (bound to a different ID)
	if err := b.StateDB.Open(ctx, statePath, dstate.ReadingFile, dstate.WithRecovery(true), dstate.OpenDmsArgs{}); err != nil {
		// State that cannot be read is not the same as state without a binding:
		// the resource may well be managed already, and binding on top of it
		// would take over a resource whose ownership was never checked. A state
		// file that does not exist yet opens successfully as an empty one.
		return nil, fmt.Errorf("cannot check whether %s is already bound: %w", resourceKey, err)
	}
	if existingID := b.StateDB.GetResourceID(resourceKey); existingID != "" {
		return nil, ErrResourceAlreadyBound{
			ResourceKey: resourceKey,
			ExistingID:  existingID,
			NewID:       resourceID,
		}
	}

	// Keep the planner's state, caches, and locks separate from the approved state.
	data := b.StateDB.Data.Clone()
	data.State[resourceKey] = dstate.ResourceEntry{ID: resourceID, State: json.RawMessage(`{}`)}
	// Adapters hold no per-resource state and can be shared.
	preview := &DeploymentBundle{Adapters: b.Adapters}
	preview.StateDB.OpenWithData(statePath, data)
	defer func() {
		if _, err := preview.StateDB.Finalize(ctx); err != nil {
			log.Warnf(ctx, "failed to finalize binding preview: %v", err)
		}
	}()

	// First plan + update: populate state with resolved config
	plan, err := preview.CalculatePlan(ctx, client, configRoot)
	if err != nil {
		return nil, err
	}
	entry := plan.Plan[resourceKey]
	sv, ok := preview.StateCache.Load(resourceKey)
	if ok && sv != nil {
		var dependsOn []deployplan.DependsOnEntry
		if entry != nil {
			dependsOn = slices.Clone(entry.DependsOn)
		}

		// Copy etag from remote state for resources that use etag-based drift
		// detection (dashboards and genie spaces). The etag is not provided by the
		// user; it comes from remote. If we don't store it in state, we won't
		// detect remote drift correctly and the next plan shows a bogus update.
		if (strings.Contains(resourceKey, ".dashboards.") || strings.Contains(resourceKey, ".genie_spaces.")) && entry != nil && entry.RemoteState != nil {
			etag, err := structaccess.Get(entry.RemoteState, structpath.NewStringKey(nil, "etag"))
			if err == nil && etag != nil {
				if etagStr, ok := etag.(string); ok && etagStr != "" {
					_ = structaccess.Set(sv.Value, structpath.NewStringKey(nil, "etag"), etagStr)
				}
			}
		}

		// Compact hashed_fields fields so the persisted state stays small. Not needed for
		// correctness — the next plan (CalculatePlan) compacts the saved state on read.
		adapter, err := preview.getAdapterForKey(resourceKey)
		if err != nil {
			return nil, err
		}
		compacted, err := dresources.CompactState(adapter.ResourceConfig(), sv.Value)
		if err != nil {
			return nil, fmt.Errorf("compacting state: %w", err)
		}

		state, err := json.Marshal(compacted)
		if err != nil {
			return nil, err
		}
		data.State[resourceKey] = dstate.ResourceEntry{ID: resourceID, State: state, DependsOn: dependsOn}
	}

	// Capture unredacted state before the second plan replaces cached values.
	resource := data.State[resourceKey]
	resource.State = slices.Clone(resource.State)
	resource.DependsOn = slices.Clone(resource.DependsOn)
	if _, err := preview.StateDB.Finalize(ctx); err != nil {
		return nil, err
	}
	preview.StateDB.OpenWithData(statePath, data)

	// Second plan: this is the plan to present to the user (change between remote resource and config)
	plan, err = preview.CalculatePlan(ctx, client, configRoot)
	if err != nil {
		return nil, err
	}
	result := &PreparedBind{Resource: resource, Action: deployplan.Skip, Plan: plan}
	entry = plan.Plan[resourceKey]
	if entry != nil {
		result.Action = entry.Action
		// A state-only entry only records something in the state (such as an internal ID a bind
		// does not write), so it changes nothing remotely.
		result.HasChanges = result.Action != deployplan.Skip && result.Action != deployplan.Undefined && !entry.IsStateOnly()
	}
	return result, nil
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
