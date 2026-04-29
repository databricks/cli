package direct

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/libs/structs/structaccess"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/ucm/deployplan"
	"github.com/databricks/databricks-sdk-go"
)

type MigrateMode bool

func (u *DeploymentUcm) Apply(ctx context.Context, client *databricks.WorkspaceClient, plan *deployplan.Plan, migrateMode MigrateMode) {
	if plan == nil {
		panic("Planning is not done")
	}

	if len(plan.Plan) == 0 {
		// Avoid creating state file if nothing to deploy
		return
	}

	u.StateDB.AssertOpened()
	u.RemoteStateCache.Clear()

	g, err := makeGraph(plan)
	if err != nil {
		logdiag.LogError(ctx, err)
		return
	}

	g.Run(defaultParallelism, func(resourceKey string, failedDependency *string) bool {
		entry, err := plan.WriteLockEntry(resourceKey)
		if err != nil {
			logdiag.LogError(ctx, fmt.Errorf("%s: internal error: %w", resourceKey, err))
			return false
		}

		if entry == nil {
			logdiag.LogError(ctx, fmt.Errorf("%s: internal error: node not in graph", resourceKey))
			return false
		}

		defer plan.WriteUnlockEntry(resourceKey)

		action := entry.Action
		errorPrefix := fmt.Sprintf("cannot %s %s", action, resourceKey)
		if migrateMode {
			errorPrefix = "cannot migrate " + resourceKey
		}

		if action == deployplan.Undefined {
			logdiag.LogError(ctx, fmt.Errorf("cannot deploy %s: unknown action %q", resourceKey, action))
			return false
		}

		// If a dependency failed, report and skip execution for this node by returning false
		if failedDependency != nil {
			if action != deployplan.Skip {
				logdiag.LogError(ctx, fmt.Errorf("%s: dependency failed: %s", errorPrefix, *failedDependency))
			}
			return false
		}

		adapter, err := u.getAdapterForKey(resourceKey)
		if adapter == nil {
			logdiag.LogError(ctx, fmt.Errorf("%s: internal error: cannot get adapter: %w", errorPrefix, err))
			return false
		}

		d := &DeploymentUnit{
			ResourceKey: resourceKey,
			Adapter:     adapter,
			DependsOn:   entry.DependsOn,
		}

		if action == deployplan.Delete {
			if migrateMode {
				logdiag.LogError(ctx, fmt.Errorf("%s: Unexpected delete action during migration", errorPrefix))
				return false
			}
			err = d.Destroy(ctx, &u.StateDB)
			if err != nil {
				logdiag.LogError(ctx, fmt.Errorf("%s: %w", errorPrefix, err))
				return false
			}
			return true
		}

		// We don't keep NewState around for 'skip' nodes

		if action != deployplan.Skip {
			if !u.resolveReferences(ctx, resourceKey, entry, errorPrefix, false) {
				return false
			}

			// Get the cached StructVar to check for unresolved refs and get value
			sv, ok := u.StateCache.Load(resourceKey)
			if !ok {
				logdiag.LogError(ctx, fmt.Errorf("%s: internal error: missing cached StructVar", errorPrefix))
				return false
			}

			if len(sv.Refs) > 0 {
				logdiag.LogError(ctx, fmt.Errorf("%s: unresolved references: %s", errorPrefix, jsonDump(sv.Refs)))
				return false
			}

			if migrateMode {
				// In migration mode we're reading resources in DAG order so that we have fully resolved config snapshots stored
				id := u.StateDB.GetResourceID(resourceKey)
				if id == "" {
					logdiag.LogError(ctx, fmt.Errorf("state entry not found for %q", resourceKey))
					return false
				}
				err = u.StateDB.SaveState(resourceKey, id, sv.Value, entry.DependsOn)
			} else {
				// TODO: redo calcDiff to downgrade planned action if possible (?)
				err = d.Deploy(ctx, &u.StateDB, sv.Value, action, entry)
			}

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
			id := u.StateDB.GetResourceID(d.ResourceKey)
			if id == "" {
				logdiag.LogError(ctx, fmt.Errorf("%s: internal error: missing entry in state after deploy", errorPrefix))
				return false
			}

			err = d.refreshRemoteState(ctx, id)
			if err != nil {
				logdiag.LogError(ctx, fmt.Errorf("%s: failed to read remote state: %w", errorPrefix, err))
				return false
			}
			u.RemoteStateCache.Store(resourceKey, d.RemoteState)
		}

		return true
	})
}

func (u *DeploymentUcm) LookupReferencePostDeploy(ctx context.Context, path *structpath.PathNode) (any, error) {
	targetResourceKey, fieldPath := splitResourcePath(path)
	fieldPathS := fieldPath.String()

	targetEntry, err := u.Plan.ReadLockEntry(targetResourceKey)
	if err != nil {
		return nil, err
	}

	if targetEntry == nil {
		return nil, fmt.Errorf("internal error: %s: missing entry in the plan", targetResourceKey)
	}

	defer u.Plan.ReadUnlockEntry(targetResourceKey)

	targetAction := targetEntry.Action
	if targetAction == deployplan.Undefined {
		return nil, fmt.Errorf("internal error: %s: missing action in the plan", targetResourceKey)
	}

	if fieldPathS == "id" {
		id := u.StateDB.GetResourceID(targetResourceKey)
		if id == "" {
			return nil, errors.New("internal error: no db entry")
		}
		return id, nil
	}

	remoteState, ok := u.RemoteStateCache.Load(targetResourceKey)
	if !ok {
		return nil, fmt.Errorf("internal error: %s: missing remote state", targetResourceKey)
	}

	return structaccess.Get(remoteState, fieldPath)
}

func jsonDump(obj any) string {
	bytes, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		return err.Error()
	}
	return string(bytes)
}
