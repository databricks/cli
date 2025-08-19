package terranova

import (
	"context"
	"errors"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/terranova/tnstate"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/libs/structdiff"
	"github.com/databricks/cli/libs/utils"
	"github.com/databricks/databricks-sdk-go"
)

type Planner struct {
	client       *databricks.WorkspaceClient
	db           *tnstate.TerranovaState
	group        string
	resourceName string
	settings     ResourceSettings
}

func (d *Planner) Plan(ctx context.Context, inputConfig any) (deployplan.ActionType, error) {
	result, err := d.plan(ctx, inputConfig)
	if err != nil {
		return deployplan.ActionTypeNoop, fmt.Errorf("planning: %s.%s: %w", d.group, d.resourceName, err)
	}
	return result, err
}

func (d *Planner) plan(_ context.Context, inputConfig any) (deployplan.ActionType, error) {
	entry, hasEntry := d.db.GetResourceEntry(d.group, d.resourceName)

	resource, cfgType, err := New(d.client, d.group, d.resourceName, inputConfig)
	if err != nil {
		return "", err
	}

	config := resource.Config()

	if !hasEntry {
		return deployplan.ActionTypeCreate, nil
	}

	oldID := entry.ID
	if oldID == "" {
		return "", errors.New("invalid state: empty id")
	}

	savedState, err := typeConvert(cfgType, entry.State)
	if err != nil {
		return "", fmt.Errorf("interpreting state: %w", err)
	}

	// Note, currently we're diffing static structs, not dynamic value.
	// This means for fields that contain references like ${resources.group.foo.id} we do one of the following:
	// for strings: comparing unresolved string like "${resoures.group.foo.id}" with actual object id. As long as IDs do not have ${...} format we're good.
	// for integers: compare 0 with actual object ID. As long as real object IDs are never 0 we're good.
	// Once we add non-id fields or add per-field details to "bundle plan", we must read dynamic data and deal with references as first class citizen.
	// This means distinguishing between 0 that are actually object ids and 0 that are there because typed struct integer cannot contain ${...} string.
	return calcDiff(d.settings, resource, savedState, config)
}

func CalculateDeployActions(ctx context.Context, b *bundle.Bundle) ([]deployplan.Action, error) {
	if !b.DirectDeployment {
		panic("direct deployment required")
	}

	// final result: collected actions per node
	var actions []deployplan.Action

	err := b.OpenResourceDatabase(ctx)
	if err != nil {
		return nil, err
	}

	client := b.WorkspaceClient()

	g, isReferenced, err := makeResourceGraph(ctx, b)
	if err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}

	err = g.DetectCycle()
	if err != nil {
		return nil, err
	}

	state := b.ResourceDatabase.ExportState(ctx)

	// Remained in state are resources that no longer present in the config
	for _, group := range utils.SortedKeys(state) {
		groupData := state[group]
		for _, key := range utils.SortedKeys(groupData) {
			if g.HasNode(nodeKey{group, key}) {
				// alive resources are processed by makeResourceGraph
				continue
			}
			actions = append(actions, deployplan.Action{
				Group:      group,
				Name:       key,
				ActionType: deployplan.ActionTypeDelete,
			})
		}
	}

	// We're processing resources in DAG order, because we're trying to get rid of all references like $resources.jobs.foo.id
	// if jobs.foo is not going to be (re)created. This means by the time we get to resource depending on $resources.jobs.foo.id
	// we might have already got rid of this reference, thus potentially downgrading actionType

	// parallelism is set to 1, so there is no multi-threaded access there.
	g.Run(1, func(node nodeKey, failedDependency *nodeKey) bool {
		if failedDependency != nil {
			logdiag.LogError(ctx, fmt.Errorf("cannot plan %s.%s: dependency failed: %s", node.Group, node.Name, failedDependency.String()))
			return false
		}
		settings, ok := SupportedResources[node.Group]
		if !ok {
			logdiag.LogError(ctx, fmt.Errorf("resource not supported on direct backend: %s", node.Group))
			// TODO: return an error so that the whole process is aborted.
			return false
		}

		pl := Planner{
			client:       client,
			db:           &b.ResourceDatabase,
			group:        node.Group,
			resourceName: node.Name,
			settings:     settings,
		}

		config, ok := b.GetResourceConfig(pl.group, pl.resourceName)
		if !ok {
			logdiag.LogError(ctx, fmt.Errorf("internal error: cannot get config for %s.%s", pl.group, pl.resourceName))
			return false
			// TODO: return an error so that the whole process is aborted.
		}

		// This currently does not do API calls, so we can run this sequentially. Once we have remote diffs, we need to run a in threadpool.
		actionType, err := pl.Plan(ctx, config)
		if err != nil {
			logdiag.LogError(ctx, err)
			// TODO: return error to abort this branch
			return false
		}

		if isReferenced[node] && actionType.KeepsID() {
			err = resolveIDReference(ctx, b, pl.group, pl.resourceName)
			if err != nil {
				logdiag.LogError(ctx, fmt.Errorf("failed to replace ref to resources.%s.%s.id: %w", pl.group, pl.resourceName, err))
				return false
			}
		}

		if actionType == deployplan.ActionTypeNoop {
			return true
		}

		actions = append(actions, deployplan.Action{
			Group:      node.Group,
			Name:       node.Name,
			ActionType: actionType,
		})
		return true
	})

	if logdiag.HasError(ctx) {
		return nil, errors.New("planning failed")
	}

	return actions, nil
}

func CalculateDestroyActions(ctx context.Context, b *bundle.Bundle) ([]deployplan.Action, error) {
	if !b.DirectDeployment {
		panic("direct deployment required")
	}

	err := b.OpenResourceDatabase(ctx)
	if err != nil {
		return nil, err
	}

	db := &b.ResourceDatabase
	db.AssertOpened()
	var actions []deployplan.Action

	for _, group := range utils.SortedKeys(db.Data.Resources) {
		groupData := db.Data.Resources[group]
		for _, name := range utils.SortedKeys(groupData) {
			actions = append(actions, deployplan.Action{
				Group:      group,
				Name:       name,
				ActionType: deployplan.ActionTypeDelete,
			})
		}
	}

	return actions, nil
}

func calcDiff(settings ResourceSettings, resource IResource, savedState, config any) (deployplan.ActionType, error) {
	localDiff, err := structdiff.GetStructDiff(savedState, config)
	if err != nil {
		return "", err
	}

	if len(localDiff) == 0 {
		return deployplan.ActionTypeNoop, nil
	}

	if settings.MustRecreate(localDiff) {
		return deployplan.ActionTypeRecreate, nil
	}

	customClassify, hasCustomClassify := resource.(IResourceCustomClassify)

	if hasCustomClassify {
		_, hasUpdateWithID := resource.(IResourceUpdatesID)

		result := customClassify.ClassifyChanges(localDiff)
		if result == deployplan.ActionTypeRecreate && !settings.RecreateAllowed {
			return "", errors.New("internal error: unexpected plan='recreate'")
		}

		if result == deployplan.ActionTypeUpdateWithID && !hasUpdateWithID {
			return "", errors.New("internal error: unexpected plan='update_with_id'")
		}

		return result, nil
	}

	return deployplan.ActionTypeUpdate, nil
}
