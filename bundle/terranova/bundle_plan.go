package terranova

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/terranova/tnresources"
	"github.com/databricks/cli/libs/dagrun"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/libs/utils"
	"github.com/databricks/databricks-sdk-go"
)

func (b *DeploymentBundle) OpenStateFile(statePath string) error {
	err := b.StateDB.Open(statePath)
	if err != nil {
		return fmt.Errorf("failed to read state from %s: %w", statePath, err)
	}
	return nil
}

func (b *DeploymentBundle) Init(client *databricks.WorkspaceClient) error {
	if b.Adapters != nil {
		return nil
	}
	var err error
	b.Adapters, err = tnresources.InitAll(client)
	return err
}

func (b *DeploymentBundle) CalculatePlanForDeploy(ctx context.Context, client *databricks.WorkspaceClient, configRoot *config.Root) error {
	b.StateDB.AssertOpened()
	err := b.Init(client)
	if err != nil {
		return err
	}

	// TODO: make this --local option
	localOnly := false

	// TODO: get this from --refresh / --norefresh option (default refresh for prod, norefresh for dev mode)
	refresh := true

	b.Graph, err = makeResourceGraph(ctx, configRoot.Value())
	if err != nil {
		return fmt.Errorf("reading config: %w", err)
	}

	err = b.Graph.DetectCycle()
	if err != nil {
		return err
	}

	b.DeploymentUnits = make(map[deployplan.ResourceNode]*DeploymentUnit)

	// We're processing resources in DAG order, because we're trying to get rid of all references like $resources.jobs.foo.id
	// if jobs.foo is not going to be (re)created. This means by the time we get to resource depending on $resources.jobs.foo.id
	// we might have already got rid of this reference, thus potentially downgrading actionType

	// parallelism is set to 1, so there is no multi-threaded access there. TODO: increase parallism, protect b.DeploymentUnits
	b.Graph.Run(1, func(node deployplan.ResourceNode, failedDependency *deployplan.ResourceNode) bool {
		errorPrefix := fmt.Sprintf("cannot plan %s.%s", node.Group, node.Key)

		if failedDependency != nil {
			logdiag.LogError(ctx, fmt.Errorf("%s: dependency failed: %s", errorPrefix, failedDependency.String()))
			return false
		}

		adapter, ok := b.Adapters[node.Group]
		if !ok {
			logdiag.LogError(ctx, fmt.Errorf("%s: resource type not supported on direct backend, available: %s", errorPrefix, strings.Join(utils.SortedKeys(b.Adapters), ", ")))
			return false
		}

		config, ok := configRoot.GetResourceConfig(node.Group, node.Key)
		if !ok {
			logdiag.LogError(ctx, fmt.Errorf("%s: internal error: cannot read config", errorPrefix))
			return false
		}

		d := &DeploymentUnit{
			Group:   node.Group,
			Key:     node.Key,
			Adapter: adapter,
		}

		// This currently does not do API calls, so we can run this sequentially. Once we have remote diffs, we need to run a in threadpool.
		var err error
		d.ActionType, err = d.Plan(ctx, client, &b.StateDB, config, localOnly, refresh)
		if err != nil {
			logdiag.LogError(ctx, err)
			return false
		}

		for _, reference := range b.Graph.OutgoingLabels(node) {
			value, err := d.ResolveReferenceLocalOrRemote(ctx, &b.StateDB, reference, d.ActionType, config)
			if errors.Is(err, ErrDelayed) {
				continue
			}
			if err != nil {
				logdiag.LogError(ctx, fmt.Errorf("cannot resolve %q: %w", reference, err))
				return false
			}
			err = replaceReferenceWithValue(ctx, configRoot, reference, value)
			if err != nil {
				logdiag.LogError(ctx, fmt.Errorf("failed to replace %q with %v: %w", reference, value, err))
				return false
			}
		}

		if d.ActionType != deployplan.ActionTypeNoop {
			b.DeploymentUnits[node] = d
		}
		return true
	})

	if logdiag.HasError(ctx) {
		return errors.New("planning failed")
	}

	state := b.StateDB.ExportState(ctx)

	// Remained in state are resources that no longer present in the config
	for _, group := range utils.SortedKeys(state) {
		adapter, ok := b.Adapters[group]

		if !ok {
			log.Warnf(ctx, "%s: resource type not supported on direct backend", group)
			continue
		}

		groupData := state[group]
		for _, key := range utils.SortedKeys(groupData) {
			n := deployplan.ResourceNode{Group: group, Key: key}
			if b.Graph.HasNode(n) {
				// action was already added by Run() above
				continue
			}
			b.DeploymentUnits[n] = &DeploymentUnit{
				Group:      group,
				Key:        key,
				Adapter:    adapter,
				ActionType: deployplan.ActionTypeDelete,
			}
			b.Graph.AddNode(n)
		}
	}

	return nil
}

func (b *DeploymentBundle) CalculatePlanForDestroy(ctx context.Context, client *databricks.WorkspaceClient) error {
	b.StateDB.AssertOpened()

	err := b.Init(client)
	if err != nil {
		return err
	}

	b.DeploymentUnits = make(map[deployplan.ResourceNode]*DeploymentUnit)
	b.Graph = dagrun.NewGraph[deployplan.ResourceNode]()

	for group, groupData := range b.StateDB.Data.DeploymentUnits {
		adapter, ok := b.Adapters[group]
		if !ok {
			logdiag.LogError(ctx, fmt.Errorf("cannot destroy %s: resource type not supported on direct backend", group))
			continue
		}
		for key := range groupData {
			n := deployplan.ResourceNode{Group: group, Key: key}
			b.DeploymentUnits[n] = &DeploymentUnit{
				Group:      group,
				Key:        key,
				Adapter:    adapter,
				ActionType: deployplan.ActionTypeDelete,
			}
			b.Graph.AddNode(n)
		}
	}

	return nil
}

func (b *DeploymentBundle) GetActions(ctx context.Context) []deployplan.Action {
	actions := make([]deployplan.Action, 0, len(b.DeploymentUnits))
	for node, deployer := range b.DeploymentUnits {
		actions = append(actions, deployplan.Action{
			ResourceNode: node,
			ActionType:   deployer.ActionType,
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
