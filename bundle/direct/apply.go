package direct

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/direct/dresources"
	"github.com/databricks/cli/bundle/direct/dstate"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go/apierr"
)

func (d *DeploymentUnit) Destroy(ctx context.Context, db *dstate.DeploymentState) error {
	ctx = log.WithPrefix(ctx, "destroying "+d.ResourceKey)
	id := db.GetResourceID(d.ResourceKey)
	if id == "" {
		log.Infof(ctx, "Cannot delete %s: missing from state", d.ResourceKey)
		return nil
	}

	return d.Delete(ctx, db, id)
}

func (d *DeploymentUnit) Deploy(ctx context.Context, db *dstate.DeploymentState, newState any, actionType deployplan.ActionType, planEntry *deployplan.PlanEntry) error {
	ctx = log.WithPrefix(ctx, "deploying "+d.ResourceKey)
	if actionType == deployplan.Create {
		return d.Create(ctx, db, newState)
	}

	oldID := db.GetResourceID(d.ResourceKey)
	if oldID == "" {
		return errors.New("state entry not found")
	}

	switch actionType {
	case deployplan.Recreate:
		return d.Recreate(ctx, db, oldID, newState)
	case deployplan.Update:
		return d.Update(ctx, db, oldID, newState, planEntry)
	case deployplan.UpdateWithID:
		return d.UpdateWithID(ctx, db, oldID, newState)
	case deployplan.Resize:
		return d.Resize(ctx, db, oldID, newState)
	default:
		return fmt.Errorf("internal error: unexpected actionType: %#v", actionType)
	}
}

func (d *DeploymentUnit) Create(ctx context.Context, db *dstate.DeploymentState, newState any) error {
	var newID string
	var remoteState any
	_, err := retryWith(ctx, func(err error) bool {
		// For DoCreate, retry feature is opt-in via retrySafe(err) error wrapper
		_, ok := errors.AsType[*dresources.RetrySafeError](err)
		return ok && isTransient(ctx, err)
	}, func() (struct{}, error) {
		var e error
		newID, remoteState, e = d.Adapter.DoCreate(ctx, newState)
		return struct{}{}, e
	})
	err = dresources.UnwrapRetrySafe(err)
	if err != nil {
		// No need to prefix error, there is no ambiguity (only one operation - DoCreate) and no additional context (like id)
		return err
	}

	log.Infof(ctx, "Created %s id=%#v", d.ResourceKey, newID)

	err = d.SetRemoteState(remoteState)
	if err != nil {
		return err
	}

	err = db.SaveState(d.ResourceKey, newID, newState, d.DependsOn)
	if err != nil {
		return fmt.Errorf("saving state after creating id=%s: %w", newID, err)
	}

	// The resource may not be immediately visible after creation (eventual consistency).
	waitRemoteState, err := retryOnTransientOrMissing(ctx, func() (any, error) {
		return d.Adapter.WaitAfterCreate(ctx, newID, newState)
	})
	if err != nil {
		return fmt.Errorf("waiting after creating id=%s: %w", newID, err)
	}

	err = d.SetRemoteState(waitRemoteState)
	if err != nil {
		return err
	}

	return nil
}

func (d *DeploymentUnit) Recreate(ctx context.Context, db *dstate.DeploymentState, oldID string, newState any) error {
	oldState, err := d.loadPersistedState(db)
	if err != nil {
		return err
	}

	// Note, unlike Delete(), we hard error on 403 here intentionally.
	// MANAGED_BY_PARENT is still disregarded — the subsequent Create with
	// replace_existing=true will reconfigure the parent-managed resource in
	// place, matching the Terraform provider's recreate behaviour.
	err = retryOnTransientErr(ctx, func() error { return d.Adapter.DoDelete(ctx, oldID, oldState) })
	if err != nil && !apierr.IsMissing(err) && !isManagedByParent(err) {
		return fmt.Errorf("deleting old id=%s: %w", oldID, err)
	}

	// Drop the state entry so a subsequent failure of Create or WaitAfterDelete
	// leaves no malformed (empty-ID) entry behind. The next plan will see "no
	// state" and retry as Create.
	err = db.DeleteState(d.ResourceKey)
	if err != nil {
		return fmt.Errorf("deleting state: %w", err)
	}

	// Wait for asynchronous teardown to finish before re-creating the same
	// name. Done after DeleteState so the bundle stays consistent if the wait
	// times out — the resource is no longer tracked in state, retry on next plan.
	err = d.Adapter.WaitAfterDelete(ctx, oldID)
	if err != nil {
		return fmt.Errorf("waiting after deleting id=%s: %w", oldID, err)
	}

	return d.Create(ctx, db, newState)
}

func (d *DeploymentUnit) Update(ctx context.Context, db *dstate.DeploymentState, id string, newState any, planEntry *deployplan.PlanEntry) error {
	if !d.Adapter.HasDoUpdate() {
		return fmt.Errorf("internal error: DoUpdate not implemented for resource %s", d.ResourceKey)
	}

	remoteState, err := retryOnTransient(ctx, func() (any, error) {
		return d.Adapter.DoUpdate(ctx, id, newState, planEntry)
	})
	if err != nil {
		return fmt.Errorf("updating id=%s: %w", id, err)
	}

	err = d.SetRemoteState(remoteState)
	if err != nil {
		return err
	}

	err = db.SaveState(d.ResourceKey, id, newState, d.DependsOn)
	if err != nil {
		return fmt.Errorf("saving state id=%s: %w", id, err)
	}

	waitRemoteState, err := retryOnTransient(ctx, func() (any, error) {
		return d.Adapter.WaitAfterUpdate(ctx, id, newState)
	})
	if err != nil {
		return fmt.Errorf("waiting after updating id=%s: %w", id, err)
	}

	// Update remote state with the result from wait operation
	err = d.SetRemoteState(waitRemoteState)
	if err != nil {
		return err
	}

	return nil
}

