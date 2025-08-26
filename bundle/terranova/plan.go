package terranova

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/terranova/tnstate"
	"github.com/databricks/cli/libs/dagrun"
	"github.com/databricks/cli/libs/dyn/dynvar"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/libs/structaccess"
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

func GetDeployActions(ctx context.Context, b *bundle.Bundle) []deployplan.Action {
	actions := make([]deployplan.Action, 0, len(b.PlannedActions))
	for node, actionType := range b.PlannedActions {
		actions = append(actions, deployplan.Action{
			ResourceNode: node,
			ActionType:   actionType,
		})
	}

	// TODO: topological sort would be more informative; once we removed terraform

	slices.SortFunc(actions, func(x, y deployplan.Action) int {
		if c := cmp.Compare(x.Group, y.Group); c != 0 {
			return c
		}
		return cmp.Compare(x.Key, y.Key)
	})

	return actions
}

func CalculatePlanForDeploy(ctx context.Context, b *bundle.Bundle) error {
	if !b.DirectDeployment {
		panic("direct deployment required")
	}

	err := b.OpenResourceDatabase(ctx)
	if err != nil {
		return err
	}

	client := b.WorkspaceClient()

	b.Graph, err = makeResourceGraph(ctx, b)
	if err != nil {
		return fmt.Errorf("reading config: %w", err)
	}

	err = b.Graph.DetectCycle()
	if err != nil {
		return err
	}

	b.PlannedActions = make(map[deployplan.ResourceNode]deployplan.ActionType)

	// We're processing resources in DAG order, because we're trying to get rid of all references like $resources.jobs.foo.id
	// if jobs.foo is not going to be (re)created. This means by the time we get to resource depending on $resources.jobs.foo.id
	// we might have already got rid of this reference, thus potentially downgrading actionType

	// parallelism is set to 1, so there is no multi-threaded access there.
	b.Graph.Run(1, func(node deployplan.ResourceNode, failedDependency *deployplan.ResourceNode) bool {
		errorPrefix := fmt.Sprintf("cannot plan %s.%s", node.Group, node.Key)

		if failedDependency != nil {
			logdiag.LogError(ctx, fmt.Errorf("%s: dependency failed: %s", errorPrefix, failedDependency.String()))
			return false
		}
		settings, ok := SupportedResources[node.Group]
		if !ok {
			logdiag.LogError(ctx, fmt.Errorf("%s: resource type not supported on direct backend", errorPrefix))
			return false
		}

		pl := Planner{
			client:       client,
			db:           &b.ResourceDatabase,
			group:        node.Group,
			resourceName: node.Key,
			settings:     settings,
		}

		config, ok := b.GetResourceConfig(pl.group, pl.resourceName)
		if !ok {
			logdiag.LogError(ctx, fmt.Errorf("%s: internal error: cannot read config", errorPrefix))
			return false
		}

		// This currently does not do API calls, so we can run this sequentially. Once we have remote diffs, we need to run a in threadpool.
		actionType, err := pl.Plan(ctx, config)
		if err != nil {
			logdiag.LogError(ctx, err)
			return false
		}

		for _, label := range b.Graph.OutgoingLabels(node) {
			path, ok := dynvar.PureReferenceToPath(label)
			if !ok || len(path) <= 3 || path[0].Key() != "resources" || path[1].Key() != node.Group || path[2].Key() != node.Key {
				logdiag.LogError(ctx, fmt.Errorf("internal error: expected reference to resources.%s.%s, got %q", node.Group, node.Key, label))
				return false
			}
			fieldPath := path[3:].String()
			if fieldPath == "id" {
				if actionType.KeepsID() {
					// Now that we know that ID of this node is not going to change, update it
					// everywhere to actual value to calculate more accurate and more conservative action for dependent nodes.
					err = resolveIDReference(ctx, b, pl.group, pl.resourceName)
					if err != nil {
						logdiag.LogError(ctx, fmt.Errorf("failed to replace ref to resources.%s.%s.id: %w", pl.group, pl.resourceName, err))
						return false
					}
				}
				continue
			}
			value, err := structaccess.Get(config, fieldPath)
			if err != nil {
				logdiag.LogError(ctx, fmt.Errorf("cannot resolve %s: %w", label, err))
				return false
			}
			err = resolveFieldReference(ctx, b, pl.group, pl.resourceName, fieldPath, value)
			if err != nil {
				logdiag.LogError(ctx, fmt.Errorf("failed to replace ref to %s: %w", label, err))
				return false
			}
		}

		if actionType != deployplan.ActionTypeNoop {
			b.PlannedActions[node] = actionType
		}
		return true
	})

	if logdiag.HasError(ctx) {
		return errors.New("planning failed")
	}

	state := b.ResourceDatabase.ExportState(ctx)

	// Remained in state are resources that no longer present in the config
	for _, group := range utils.SortedKeys(state) {
		groupData := state[group]
		for _, key := range utils.SortedKeys(groupData) {
			n := deployplan.ResourceNode{Group: group, Key: key}
			if b.Graph.HasNode(n) {
				// action was already added by Run() above
				continue
			}
			b.PlannedActions[n] = deployplan.ActionTypeDelete
		}
	}

	return nil
}

func CalculatePlanForDestroy(ctx context.Context, b *bundle.Bundle) error {
	if !b.DirectDeployment {
		panic("direct deployment required")
	}

	err := b.OpenResourceDatabase(ctx)
	if err != nil {
		return err
	}

	db := &b.ResourceDatabase
	db.AssertOpened()

	b.PlannedActions = make(map[deployplan.ResourceNode]deployplan.ActionType)
	b.Graph = dagrun.NewGraph[deployplan.ResourceNode]()

	for group, groupData := range db.Data.Resources {
		for key := range groupData {
			node := deployplan.ResourceNode{Group: group, Key: key}
			b.PlannedActions[node] = deployplan.ActionTypeDelete
			b.Graph.AddNode(node)
		}
	}

	return nil
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
