package terranova

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/terranova/tnstate"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go"
)

func (d *Deployer) Destroy(ctx context.Context, client *databricks.WorkspaceClient, db *tnstate.TerranovaState) error {
	entry, hasEntry := db.GetResourceEntry(d.Group, d.Key)
	if !hasEntry {
		log.Infof(ctx, "Cannot delete %s.%s: missing from state", d.Group, d.Key)
		return nil
	}

	if entry.ID == "" {
		return errors.New("invalid state: empty id")
	}

	err := d.Delete(ctx, client, db, entry.ID)
	if err != nil {
		return err
	}

	return nil
}

func (d *Deployer) Deploy(ctx context.Context, client *databricks.WorkspaceClient, db *tnstate.TerranovaState, inputConfig any, actionType deployplan.ActionType) error {
	resource, _, err := New(client, d.Group, d.Key, inputConfig)
	if err != nil {
		return err
	}

	config := resource.Config()

	if actionType == deployplan.ActionTypeCreate {
		return d.Create(ctx, client, db, resource, config)
	}

	entry, hasEntry := db.GetResourceEntry(d.Group, d.Key)
	if !hasEntry {
		return errors.New("state entry not found")
	}

	oldID := entry.ID
	if oldID == "" {
		return errors.New("invalid state: empty id")
	}

	switch actionType {
	case deployplan.ActionTypeRecreate:
		return d.Recreate(ctx, client, db, resource, oldID, config)
	case deployplan.ActionTypeUpdate:
		return d.Update(ctx, client, db, resource, oldID, config)
	case deployplan.ActionTypeUpdateWithID:
		updater, hasUpdater := resource.(IResourceUpdatesID)
		if !hasUpdater {
			return errors.New("internal error: plan is update_with_id but resource does not implement UpdateWithID")
		}
		return d.UpdateWithID(ctx, client, db, resource, updater, oldID, config)
	default:
		return fmt.Errorf("internal error: unexpected actionType: %#v", actionType)
	}
}

func (d *Deployer) Create(ctx context.Context, client *databricks.WorkspaceClient, db *tnstate.TerranovaState, resource IResource, config any) error {
	newID, remoteState, err := resource.DoCreate(ctx)
	if err != nil {
		// No need to prefix error, there is no ambiguity (only one operation - DoCreate) and no additional context (like id)
		return err
	}

	// Update remote state if returned and resource supports it
	if remoteState != nil {
		d.RemoteState = remoteState
	}

	log.Infof(ctx, "Created %s.%s id=%#v", d.Group, d.Key, newID)

	err = db.SaveState(d.Group, d.Key, newID, config)
	if err != nil {
		return fmt.Errorf("saving state after creating id=%s: %w", newID, err)
	}

	waitRemoteState, err := resource.WaitAfterCreate(ctx)
	if err != nil {
		return fmt.Errorf("waiting after creating id=%s: %w", newID, err)
	}

	// Update remote state from wait if returned
	if waitRemoteState != nil {
		d.RemoteState = waitRemoteState
	}

	return nil
}

func (d *Deployer) Recreate(ctx context.Context, client *databricks.WorkspaceClient, db *tnstate.TerranovaState, resource IResource, oldID string, config any) error {
	err := DeleteResource(ctx, client, d.Group, oldID)
	if err != nil {
		return fmt.Errorf("deleting old id=%s: %w", oldID, err)
	}

	err = db.SaveState(d.Group, d.Key, "", nil)
	if err != nil {
		return fmt.Errorf("deleting state: %w", err)
	}

	return d.Create(ctx, client, db, resource, config)
}

func (d *Deployer) Update(ctx context.Context, client *databricks.WorkspaceClient, db *tnstate.TerranovaState, resource IResource, id string, config any) error {
	remoteState, err := resource.DoUpdate(ctx, id)
	if err != nil {
		return fmt.Errorf("updating id=%s: %w", id, err)
	}

	// Update remote state if returned and resource supports it
	if remoteState != nil {
		d.RemoteState = remoteState
	}

	err = db.SaveState(d.Group, d.Key, id, config)
	if err != nil {
		return fmt.Errorf("saving state id=%s: %w", id, err)
	}

	waitRemoteState, err := resource.WaitAfterUpdate(ctx)
	if err != nil {
		return fmt.Errorf("waiting after updating id=%s: %w", id, err)
	}

	// Update remote state from wait if returned
	if waitRemoteState != nil {
		d.RemoteState = waitRemoteState
	}

	return nil
}

func (d *Deployer) UpdateWithID(ctx context.Context, client *databricks.WorkspaceClient, db *tnstate.TerranovaState, resource IResource, updater IResourceUpdatesID, oldID string, config any) error {
	newID, remoteState, err := updater.DoUpdateWithID(ctx, oldID)
	if err != nil {
		return fmt.Errorf("updating id=%s: %w", oldID, err)
	}

	// Update remote state if returned and resource supports it
	if remoteState != nil {
		d.RemoteState = remoteState
	}

	if oldID != newID {
		log.Infof(ctx, "Updated %s.%s id=%#v (previously %#v)", d.Group, d.Key, newID, oldID)
	} else {
		log.Infof(ctx, "Updated %s.%s id=%#v", d.Group, d.Key, newID)
	}

	err = db.SaveState(d.Group, d.Key, newID, config)
	if err != nil {
		return fmt.Errorf("saving state id=%s: %w", oldID, err)
	}

	waitRemoteState, err := resource.WaitAfterUpdate(ctx)
	if err != nil {
		return fmt.Errorf("waiting after updating id=%s: %w", newID, err)
	}

	// Update remote state from wait if returned
	if waitRemoteState != nil {
		d.RemoteState = waitRemoteState
	}

	return nil
}

func (d *Deployer) Delete(ctx context.Context, client *databricks.WorkspaceClient, db *tnstate.TerranovaState, oldID string) error {
	// TODO: recognize 404 and 403 as "deleted" and proceed to removing state
	err := DeleteResource(ctx, client, d.Group, oldID)
	if err != nil {
		return fmt.Errorf("deleting id=%s: %w", oldID, err)
	}

	err = db.DeleteState(d.Group, d.Key)
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
