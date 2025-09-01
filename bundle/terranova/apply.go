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

type Deployer struct {
	client       *databricks.WorkspaceClient
	db           *tnstate.TerranovaState
	group        string
	resourceName string
	settings     ResourceSettings
}

func (d *Deployer) Destroy(ctx context.Context) error {
	entry, hasEntry := d.db.GetResourceEntry(d.group, d.resourceName)
	if !hasEntry {
		log.Infof(ctx, "Cannot delete %s.%s: missing from state", d.group, d.resourceName)
		return nil
	}

	if entry.ID == "" {
		return errors.New("invalid state: empty id")
	}

	err := d.Delete(ctx, entry.ID)
	if err != nil {
		return err
	}

	return nil
}

func (d *Deployer) Deploy(ctx context.Context, inputConfig any, actionType deployplan.ActionType) error {
	resource, _, err := New(d.client, d.group, d.resourceName, inputConfig)
	if err != nil {
		return err
	}

	config := resource.Config()

	if actionType == deployplan.ActionTypeCreate {
		return d.Create(ctx, resource, config)
	}

	entry, hasEntry := d.db.GetResourceEntry(d.group, d.resourceName)
	if !hasEntry {
		return errors.New("state entry not found")
	}

	oldID := entry.ID
	if oldID == "" {
		return errors.New("invalid state: empty id")
	}

	switch actionType {
	case deployplan.ActionTypeRecreate:
		return d.Recreate(ctx, resource, oldID, config)
	case deployplan.ActionTypeUpdate:
		return d.Update(ctx, resource, oldID, config)
	case deployplan.ActionTypeUpdateWithID:
		updater, hasUpdater := resource.(IResourceUpdatesID)
		if !hasUpdater {
			return errors.New("internal error: plan is update_with_id but resource does not implement UpdateWithID")
		}
		return d.UpdateWithID(ctx, resource, updater, oldID, config)
	default:
		return fmt.Errorf("internal error: unexpected actionType: %#v", actionType)
	}
}

func (d *Deployer) Create(ctx context.Context, resource IResource, config any) error {
	newID, err := resource.DoCreate(ctx)
	if err != nil {
		// No need to prefix error, there is no ambiguity (only one operation - DoCreate) and no additional context (like id)
		return err
	}

	log.Infof(ctx, "Created %s.%s id=%#v", d.group, d.resourceName, newID)

	err = d.db.SaveState(d.group, d.resourceName, newID, config)
	if err != nil {
		return fmt.Errorf("saving state after creating id=%s: %w", newID, err)
	}

	err = resource.WaitAfterCreate(ctx)
	if err != nil {
		return fmt.Errorf("waiting after creating id=%s: %w", newID, err)
	}

	return nil
}

func (d *Deployer) Recreate(ctx context.Context, resource IResource, oldID string, config any) error {
	err := DeleteResource(ctx, d.client, d.group, oldID)
	if err != nil {
		return fmt.Errorf("deleting old id=%s: %w", oldID, err)
	}

	err = d.db.SaveState(d.group, d.resourceName, "", nil)
	if err != nil {
		return fmt.Errorf("deleting state: %w", err)
	}

	newID, err := resource.DoCreate(ctx)
	if err != nil {
		return fmt.Errorf("re-creating: %w", err)
	}

	// TODO: This should be at notice level (info < notice < warn) and it should be visible by default,
	// but to match terraform output today, we hide it (and also we don't have notice level)
	log.Infof(ctx, "Recreated %s.%s id=%#v (previously %#v)", d.group, d.resourceName, newID, oldID)
	err = d.db.SaveState(d.group, d.resourceName, newID, config)
	if err != nil {
		return fmt.Errorf("saving state for id=%s: %w", newID, err)
	}

	err = resource.WaitAfterCreate(ctx)
	if err != nil {
		return fmt.Errorf("waiting after re-creating id=%s: %w", newID, err)
	}

	return nil
}

func (d *Deployer) Update(ctx context.Context, resource IResource, id string, config any) error {
	err := resource.DoUpdate(ctx, id)
	if err != nil {
		return fmt.Errorf("updating id=%s: %w", id, err)
	}

	err = d.db.SaveState(d.group, d.resourceName, id, config)
	if err != nil {
		return fmt.Errorf("saving state id=%s: %w", id, err)
	}

	err = resource.WaitAfterUpdate(ctx)
	if err != nil {
		return fmt.Errorf("waiting after updating id=%s: %w", id, err)
	}

	return nil
}

func (d *Deployer) UpdateWithID(ctx context.Context, resource IResource, updater IResourceUpdatesID, oldID string, config any) error {
	newID, err := updater.DoUpdateWithID(ctx, oldID)
	if err != nil {
		return fmt.Errorf("updating id=%s: %w", oldID, err)
	}

	if oldID != newID {
		log.Infof(ctx, "Updated %s.%s id=%#v (previously %#v)", d.group, d.resourceName, newID, oldID)
	} else {
		log.Infof(ctx, "Updated %s.%s id=%#v", d.group, d.resourceName, newID)
	}

	err = d.db.SaveState(d.group, d.resourceName, newID, config)
	if err != nil {
		return fmt.Errorf("saving state id=%s: %w", oldID, err)
	}

	err = resource.WaitAfterUpdate(ctx)
	if err != nil {
		return fmt.Errorf("waiting after updating id=%s: %w", newID, err)
	}

	return nil
}

func (d *Deployer) Delete(ctx context.Context, oldID string) error {
	// TODO: recognize 404 and 403 as "deleted" and proceed to removing state
	err := DeleteResource(ctx, d.client, d.group, oldID)
	if err != nil {
		return fmt.Errorf("deleting id=%s: %w", oldID, err)
	}

	err = d.db.DeleteState(d.group, d.resourceName)
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
