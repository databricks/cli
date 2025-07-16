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
	"github.com/databricks/cli/bundle/terranova/tnresources"
	"github.com/databricks/cli/bundle/terranova/tnstate"
	"github.com/databricks/cli/libs/dagrun"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/libs/structdiff"
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

	g := dagrun.NewGraph[deployplan.Action]()

	for _, action := range b.Plan.Actions {
		g.AddNode(action)
		// TODO: Scan v for references and use g.AddDirectedEdge to add dependency
	}

	client := b.WorkspaceClient()

	err := g.Run(defaultParallelism, func(node deployplan.Action) {
		// TODO: if a given node fails, all downstream nodes should not be run. We should report those nodes.
		// TODO: ensure that config for this node is fully resolved at this point.

		if node.ActionType == deployplan.ActionTypeUnset {
			logdiag.LogError(ctx, fmt.Errorf("internal error, no action set %#v", node))
			return
		}

		var config any

		if node.ActionType != deployplan.ActionTypeDelete {
			var ok bool
			config, ok = b.GetResourceConfig(node.Group, node.Name)
			if !ok {
				logdiag.LogError(ctx, fmt.Errorf("internal error: cannot get config for group=%v name=%v", node.Group, node.Name))
				return
			}
		}

		d := Deployer{
			client:       client,
			db:           &b.ResourceDatabase,
			group:        node.Group,
			resourceName: node.Name,
		}

		// TODO: pass node.ActionType and respect it. Do not change Update to Recreate or Delete for example.
		err := d.Deploy(ctx, config, node.ActionType)
		if err != nil {
			logdiag.LogError(ctx, err)
			return
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
}

func (d *Deployer) Deploy(ctx context.Context, inputConfig any, actionType deployplan.ActionType) error {
	if actionType == deployplan.ActionTypeDelete {
		err := d.destroy(ctx, inputConfig)
		if err != nil {
			return fmt.Errorf("destroying %s.%s: %w", d.group, d.resourceName, err)
		}
		return nil
	}

	err := d.deploy(ctx, inputConfig, actionType)
	if err != nil {
		return fmt.Errorf("deploying %s.%s: %w", d.group, d.resourceName, err)
	}
	return nil
}

func (d *Deployer) destroy(ctx context.Context, inputConfig any) error {
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
	entry, hasEntry := d.db.GetResourceEntry(d.group, d.resourceName)

	resource, cfgType, err := tnresources.New(d.client, d.group, d.resourceName, inputConfig)
	if err != nil {
		return err
	}

	config := resource.Config()

	if !hasEntry {
		return d.Create(ctx, resource, config)
	}

	oldID := entry.ID
	if oldID == "" {
		return errors.New("invalid state: empty id")
	}

	savedState, err := typeConvert(cfgType, entry.State)
	if err != nil {
		return fmt.Errorf("interpreting state: %w", err)
	}

	localDiff, err := structdiff.GetStructDiff(savedState, config)
	if err != nil {
		return fmt.Errorf("comparing state and config: %w", err)
	}

	localDiffType := deployplan.ActionTypeNoop

	if len(localDiff) > 0 {
		localDiffType = resource.ClassifyChanges(localDiff)
	}

	if localDiffType == deployplan.ActionTypeRecreate {
		return d.Recreate(ctx, resource, oldID, config)
	}

	if localDiffType == deployplan.ActionTypeUpdate {
		return d.Update(ctx, resource, oldID, config)
	}

	// localDiffType is either None or Partial: we should proceed to fetching remote state and calculate local+remote diff

	log.Debugf(ctx, "Unchanged %s.%s id=%#v", d.group, d.resourceName, oldID)
	return nil
}

func (d *Deployer) Create(ctx context.Context, resource tnresources.IResource, config any) error {
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

func (d *Deployer) Recreate(ctx context.Context, oldResource tnresources.IResource, oldID string, config any) error {
	err := tnresources.DeleteResource(ctx, d.client, d.group, oldID)
	if err != nil {
		return fmt.Errorf("deleting old id=%s: %w", oldID, err)
	}

	err = d.db.SaveState(d.group, d.resourceName, "", nil)
	if err != nil {
		return fmt.Errorf("deleting state: %w", err)
	}

	newResource, _, err := tnresources.New(d.client, d.group, d.resourceName, config)
	if err != nil {
		return fmt.Errorf("initializing: %w", err)
	}

	newID, err := newResource.DoCreate(ctx)
	if err != nil {
		return fmt.Errorf("re-creating: %w", err)
	}

	log.Warnf(ctx, "Re-created %s.%s id=%#v (previously %#v)", d.group, d.resourceName, newID, oldID)
	err = d.db.SaveState(d.group, d.resourceName, newID, config)
	if err != nil {
		return fmt.Errorf("saving state for id=%s: %w", newID, err)
	}

	err = newResource.WaitAfterCreate(ctx)
	if err != nil {
		return fmt.Errorf("waiting after re-creating id=%s: %w", newID, err)
	}

	return nil
}

func (d *Deployer) Update(ctx context.Context, resource tnresources.IResource, oldID string, config any) error {
	newID, err := resource.DoUpdate(ctx, oldID)
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
	err := tnresources.DeleteResource(ctx, d.client, d.group, oldID)
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
