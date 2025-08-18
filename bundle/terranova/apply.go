package terranova

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"reflect"
	"slices"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/terranova/tnstate"
	"github.com/databricks/cli/libs/dagrun"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/logdiag"
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

	g, isReferenced, err := makeResourceGraph(ctx, b)
	if err != nil {
		logdiag.LogError(ctx, fmt.Errorf("error while reading config: %w", err))
	}

	// Maps node key to originally planned action
	plannedActionsMap := map[nodeKey]deployplan.ActionType{}

	for _, action := range b.Plan.Actions {
		node := nodeKey{action.Group, action.Name}
		plannedActionsMap[nodeKey{action.Group, action.Name}] = action.ActionType
		if !g.HasNode(node) {
			if action.ActionType == deployplan.ActionTypeDelete {
				// it is expected that this node is not seen by makeResourceGraph
				g.AddNode(node)
			} else {
				// it's internal error today because plan cannot be outdated. In the future when we load serialized plan, this will become user error
				logdiag.LogError(ctx, fmt.Errorf("cannot %s %s.%s: internal error, plan is outdated", action.ActionType, action.Group, action.Name))
			}
		}
	}

	if logdiag.HasError(ctx) {
		return nil
	}

	err = g.DetectCycle()
	if err != nil {
		return diag.FromErr(err)
	}

	client := b.WorkspaceClient()

	result := g.Run(defaultParallelism, func(node nodeKey) bool {
		// TODO: if a given node fails, all downstream nodes should not be run. We should report those nodes.
		// TODO: ensure that config for this node is fully resolved at this point.

		settings, ok := SupportedResources[node.Group]
		if !ok {
			return false
		}

		actionType := plannedActionsMap[node]

		// The way plan currently works, is that it does not add resources with Noop action, turning them into Unset.
		// So we skip both, although at this point we will not see Noop here.
		if actionType == deployplan.ActionTypeUnset || actionType == deployplan.ActionTypeNoop {
			return true
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
				return false
			}
			return true
		}

		config, ok := b.GetResourceConfig(node.Group, node.Name)
		if !ok {
			logdiag.LogError(ctx, fmt.Errorf("internal error when reading config for %s.%s", node.Group, node.Name))
			return false
		}

		// Fetch the references to ensure all are resolved
		myReferences, err := extractReferences(b.Config.Value(), node)
		if err != nil {
			logdiag.LogError(ctx, err)

			return false
		}

		// At this point it's an error to have unresolved deps
		if len(myReferences) > 0 {
			// TODO: include the deps themselves in the message
			logdiag.LogError(ctx, fmt.Errorf("cannot deploy %s.%s due to unresolved deps", node.Group, node.Name))
			return false
		}

		// TODO: redo plan to downgrade planned action if possible (?)

		err = d.Deploy(ctx, config, actionType)
		if err != nil {
			logdiag.LogError(ctx, err)
			return false
		}

		// Update resources.id after successful deploy so that future ${resources...id} refs are replaced
		if isReferenced[node] {
			err = resolveIDReference(ctx, b, node.Group, node.Name)
			if err != nil {
				logdiag.LogError(ctx, fmt.Errorf("failed to replace ref to resources.%s.%s.id: %w", node.Group, node.Name, err))
				return false
			}
		}

		return true
	})

	// This must run even if deploy failed:
	err = b.ResourceDatabase.Finalize()
	if err != nil {
		logdiag.LogError(ctx, err)
	}

	logStats(ctx, result, plannedActionsMap)
	return nil
}

func logStats(ctx context.Context, result dagrun.RunResult[nodeKey], plannedActionsMap map[nodeKey]deployplan.ActionType) {
	deleteKeys(result.Successful, plannedActionsMap)
	deleteKeys(result.Failed, plannedActionsMap)
	deleteKeys(result.NotRun, plannedActionsMap)

	total := len(result.Successful) + len(result.Failed) + len(result.NotRun)

	if len(result.Failed) > 0 {
		names := strings.Join(toStrings(result.Failed), ", ")
		logdiag.LogError(ctx, fmt.Errorf("%d out of %d resources failed to deploy: %s", len(result.Failed), total, names))
	}

	if len(result.NotRun) > 0 {
		names := strings.Join(toStrings(result.NotRun), ", ")
		logdiag.LogError(ctx, fmt.Errorf("%d out of %d resources not deployed because of failed dependency: %s", len(result.NotRun), total, names))
	}

	if len(plannedActionsMap) > 0 {
		names := strings.Join(toStrings(slices.Collect(maps.Keys(plannedActionsMap))), ", ")
		logdiag.LogError(ctx, fmt.Errorf("%d resources present in plan but not processed: %s", len(plannedActionsMap), names))
	}
}

func toStrings(nodes []nodeKey) []string {
	result := make([]string, len(nodes))
	for i, node := range nodes {
		result[i] = node.String()
	}
	return result
}

func checkPresenceAndDelete(ctx context.Context, nodes []nodeKey, plannedActionsMap map[nodeKey]deployplan.ActionType) {
	// XXX swap arguments
	for _, node := range nodes {
		_, exists := plannedActionsMap[node]
		if !exists {
			logdiag.LogError(ctx, fmt.Errorf("internal error: unexpected deployment: %s.%s", node.Group, node.Name))
		}
		delete(plannedActionsMap, node)
	}
}

func deleteKeys(nodes []nodeKey, plannedActionsMap map[nodeKey]deployplan.ActionType) {
	for _, node := range nodes {
		delete(plannedActionsMap, node)
	}
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
