package direct

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/direct/dstate"
	"github.com/databricks/cli/libs/log"
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

	err := d.Delete(ctx, db, entry.ID)
	if err != nil {
		return err
	}

	return nil
}

func (d *DeploymentUnit) Deploy(ctx context.Context, db *dstate.DeploymentState, inputConfig any, actionType deployplan.ActionType) error {
	// Note, newState may be different between plan and deploy due to resolved $resource references
	newState, err := d.Adapter.PrepareState(inputConfig)
	if err != nil {
		return fmt.Errorf("reading config: %w", err)
	}

	if actionType == deployplan.ActionTypeCreate {
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
	case deployplan.ActionTypeRecreate:
		return d.Recreate(ctx, db, oldID, newState)
	case deployplan.ActionTypeUpdate:
		return d.Update(ctx, db, oldID, newState)
	case deployplan.ActionTypeUpdateWithID:
		return d.UpdateWithID(ctx, db, oldID, newState)
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

	err = db.SaveState(d.ResourceKey, newID, newState)
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
	err := d.Adapter.DoDelete(ctx, oldID)
	if err != nil {
		return fmt.Errorf("deleting old id=%s: %w", oldID, err)
	}

	err = db.SaveState(d.ResourceKey, "", nil)
	if err != nil {
		return fmt.Errorf("deleting state: %w", err)
	}

	return d.Create(ctx, db, newState)
}

func (d *DeploymentUnit) Update(ctx context.Context, db *dstate.DeploymentState, id string, newState any) error {
	remoteState, err := d.Adapter.DoUpdate(ctx, id, newState)
	if err != nil {
		return fmt.Errorf("updating id=%s: %w", id, err)
	}

	err = d.SetRemoteState(remoteState)
	if err != nil {
		return err
	}

	err = db.SaveState(d.ResourceKey, id, newState)
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

	err = db.SaveState(d.ResourceKey, newID, newState)
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
	// TODO: recognize 404 and 403 as "deleted" and proceed to removing state
	err := d.Adapter.DoDelete(ctx, oldID)
	if err != nil {
		return fmt.Errorf("deleting id=%s: %w", oldID, err)
	}

	err = db.DeleteState(d.ResourceKey)
	if err != nil {
		return fmt.Errorf("deleting state id=%s: %w", oldID, err)
	}

	return nil
}

func typeConvert(destType reflect.Type, src any) (any, error) {
	raw, err := json.Marshal(src)
	if err != nil {
		return nil, fmt.Errorf("marshalling: %w", err)
	}

	destPtr := reflect.New(destType).Interface()
	dec := json.NewDecoder(bytes.NewReader(raw))
	err = dec.Decode(destPtr)
	if err != nil {
		return nil, fmt.Errorf("unmarshalling into %s: %w", destType, err)
	}

	return reflect.ValueOf(destPtr).Elem().Interface(), nil
}