func (d *DeploymentUnit) UpdateWithID(ctx context.Context, db *dstate.DeploymentState, oldID string, newState any) error {
	var newID string
	var remoteState any
	err := retryOnTransientErr(ctx, func() error {
		var e error
		newID, remoteState, e = d.Adapter.DoUpdateWithID(ctx, oldID, newState)
		return e
	})
	if err != nil {
		return fmt.Errorf("updating id=%s: %w", oldID, err)
	}

	if oldID != newID {
		log.Infof(ctx, "Updated %s id=%#v (previously %#v)", d.ResourceKey, newID, oldID)
	} else {
		log.Infof(ctx, "Updated %s id=%#v", d.ResourceKey, newID)
	}

	err = d.SetRemoteState(remoteState)
	if err != nil {
		return err
	}

	err = db.SaveState(d.ResourceKey, newID, newState, d.DependsOn)
	if err != nil {
		return fmt.Errorf("saving state id=%s: %w", oldID, err)
	}

	waitRemoteState, err := retryOnTransient(ctx, func() (any, error) {
		return d.Adapter.WaitAfterUpdate(ctx, newID, newState)
	})
	if err != nil {
		return fmt.Errorf("waiting after updating id=%s: %w", newID, err)
	}

	// Update remote state with the result from wait operation
	err = d.SetRemoteState(waitRemoteState)
	if err != nil {
		return err
	}

	return nil
}

func (d *DeploymentUnit) Delete(ctx context.Context, db *dstate.DeploymentState, oldID string) error {
	oldState, err := d.loadPersistedState(db)
	if err != nil {
		return err
	}

	err = d.Adapter.DoDelete(ctx, oldID, oldState)
	if err != nil && !apierr.IsMissing(err) && !isManagedByParent(err) {
		// Rather than failing delete and requiring user to unbind, we perform unbind automatically there.
		// Some services, e.g. jobs, return 403 for missing resources if caller did not have permissions to it when job existed.
		// In those cases 403 hides 404. In other cases, user not having permissions to resource but having in the bundle might
		// mean configuration error that user is trying to fix by removing resource from their bundle.
		if errors.Is(err, apierr.ErrPermissionDenied) {
			log.Warnf(ctx, "Ignoring permission error when deleting %s id=%s: %s", d.ResourceKey, oldID, err)
		} else {
			return fmt.Errorf("deleting id=%s: %w", oldID, err)
		}
	}

	err = db.DeleteState(d.ResourceKey)
	if err != nil {
		return fmt.Errorf("deleting state id=%s: %w", oldID, err)
	}

	// Wait for asynchronous teardown after dropping state. Mirrors Recreate so
	// the contract is the same regardless of whether the user triggered
	// `bundle destroy` or a recreate.
	err = d.Adapter.WaitAfterDelete(ctx, oldID)
	if err != nil {
		return fmt.Errorf("waiting after deleting id=%s: %w", oldID, err)
	}

	return nil
}

func (d *DeploymentUnit) Resize(ctx context.Context, db *dstate.DeploymentState, id string, newState any) error {
	err := retryOnTransientErr(ctx, func() error { return d.Adapter.DoResize(ctx, id, newState) })
	if err != nil {
		return fmt.Errorf("resizing id=%s: %w", id, err)
	}

	err = db.SaveState(d.ResourceKey, id, newState, d.DependsOn)
	if err != nil {
		return fmt.Errorf("saving state id=%s: %w", id, err)
	}

	return nil
}

func parseState(destType reflect.Type, raw json.RawMessage) (any, error) {
	destPtr := reflect.New(destType).Interface()
	err := json.Unmarshal(raw, destPtr)
	if err != nil {
		return nil, fmt.Errorf("unmarshalling into %s: %w", destType, err)
	}

	return reflect.ValueOf(destPtr).Elem().Interface(), nil
}

// loadPersistedState reads and parses the resource's last-persisted state for
// the DoDelete call. Returns a zero-value state pointer when nothing has been
// persisted yet (e.g. delete after a partial-create failure), so the call site
// always passes a typed value.
func (d *DeploymentUnit) loadPersistedState(db *dstate.DeploymentState) (any, error) {
	stateType := d.Adapter.StateType()
	dbentry, ok := db.GetResourceEntry(d.ResourceKey)
	if !ok || len(dbentry.State) == 0 {
		return reflect.New(stateType.Elem()).Interface(), nil
	}
	state, err := parseState(stateType, dbentry.State)
	if err != nil {
		return nil, fmt.Errorf("parsing persisted state: %w", err)
	}
	return state, nil
}

func (d *DeploymentUnit) refreshRemoteState(ctx context.Context, id string) error {
	if d.RemoteState != nil {
		return nil
	}
	// Retry on 404: the resource may not be visible yet after a recent create
	// (eventual consistency). The engine knows the resource should exist because
	// it has the ID on record.
	remoteState, err := retryOnTransientOrMissing(ctx, func() (any, error) {
		return d.Adapter.DoRead(ctx, id)
	})
	if err != nil {
		return fmt.Errorf("failed to refresh remote state id=%s: %w", id, err)
	}
	return d.SetRemoteState(remoteState)
}
