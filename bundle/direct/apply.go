package direct

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/direct/dstate"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go/apierr"
)

func (d *DeploymentUnit) Destroy(ctx context.Context, db *dstate.DeploymentState) error {
	entry, hasEntry := db.GetResourceEntry(d.ResourceKey)
	if !hasEntry {
		log.Infof(ctx, "Cannot delete %s: missing from state", d.ResourceKey)
		return nil
	}

	if entry.ID == "" {
		return errors.New("invalid state: empty id")
	}

	return d.Delete(ctx, db, entry.ID)
}

func (d *DeploymentUnit) Deploy(ctx context.Context, db *dstate.DeploymentState, newState any, actionType deployplan.ActionType, changes deployplan.Changes) error {
	if actionType == deployplan.Create {
		return d.Create(ctx, db, newState)
	}

	entry, hasEntry := db.GetResourceEntry(d.ResourceKey)
	if !hasEntry {
		return errors.New("state entry not found")
	}

	oldID := entry.ID
	if oldID == "" {
		return errors.New("invalid state: empty id")
	}

	switch actionType {
	case deployplan.Recreate:
		return d.Recreate(ctx, db, oldID, newState)
	case deployplan.Update:
		return d.Update(ctx, db, oldID, newState, changes)
	case deployplan.UpdateWithID:
		return d.UpdateWithID(ctx, db, oldID, newState)
	case deployplan.Resize:
		return d.Resize(ctx, db, oldID, newState)
	default:
		return fmt.Errorf("internal error: unexpected actionType: %#v", actionType)
	}
}

func (d *DeploymentUnit) Create(ctx context.Context, db *dstate.DeploymentState, newState any) error {
	newID, remoteState, err := d.Adapter.DoCreate(ctx, newState)
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

	waitRemoteState, err := d.Adapter.WaitAfterCreate(ctx, newState)
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
	// Note, unlike Delete(), we hard error on 403 here intentionally
	err := d.Adapter.DoDelete(ctx, oldID)
	if err != nil && !isResourceGone(err) {
		return fmt.Errorf("deleting old id=%s: %w", oldID, err)
	}

	err = db.SaveState(d.ResourceKey, "", nil, nil)
	if err != nil {
		return fmt.Errorf("deleting state: %w", err)
	}

	return d.Create(ctx, db, newState)
}

func (d *DeploymentUnit) Update(ctx context.Context, db *dstate.DeploymentState, id string, newState any, changes deployplan.Changes) error {
	if !d.Adapter.HasDoUpdate() {
		return fmt.Errorf("internal error: DoUpdate not implemented for resource %s", d.ResourceKey)
	}

	remoteState, err := d.Adapter.DoUpdate(ctx, id, newState, changes)
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

	waitRemoteState, err := d.Adapter.WaitAfterUpdate(ctx, newState)
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
	newID, remoteState, err := d.Adapter.DoUpdateWithID(ctx, oldID, newState)
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

	waitRemoteState, err := d.Adapter.WaitAfterUpdate(ctx, newState)
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
	err := d.Adapter.DoDelete(ctx, oldID)
	if err != nil && !isResourceGone(err) {
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

	return nil
}

func (d *DeploymentUnit) Resize(ctx context.Context, db *dstate.DeploymentState, id string, newState any) error {
	err := d.Adapter.DoResize(ctx, id, newState)
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

func (d *DeploymentUnit) refreshRemoteState(ctx context.Context, id string) error {
	if d.RemoteState != nil {
		return nil
	}
	remoteState, err := d.Adapter.DoRead(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to refresh remote state id=%s: %w", id, err)
	}
	return d.SetRemoteState(remoteState)
}
