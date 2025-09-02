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
)

func (d *DeploymentUnit) Destroy(ctx context.Context, db *tnstate.TerranovaState) error {
	entry, hasEntry := db.GetResourceEntry(d.Group, d.Key)
	if !hasEntry {
		log.Infof(ctx, "Cannot delete %s.%s: missing from state", d.Group, d.Key)
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

func (d *DeploymentUnit) Deploy(ctx context.Context, db *tnstate.TerranovaState, inputConfig any, actionType deployplan.ActionType) error {
	config, err := d.Adapter.PrepareConfig(inputConfig)
	if err != nil {
		return err
	}

	if actionType == deployplan.ActionTypeCreate {
		return d.Create(ctx, db, config)
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
		return d.Recreate(ctx, db, oldID, config)
	case deployplan.ActionTypeUpdate:
		return d.Update(ctx, db, oldID, config)
	case deployplan.ActionTypeUpdateWithID:
		return d.UpdateWithID(ctx, db, oldID, config)
	default:
		return fmt.Errorf("internal error: unexpected actionType: %#v", actionType)
	}
}

func (d *DeploymentUnit) Create(ctx context.Context, db *tnstate.TerranovaState, config any) error {
	newID, err := d.Adapter.DoCreate(ctx, config)
	if err != nil {
		// No need to prefix error, there is no ambiguity (only one operation - DoCreate) and no additional context (like id)
		return err
	}

	log.Infof(ctx, "Created %s.%s id=%#v", d.Group, d.Key, newID)

	err = db.SaveState(d.Group, d.Key, newID, config)
	if err != nil {
		return fmt.Errorf("saving state after creating id=%s: %w", newID, err)
	}

	err = d.Adapter.WaitAfterCreate(ctx, newID, config)
	if err != nil {
		return fmt.Errorf("waiting after creating id=%s: %w", newID, err)
	}

	return nil
}

func (d *DeploymentUnit) Recreate(ctx context.Context, db *tnstate.TerranovaState, oldID string, config any) error {
	err := d.Adapter.DoDelete(ctx, oldID)
	if err != nil {
		return fmt.Errorf("deleting old id=%s: %w", oldID, err)
	}

	err = db.SaveState(d.Group, d.Key, "", nil)
	if err != nil {
		return fmt.Errorf("deleting state: %w", err)
	}

	return d.Create(ctx, db, config)
}

func (d *DeploymentUnit) Update(ctx context.Context, db *tnstate.TerranovaState, id string, config any) error {
	err := d.Adapter.DoUpdate(ctx, id, config)
	if err != nil {
		return fmt.Errorf("updating id=%s: %w", id, err)
	}

	err = db.SaveState(d.Group, d.Key, id, config)
	if err != nil {
		return fmt.Errorf("saving state id=%s: %w", id, err)
	}

	err = d.Adapter.WaitAfterUpdate(ctx, id, config)
	if err != nil {
		return fmt.Errorf("waiting after updating id=%s: %w", id, err)
	}

	return nil
}

func (d *DeploymentUnit) UpdateWithID(ctx context.Context, db *tnstate.TerranovaState, oldID string, config any) error {
	newID, err := d.Adapter.DoUpdateWithID(ctx, oldID, config)
	if err != nil {
		return fmt.Errorf("updating id=%s: %w", oldID, err)
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

	err = d.Adapter.WaitAfterUpdate(ctx, newID, config)
	if err != nil {
		return fmt.Errorf("waiting after updating id=%s: %w", newID, err)
	}

	return nil
}

func (d *DeploymentUnit) Delete(ctx context.Context, db *tnstate.TerranovaState, oldID string) error {
	// TODO: recognize 404 and 403 as "deleted" and proceed to removing state
	err := d.Adapter.DoDelete(ctx, oldID)
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
