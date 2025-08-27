package terranova

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/libs/dagrun"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/libs/utils"
	"github.com/databricks/databricks-sdk-go"
)

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

	b.Resources = make(map[deployplan.ResourceNode]Deployer)

	// We're processing resources in DAG order, because we're trying to get rid of all references like $resources.jobs.foo.id
	// if jobs.foo is not going to be (re)created. This means by the time we get to resource depending on $resources.jobs.foo.id
	// we might have already got rid of this reference, thus potentially downgrading actionType

	// parallelism is set to 1, so there is no multi-threaded access there. TODO: increase parallism, protect b.Resources
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

		config, ok := configRoot.GetResourceConfig(node.Group, node.Key)
		if !ok {
			logdiag.LogError(ctx, fmt.Errorf("%s: internal error: cannot read config", errorPrefix))
			return false
		}

		d := Deployer{
			Group: node.Group,
			Key:   node.Key,
		}

		// This currently does not do API calls, so we can run this sequentially. Once we have remote diffs, we need to run a in threadpool.
		var err error
		d.ActionType, err = d.Plan(ctx, client, &b.StateDB, settings, config, localOnly, refresh)
		if err != nil {
			logdiag.LogError(ctx, err)
			return false
		}

		for _, reference := range b.Graph.OutgoingLabels(node) {
			value, err := d.ResolveReferenceLocalOrRemote(ctx, &b.StateDB, reference, d.ActionType, config, settings.RemoteType)
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
			b.Resources[node] = d
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
			b.Resources[n] = Deployer{
				Group:      group,
				Key:        key,
				ActionType: deployplan.ActionTypeDelete,
			}
			b.Graph.AddNode(n)
		}
	}

	return nil
}

func (b *BundleDeployer) CalculatePlanForDestroy(ctx context.Context) error {
	b.StateDB.AssertOpened()

	b.Resources = make(map[deployplan.ResourceNode]Deployer)
	b.Graph = dagrun.NewGraph[deployplan.ResourceNode]()

	for group, groupData := range b.StateDB.Data.Resources {
		for key := range groupData {
			n := deployplan.ResourceNode{Group: group, Key: key}
			b.Resources[n] = Deployer{
				Group:      group,
				Key:        key,
				ActionType: deployplan.ActionTypeDelete,
			}
			b.Graph.AddNode(n)
		}
	}

	return nil
}

func (b *BundleDeployer) GetActions(ctx context.Context) []deployplan.Action {
	actions := make([]deployplan.Action, 0, len(b.Resources))
	for node, deployer := range b.Resources {
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
