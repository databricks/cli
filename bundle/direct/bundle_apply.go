package direct

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/databricks-sdk-go"
)

func (b *DeploymentBundle) Apply(ctx context.Context, client *databricks.WorkspaceClient, configRoot *config.Root, plan *deployplan.Plan) {
	if b.Graph == nil {
		panic("Planning is not done")
	}

	if len(plan.Plan) == 0 {
		// Avoid creating state file if nothing to deploy
		return
	}

	b.StateDB.AssertOpened()

	b.Graph.Run(defaultParallelism, func(node string, failedDependency *string) bool {
		entry, ok := plan.Plan[node]
		if !ok {
			// Nothing to do for this node
			return true
		}

		group, key := deployplan.ParseResourceKey(node)
		if group == "" {
			logdiag.LogError(ctx, fmt.Errorf("internal error: bad node key: %s", node))
			return false
		}

		at := deployplan.ActionTypeFromString(entry.Action)
		if at == deployplan.ActionTypeUnset {
			logdiag.LogError(ctx, fmt.Errorf("unknown action %q for %s", entry.Action, node))
			return false
		}
		d := &DeploymentUnit{
			ResourceKey: node,
			Adapter:     b.Adapters[group],
		}
		errorPrefix := fmt.Sprintf("cannot %s %s", entry.Action, node)

		// If a dependency failed, report and skip execution for this node by returning false
		if failedDependency != nil {
			logdiag.LogError(ctx, fmt.Errorf("%s: dependency failed: %s", errorPrefix, *failedDependency))
			return false
		}

		if at == deployplan.ActionTypeDelete {
			err := d.Destroy(ctx, &b.StateDB)
			if err != nil {
				logdiag.LogError(ctx, fmt.Errorf("%s: %w", errorPrefix, err))
				return false
			}
			return true
		}

		// Fetch the references to ensure all are resolved
		myReferences, err := extractReferences(configRoot.Value(), node)
		if err != nil {
			logdiag.LogError(ctx, fmt.Errorf("%s: reading references from config: %w", errorPrefix, err))
			return false
		}

		// At this point it's an error to have unresolved deps
		if len(myReferences) > 0 {
			// TODO: include the deps themselves in the message
			logdiag.LogError(ctx, fmt.Errorf("%s: unresolved deps", errorPrefix))
			return false
		}

		config, ok := configRoot.GetResourceConfig(group, key)
		if !ok {
			logdiag.LogError(ctx, fmt.Errorf("%s: internal error when reading config", errorPrefix))
			return false
		}

		// TODO: redo calcDiff to downgrade planned action if possible (?)

		err = d.Deploy(ctx, &b.StateDB, config, at)
		if err != nil {
			logdiag.LogError(ctx, fmt.Errorf("%s: %w", errorPrefix, err))
			return false
		}

		// We now process references of the form "resources.<group>.<key>.<field...>" and refers
		// for the resource that was just deployed. We first look up those references (ResolveReferenceRemote)
		// and the replace them across the whole bundle (replaceReferenceWithValue).
		// Note, we've already replaced what we could in plan phase:
		// - "id" for cases where id cannot change;
		// - "field" for cases where field is part of the config.
		// Now we're focussing on the remaining cases:
		// - "id" for cases where id could have changed;
		// - "field" for cases where field is part of the remote state.
		for _, reference := range b.Graph.OutgoingLabels(node) {
			value, err := d.ResolveReferenceRemote(ctx, &b.StateDB, reference)
			if err != nil {
				logdiag.LogError(ctx, fmt.Errorf("failed to resolve reference %q for %s after deployment: %w", reference, node, err))
				return false
			}

			err = replaceReferenceWithValue(ctx, configRoot, reference, value)
			if err != nil {
				logdiag.LogError(ctx, fmt.Errorf("failed to replace reference %q with value %v for %s: %w", reference, value, node, err))
				return false
			}
		}

		return true
	})

	// This must run even if deploy failed:
	err := b.StateDB.Finalize()
	if err != nil {
		logdiag.LogError(ctx, err)
	}
}
