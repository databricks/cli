package direct

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/direct/dresources"
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
	b.Adapters, err = dresources.InitAll(client)
	return err
}

func (b *DeploymentBundle) CalculatePlanForDeploy(ctx context.Context, client *databricks.WorkspaceClient, configRoot *config.Root) (*deployplan.Plan, error) {
	b.StateDB.AssertOpened()
	err := b.Init(client)
	if err != nil {
		return nil, err
	}

	// TODO: make this --local option
	localOnly := false

	// TODO: get this from --refresh / --norefresh option (default refresh for prod, norefresh for dev mode)
	refresh := true

	b.Graph, err = makeResourceGraph(ctx, configRoot.Value())
	if err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}

	err = b.Graph.DetectCycle()
	if err != nil {
		return nil, err
	}

	plan := deployplan.Plan{
		Plan: make(map[string]deployplan.PlanEntry),
	}

	// We're processing resources in DAG order, because we're trying to get rid of all references like $resources.jobs.foo.id
	// if jobs.foo is not going to be (re)created. This means by the time we get to resource depending on $resources.jobs.foo.id
	// we might have already got rid of this reference, thus potentially downgrading actionType
	//
	// parallelism is set to 1, so there is no multi-threaded access there. TODO: increase parallism
	b.Graph.Run(1, func(node string, failedDependency *string) bool {
		group, key := deployplan.ParseResourceKey(node)
		if group == "" {
			logdiag.LogError(ctx, fmt.Errorf("internal error: bad node key: %s", node))
			return false
		}
		errorPrefix := "cannot plan " + node

		if failedDependency != nil {
			logdiag.LogError(ctx, fmt.Errorf("%s: dependency failed: %s", errorPrefix, *failedDependency))
			return false
		}

		adapter, ok := b.Adapters[group]
		if !ok {
			logdiag.LogError(ctx, fmt.Errorf("%s: resource type not supported on direct backend, available: %s", errorPrefix, strings.Join(utils.SortedKeys(b.Adapters), ", ")))
			return false
		}

		config, ok := configRoot.GetResourceConfig(group, key)
		if !ok {
			logdiag.LogError(ctx, fmt.Errorf("%s: internal error: cannot read config", errorPrefix))
			return false
		}

		d := &DeploymentUnit{
			ResourceKey: node,
			Adapter:     adapter,
		}

		// This currently does not do API calls, so we can run this sequentially. Once we have remote diffs, we need to run in threadpool.
		actionType, err := d.Plan(ctx, client, &b.StateDB, config, localOnly, refresh)
		if err != nil {
			logdiag.LogError(ctx, err)
			return false
		}

		hasDelayedResolutions := false

		for _, reference := range b.Graph.OutgoingLabels(node) {
			value, err := d.ResolveReferenceLocalOrRemote(ctx, &b.StateDB, reference, actionType, config)
			if errors.Is(err, ErrDelayed) {
				hasDelayedResolutions = true
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

		if actionType == deployplan.ActionTypeNoop {
			if hasDelayedResolutions {
				logdiag.LogError(ctx, fmt.Errorf("%s: internal error, action noop must not have delayed resolutions", errorPrefix))
				return false
			}
			return true
		}

		plan.Plan[node] = deployplan.PlanEntry{Action: actionType.StringFull()}
		return true
	})

	if logdiag.HasError(ctx) {
		return nil, errors.New("planning failed")
	}

	state := b.StateDB.ExportState(ctx)

	// Remained in state are resources that no longer present in the config
	for _, group := range utils.SortedKeys(state) {
		_, ok := b.Adapters[group]

		if !ok {
			log.Warnf(ctx, "%s: resource type not supported on direct backend", group)
			continue
		}
		for _, key := range utils.SortedKeys(state[group]) {
			n := "resources." + group + "." + key
			if b.Graph.HasNode(n) {
				continue
			}
			b.Graph.AddNode(n)
			plan.Plan[n] = deployplan.PlanEntry{Action: deployplan.ActionTypeDelete.StringFull()}
		}
	}

	return &plan, nil
}

func (b *DeploymentBundle) CalculatePlanForDestroy(ctx context.Context, client *databricks.WorkspaceClient) (*deployplan.Plan, error) {
	b.StateDB.AssertOpened()

	err := b.Init(client)
	if err != nil {
		return nil, err
	}

	b.Graph = dagrun.NewGraph()
	plan := deployplan.Plan{Plan: make(map[string]deployplan.PlanEntry)}

	state := b.StateDB.ExportState(ctx)
	for group, groupData := range state {
		_, ok := b.Adapters[group]
		if !ok {
			logdiag.LogError(ctx, fmt.Errorf("cannot destroy %s: resource type not supported on direct backend", group))
			continue
		}
		for key := range groupData {
			n := "resources." + group + "." + key
			b.Graph.AddNode(n)
			plan.Plan[n] = deployplan.PlanEntry{Action: deployplan.ActionTypeDelete.StringFull()}
		}
	}

	return &plan, nil
}
