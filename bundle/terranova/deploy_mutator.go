package terranova

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/terranova/tnresources"
	"github.com/databricks/cli/bundle/terranova/tnstate"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/dagrun"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/structdiff"
	"github.com/databricks/databricks-sdk-go"
)

const maxPoolSize = 10

type terranovaDeployMutator struct{}

func TerranovaDeploy() bundle.Mutator {
	return &terranovaDeployMutator{}
}

func (m *terranovaDeployMutator) Name() string {
	return "TerranovaDeploy"
}

func (m *terranovaDeployMutator) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	var diags diag.SafeDiagnostics

	client := b.WorkspaceClient()

	g := dagrun.NewGraph[tnstate.ResourceNode]()

	_, err := dyn.MapByPattern(
		b.Config.Value(),
		dyn.NewPattern(dyn.Key("resources"), dyn.AnyKey(), dyn.AnyKey()),
		func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
			section := p[1].Key()
			name := p[2].Key()
			// log.Warnf(ctx, "Adding node=%s", node)
			g.AddNode(tnstate.ResourceNode{
				Section: section,
				Name:    name,
			})

			// TODO: Scan v for references and use g.AddDirectedEdge to add dependency
			return v, nil
		},
	)

	countDeployed := 0

	err = g.Run(maxPoolSize, func(node tnstate.ResourceNode) {
		// TODO: if a given node fails, all downstream nodes should not be run. We should report those nodes.
		// TODO: ensure that config for this node is fully resolved at this point.

		config, ok := b.GetResourceConfig(node.Section, node.Name)
		if !ok {
			diags.AppendErrorf("internal error: cannot get config for %s", node)
			return
		}

		d := Deployer{
			client:       client,
			db:           &b.ResourceDatabase,
			section:      node.Section,
			resourceName: node.Name,
		}

		err = d.Deploy(ctx, config)
		if err != nil {
			diags.AppendError(err)
			return
		}

		countDeployed = countDeployed + 1
	})
	if err != nil {
		diags.AppendError(err)
	}

	// Not uploading at the moment, just logging to match the output.
	if countDeployed > 0 {
		cmdio.LogString(ctx, "Updating deployment state...")
	}

	err = b.ResourceDatabase.Finalize()
	if err != nil {
		diags.AppendError(err)
	}

	return diags.Diags
}

type Deployer struct {
	client       *databricks.WorkspaceClient
	db           *tnstate.TerranovaState
	section      string
	resourceName string
}

func (d *Deployer) Deploy(ctx context.Context, inputConfig any) error {
	err := d.deploy(ctx, inputConfig)
	if err != nil {
		return fmt.Errorf("deploying %s.%s: %w", d.section, d.resourceName, err)
	}
	return nil
}

func (d *Deployer) deploy(ctx context.Context, inputConfig any) error {
	oldID, err := d.db.GetResourceID(d.section, d.resourceName)
	if err != nil {
		return err
	}

	resource, err := tnresources.New(d.client, d.section, d.resourceName, inputConfig)
	if err != nil {
		return err
	}

	config := resource.Config()

	// Presence of id in the state file implies that the resource was created by us

	if oldID == "" {
		newID, err := resource.DoCreate(ctx)
		if err != nil {
			return err
		}

		log.Infof(ctx, "Created %s.%s id=%#v", d.section, d.resourceName, newID)

		err = d.db.SaveState(d.section, d.resourceName, newID, config)
		if err != nil {
			return err
		}

		return resource.WaitAfterCreate(ctx)
	}

	savedState, err := d.db.GetSavedState(d.section, d.resourceName, resource.GetType())
	if err != nil {
		return fmt.Errorf("reading state: %w", err)
	}

	localDiff, err := structdiff.GetStructDiff(savedState, config)
	if err != nil {
		return fmt.Errorf("state error: %w", err)
	}

	localDiffType := tnresources.ChangeTypeNone
	if len(localDiff) > 0 {
		localDiffType = resource.ClassifyChanges(localDiff)
	}

	if localDiffType.IsRecreate() {
		return d.Recreate(ctx, resource, oldID, config)
	}

	if localDiffType.IsUpdate() {
		return d.Update(ctx, resource, oldID, config)
	}

	// localDiffType is either None or Partial: we should proceed to fetching remote state and calculate local+remote diff

	log.Debugf(ctx, "Unchanged %s.%s id=%#v", d.section, d.resourceName, oldID)
	return nil
}

func (d *Deployer) Recreate(ctx context.Context, oldResource tnresources.IResource, oldID string, config any) error {
	err := oldResource.DoDelete(ctx, oldID)
	if err != nil {
		return fmt.Errorf("deleting old id=%s: %w", oldID, err)
	}

	err = d.db.SaveState(d.section, d.resourceName, "", nil)
	if err != nil {
		return fmt.Errorf("deleting state: %w", err)
	}

	newResource, err := tnresources.New(d.client, d.section, d.resourceName, config)
	if err != nil {
		return fmt.Errorf("initializing: %w", err)
	}

	newID, err := newResource.DoCreate(ctx)
	if err != nil {
		return fmt.Errorf("re-creating: %w", err)
	}

	log.Warnf(ctx, "Re-created %s.%s id=%#v (previously %#v)", d.section, d.resourceName, newID, oldID)
	err = d.db.SaveState(d.section, d.resourceName, newID, config)
	if err != nil {
		return fmt.Errorf("saving state: %w", err)
	}

	return newResource.WaitAfterCreate(ctx)
}

func (d *Deployer) Update(ctx context.Context, resource tnresources.IResource, oldID string, config any) error {
	newID, err := resource.DoUpdate(ctx, oldID)
	if err != nil {
		return fmt.Errorf("updating id=%s: %w", oldID, err)
	}

	if oldID != newID {
		log.Infof(ctx, "Updated %s.%s id=%#v (previously %#v)", d.section, d.resourceName, newID, oldID)
	} else {
		log.Infof(ctx, "Updated %s.%s id=%#v", d.section, d.resourceName, newID)
	}

	err = d.db.SaveState(d.section, d.resourceName, newID, config)
	if err != nil {
		return fmt.Errorf("saving state id=%s: %w", oldID, err)
	}

	return resource.WaitAfterUpdate(ctx)
}
