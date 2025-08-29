package terranova

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"reflect"
	"slices"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/terranova/tnstate"
	"github.com/databricks/cli/libs/dagrun"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/dynvar"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/libs/structaccess"
	"github.com/databricks/cli/libs/utils"
	"github.com/databricks/databricks-sdk-go"
)

type BundleDeployer struct {
	StateDB        tnstate.TerranovaState
	Graph          *dagrun.Graph[deployplan.ResourceNode]
	PlannedActions map[deployplan.ResourceNode]deployplan.ActionType
}

func (b *BundleDeployer) OpenStateFile(statePath string) error {
	err := b.StateDB.Open(statePath)
	if err != nil {
		return fmt.Errorf("failed to read state from %s: %w", statePath, err)
	}
	return nil
}

func (b *BundleDeployer) CalculatePlanForDeploy(ctx context.Context, client *databricks.WorkspaceClient, configRoot *config.Root) error {
	var err error
	b.StateDB.AssertOpened()

	b.Graph, err = makeResourceGraph(ctx, configRoot.Value())
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
			db:           &b.StateDB,
			group:        node.Group,
			resourceName: node.Key,
			settings:     settings,
		}

		config, ok := configRoot.GetResourceConfig(pl.group, pl.resourceName)
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

		for _, reference := range b.Graph.OutgoingLabels(node) {
			path, ok := dynvar.PureReferenceToPath(reference)
			if !ok || len(path) <= 3 || path[0].Key() != "resources" || path[1].Key() != node.Group || path[2].Key() != node.Key {
				logdiag.LogError(ctx, fmt.Errorf("internal error: expected reference to resources.%s.%s, got %q", node.Group, node.Key, reference))
				return false
			}
			fieldPath := path[3:].String()
			if fieldPath == "id" {
				if actionType.KeepsID() {
					// Now that we know that ID of this node is not going to change, update it
					// everywhere to actual value to calculate more accurate and more conservative action for dependent nodes.
					err = resolveIDReference(ctx, pl.db, configRoot, pl.group, pl.resourceName)
					if err != nil {
						logdiag.LogError(ctx, fmt.Errorf("failed to replace ref to resources.%s.%s.id: %w", pl.group, pl.resourceName, err))
						return false
					}
				}
				continue
			}
			dynPath, err := dyn.NewPathFromString(fieldPath)
			if err != nil {
				logdiag.LogError(ctx, fmt.Errorf("cannot parse path %s: %w", fieldPath, err))
				return false
			}
			validationErr := structaccess.Validate(reflect.TypeOf(config), dynPath)
			if validationErr != nil {
				logdiag.LogError(ctx, fmt.Errorf("schema mismatch for %s: %w", reference, validationErr))
				return false
			}
			value, err := structaccess.Get(config, dynPath)
			if err != nil {
				logdiag.LogError(ctx, fmt.Errorf("field not set %s: %w", reference, err))
				return false
			}
			err = resolveFieldReference(ctx, configRoot, path, value)
			if err != nil {
				logdiag.LogError(ctx, fmt.Errorf("failed to replace ref to %s: %w", reference, err))
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

	state := b.StateDB.ExportState(ctx)

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
			b.Graph.AddNode(n)
		}
	}

	return nil
}

func (b *BundleDeployer) CalculatePlanForDestroy(ctx context.Context) error {
	b.StateDB.AssertOpened()

	b.PlannedActions = make(map[deployplan.ResourceNode]deployplan.ActionType)
	b.Graph = dagrun.NewGraph[deployplan.ResourceNode]()

	for group, groupData := range b.StateDB.Data.Resources {
		for key := range groupData {
			node := deployplan.ResourceNode{Group: group, Key: key}
			b.PlannedActions[node] = deployplan.ActionTypeDelete
			b.Graph.AddNode(node)
		}
	}

	return nil
}

func (b *BundleDeployer) GetActions(ctx context.Context) []deployplan.Action {
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
