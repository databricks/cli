package direct

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/dynvar"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/libs/structs/structaccess"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/databricks-sdk-go"
)

func (b *DeploymentBundle) Apply(ctx context.Context, client *databricks.WorkspaceClient, configRoot *config.Root, plan *deployplan.Plan) {
	if plan == nil {
		panic("Planning is not done")
	}

	if len(plan.Plan) == 0 {
		// Avoid creating state file if nothing to deploy
		return
	}

	b.StateDB.AssertOpened()

	g, err := makeGraph(plan)
	if err != nil {
		logdiag.LogError(ctx, err)
		return
	}

	g.Run(defaultParallelism, func(resourceKey string, failedDependency *string) bool {
		entry := plan.LockEntry(resourceKey)
		defer plan.UnlockEntry(resourceKey)

		if entry == nil {
			logdiag.LogError(ctx, fmt.Errorf("internal error: node not in graph: %q", resourceKey))
			return false
		}

		action := entry.Action
		errorPrefix := fmt.Sprintf("cannot %s %s", action, resourceKey)

		at := deployplan.ActionTypeFromString(action)
		if at == deployplan.ActionTypeUnset {
			logdiag.LogError(ctx, fmt.Errorf("cannot deploy %s: unknown action %q", resourceKey, action))
			return false
		}

		// If a dependency failed, report and skip execution for this node by returning false
		if failedDependency != nil {
			if at != deployplan.ActionTypeSkip {
				logdiag.LogError(ctx, fmt.Errorf("%s: dependency failed: %s", errorPrefix, *failedDependency))
			}
			return false
		}

		adapter, err := b.getAdapterForKey(resourceKey)
		if adapter == nil {
			logdiag.LogError(ctx, fmt.Errorf("%s: internal error: cannot get adapter: %w", errorPrefix, err))
			return false
		}

		d := &DeploymentUnit{
			ResourceKey: resourceKey,
			Adapter:     adapter,
		}

		if at == deployplan.ActionTypeDelete {
			err = d.Destroy(ctx, &b.StateDB)
			if err != nil {
				logdiag.LogError(ctx, fmt.Errorf("%s: %w", errorPrefix, err))
				return false
			}
			return true
		}

		// We don't keep NewState around for 'skip' nodes

		if at != deployplan.ActionTypeSkip {
			for fieldPathStr, refString := range entry.NewState.Refs {
				refs, ok := dynvar.NewRef(dyn.V(refString))
				if !ok {
					logdiag.LogError(ctx, fmt.Errorf("%s: cannot parse %q", errorPrefix, refString))
					return false
				}

				for _, pathString := range refs.References() {
					ref := "${" + pathString + "}"
					targetPath, err := structpath.Parse(pathString)
					if !ok {
						logdiag.LogError(ctx, fmt.Errorf("%s: cannot parse reference %q: %w", errorPrefix, ref, err))
						return false
					}

					value, err := b.LookupReferenceRemote(ctx, targetPath)
					if err != nil {
						logdiag.LogError(ctx, fmt.Errorf("%s: cannot resolve %q: %w", errorPrefix, ref, err))
						return false
					}

					err = entry.NewState.ResolveRef(ref, value)
					if err != nil {
						logdiag.LogError(ctx, fmt.Errorf("%s: cannot update %s: with value of %q=%v: %w", errorPrefix, fieldPathStr, ref, value, err))
						return false
					}
				}
			}

			if len(entry.NewState.Refs) > 0 {
				logdiag.LogError(ctx, fmt.Errorf("%s: unresolved references: %s", errorPrefix, jsonDump(entry.NewState.Refs)))
				return false
			}

			// TODO: redo calcDiff to downgrade planned action if possible (?)

			err = d.Deploy(ctx, &b.StateDB, entry.NewState.Config, at)
			if err != nil {
				logdiag.LogError(ctx, fmt.Errorf("%s: %w", errorPrefix, err))
				return false
			}
		}

		// TODO: Note, we only really need remote state if there are remote references.
		//       The graph includes edges for both local and remote references. The local references are
		//       already resolved and should not play a role here.
		needRemoteState := len(g.Adj[resourceKey]) > 0
		if needRemoteState {
			entry, _ := b.StateDB.GetResourceEntry(d.ResourceKey)
			if entry.ID == "" {
				logdiag.LogError(ctx, fmt.Errorf("%s: internal error: missing entry in state after deploy", errorPrefix))
				return false
			}

			err = d.refreshRemoteState(ctx, entry.ID)
			if err != nil {
				logdiag.LogError(ctx, fmt.Errorf("%s: failed to read remote state: %w", errorPrefix, err))
				return false
			}
			b.RemoteStateCache.Store(resourceKey, d.RemoteState)
		}

		return true
	})

	// This must run even if deploy failed:
	err = b.StateDB.Finalize()
	if err != nil {
		logdiag.LogError(ctx, err)
	}
}

func (b *DeploymentBundle) LookupReferenceRemote(ctx context.Context, path *structpath.PathNode) (any, error) {
	targetResourceKey := path.Prefix(3).String()
	fieldPath := path.SkipPrefix(3)
	fieldPathS := fieldPath.String()

	targetEntry := b.Plan.LockEntry(targetResourceKey)
	defer b.Plan.UnlockEntry(targetResourceKey)

	if targetEntry == nil {
		return nil, fmt.Errorf("internal error: %s: missing entry in the plan", targetResourceKey)
	}

	targetAction := deployplan.ActionTypeFromString(targetEntry.Action)
	if targetAction == deployplan.ActionTypeUnset {
		return nil, fmt.Errorf("internal error: %s: missing action in the plan", targetResourceKey)
	}

	if fieldPathS == "id" {
		dbentry, hasEntry := b.StateDB.GetResourceEntry(targetResourceKey)
		if !hasEntry || dbentry.ID == "" {
			return nil, errors.New("internal error: no db entry")
		}
		return dbentry.ID, nil
	}

	remoteState, ok := b.RemoteStateCache.Load(targetResourceKey)
	if !ok {
		return nil, fmt.Errorf("internal error: %s: missing remote state", targetResourceKey)
	}

	return structaccess.Get(remoteState, fieldPath)
}

func jsonDump(obj map[string]string) string {
	bytes, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		return err.Error()
	}
	return string(bytes)
}
