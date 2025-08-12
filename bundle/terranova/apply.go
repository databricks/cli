package terranova

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/terranova/tnstate"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/libs/utils"
	"github.com/databricks/databricks-sdk-go"
)

// How many parallel operations (API calls) are allowed
const defaultParallelism = 10

type terranovaApplyMutator struct{}

func TerranovaApply() bundle.Mutator {
	return &terranovaApplyMutator{}
}

func (m *terranovaApplyMutator) Name() string {
	return "TerranovaDeploy"
}

func (m *terranovaApplyMutator) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	if len(b.Plan.Actions) == 0 {
		// Avoid creating state file if nothing to deploy
		return nil
	}

	// if true, then this node has references to its "id" field
	IDIsReferenced := map[nodeKey]bool{}

	// Maps node to all references within this node that need resolution
	fieldRefsMap := map[nodeKey][]fieldRef{}

	// Maps node key to originally planned action
	plannedActionsMap := map[nodeKey]deployplan.ActionType{}

	for _, action := range b.Plan.Actions {
		plannedActionsMap[nodeKey{action.Group, action.Name}] = action.ActionType
	}

	state := b.ResourceDatabase.ExportState(ctx)
	g, err := makeResourceGraph(ctx, b, state, fieldRefsMap)

	for _, fieldRefs := range fieldRefsMap {
		for _, fieldRef := range fieldRefs {
			for _, referencedNode := range fieldRef.referencedNodes {
				IDIsReferenced[referencedNode] = true
			}
		}
	}

	// Remained in state are resources that no longer present in the config
	for _, group := range utils.SortedKeys(state) {
		groupData := state[group]
		for _, name := range utils.SortedKeys(groupData) {
			n := nodeKey{group, name}
			g.AddNode(n)
			plannedActionsMap[n] = deployplan.ActionTypeDelete
		}
	}

	client := b.WorkspaceClient()

	err = g.Run(defaultParallelism, func(node nodeKey) {
		// TODO: if a given node fails, all downstream nodes should not be run. We should report those nodes.
		// TODO: ensure that config for this node is fully resolved at this point.

		settings, ok := SupportedResources[node.Group]
		if !ok {
			return
		}

		actionType := plannedActionsMap[node]

		if actionType == deployplan.ActionTypeUnset {
			return
			// TODO: ensure dependencies are not in the plan as well
		}

		d := Deployer{
			client:       client,
			db:           &b.ResourceDatabase,
			group:        node.Group,
			resourceName: node.Name,
			settings:     settings,
		}

		if actionType == deployplan.ActionTypeDelete {
			err = d.destroy(ctx)
			if err != nil {
				logdiag.LogError(ctx, fmt.Errorf("destroying %s.%s: %w", d.group, d.resourceName, err))
			}
			return
		}

		config, ok := b.GetResourceConfig(node.Group, node.Name)
		if !ok {
			logdiag.LogError(ctx, fmt.Errorf("internal error when reading config for %s.%s", node.Group, node.Name))
			return
		}

		// Extract unresolved references from a given node only.
		// We need to this here and not rely on fieldRefsMap because the config might have been updated with more resolutions.
		myReferences, err := extractReferences(b.Config.Value(), node)
		// log.Warnf(ctx, "extract myReferences=%v", myReferences)
		if err != nil {
			logdiag.LogError(ctx, err)
			return
		}

		// At this point it's an error to have unresolved deps
		if len(myReferences) > 0 {
			// TODO: include the deps themselves in the message
			logdiag.LogError(ctx, fmt.Errorf("cannot deploy %s.%s due to unresolved deps", node.Group, node.Name))
			return
		}

		// TODO: redo plan to downgrade planned action if possible (?)

		err = d.Deploy(ctx, config, actionType)
		if err != nil {
			logdiag.LogError(ctx, err)
			return
		}

		if IDIsReferenced[node] {
			err = resolveIDReference(ctx, b, d.group, d.resourceName)
			if err != nil {
				logdiag.LogError(ctx, fmt.Errorf("failed to replace ref to resources.%s.%s.id: %w", d.group, d.resourceName, err))
				return
			}
		}
	})
	if err != nil {
		logdiag.LogError(ctx, err)
	}

	err = b.ResourceDatabase.Finalize()
	if err != nil {
		logdiag.LogError(ctx, err)
	}

	return nil
}

type Deployer struct {
	client       *databricks.WorkspaceClient
	db           *tnstate.TerranovaState
	group        string
	resourceName string
	settings     ResourceSettings
}

func (d *Deployer) Deploy(ctx context.Context, inputConfig any, actionType deployplan.ActionType) error {
	err := d.deploy(ctx, inputConfig, actionType)
	if err != nil {
		return fmt.Errorf("deploying %s.%s: %w", d.group, d.resourceName, err)
	}
	return nil
}

func (d *Deployer) destroy(ctx context.Context) error {
	entry, hasEntry := d.db.GetResourceEntry(d.group, d.resourceName)
	if !hasEntry {
		log.Infof(ctx, "%s.%s: Cannot delete, missing from state", d.group, d.resourceName)
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

func (d *Deployer) deploy(ctx context.Context, inputConfig any, actionType deployplan.ActionType) error {
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
		return fmt.Errorf("internal error: unexpected plan: %#v", actionType)
	}
}

func (d *Deployer) Create(ctx context.Context, resource IResource, config any) error {
	newID, err := resource.DoCreate(ctx)
	if err != nil {
		return fmt.Errorf("creating: %w", err)
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
	// but to match terraform output today, we hide it.
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
