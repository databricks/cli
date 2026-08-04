package direct

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/terraform_dabs_map"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/libs/structs/structaccess"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/databricks-sdk-go"
)

func (b *DeploymentBundle) Apply(ctx context.Context, client *databricks.WorkspaceClient, plan *deployplan.Plan) {
	if plan == nil {
		panic("Planning is not done")
	}

	if len(plan.Plan) == 0 {
		// Avoid creating state file if nothing to deploy
		return
	}

	b.StateDB.AssertOpenedForWrite()
	b.RemoteStateCache.Clear()

	g, err := makeGraph(plan)
	if err != nil {
		logdiag.LogError(ctx, err)
		return
	}

	// Operations are recorded with DMS from background workers so a resource's
	// deploy is not held up by the CreateOperation round trip. The queue is
	// drained below, once every apply worker has finished recording.
	opQueue := newOperationQueue(ctx, b.OpRec)

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

		// Stop before touching the workspace once recording an operation has failed.
		// A completed version makes DMS the source of truth for resource state (see
		// dstate.readDMSState), so continuing would create resources it has no record
		// of and the next deploy would create them a second time. Checked here rather
		// than only where operations are recorded, which is after the resource has
		// already been modified.
		if err := opQueue.firstErr(); err != nil {
			logdiag.LogError(ctx, fmt.Errorf("%s: %w", errorPrefix, err))
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
			DependsOn:   entry.DependsOn,
		}

		if action == deployplan.Delete {
			// Capture the ID before the delete: DMS requires resource_id on a
			// DELETE operation, but both Destroy and DeleteState drop it from state,
			// so GetResourceID would return empty afterwards.
			resourceID := b.StateDB.GetResourceID(resourceKey)
			if entry.Gone {
				// Planning confirmed the resource is already deleted remotely; only
				// remove it from the state, without calling the delete API.
				err = b.StateDB.DeleteState(resourceKey)
			} else {
				err = d.Destroy(ctx, &b.StateDB)
			}
			if err != nil {
				// Record the delete failure with DMS so the version captures why.
				// Best-effort: don't let a recording error mask the delete error.
				if recErr := opQueue.recordFailure(ctx, resourceKey, action, resourceID, err.Error()); recErr != nil {
					log.Debugf(ctx, "failed to record operation failure for %s: %v", resourceKey, recErr)
				}
				logdiag.LogError(ctx, fmt.Errorf("%s: %w", errorPrefix, err))
				return false
			}
			// Record the delete with DMS. State is nil: the resource is gone.
			if err := opQueue.record(ctx, resourceKey, action, resourceID, nil, nil); err != nil {
				logdiag.LogError(ctx, fmt.Errorf("%s: %w", errorPrefix, err))
				return false
			}
			return true
		}

		// We don't keep NewState around for 'skip' nodes

		if action != deployplan.Skip {
			if !b.resolveReferences(ctx, resourceKey, entry, errorPrefix, false) {
				return false
			}

			// Get the cached StructVar to check for unresolved refs and get value
			sv, ok := b.StateCache.Load(resourceKey)
			if !ok {
				logdiag.LogError(ctx, fmt.Errorf("%s: internal error: missing cached StructVar", errorPrefix))
				return false
			}

			if len(sv.Refs) > 0 {
				logdiag.LogError(ctx, fmt.Errorf("%s: unresolved references: %s", errorPrefix, jsonDump(sv.Refs)))
				return false
			}

			// TODO: redo calcDiff to downgrade planned action if possible (?)
			err = d.Deploy(ctx, &b.StateDB, sv.Value, action, entry)
			if err != nil {
				// Record the failure with DMS so the version captures why it failed,
				// not just that it did. Best-effort: a recording error must not mask
				// the deploy error, which is the one the user needs to see.
				if recErr := opQueue.recordFailure(ctx, resourceKey, action, b.StateDB.GetResourceID(resourceKey), err.Error()); recErr != nil {
					log.Debugf(ctx, "failed to record operation failure for %s: %v", resourceKey, recErr)
				}
				logdiag.LogError(ctx, fmt.Errorf("%s: %w", errorPrefix, err))
				return false
			}

			// Record the operation with DMS. The resource ID and applied config
			// (sv.Value) come from the write just performed; GetResourceID reads
			// the ID assigned by Deploy. depends_on is recorded alongside the config
			// because it cannot be recomputed from it (see dstate.RecordedState).
			if err := opQueue.record(ctx, resourceKey, action, b.StateDB.GetResourceID(resourceKey), sv.Value, d.DependsOn); err != nil {
				logdiag.LogError(ctx, fmt.Errorf("%s: %w", errorPrefix, err))
				return false
			}
		}

		// TODO: Note, we only really need remote state if there are remote references.
		//       The graph includes edges for both local and remote references. The local references are
		//       already resolved and should not play a role here.
		needRemoteState := len(g.Adj[resourceKey]) > 0
		if needRemoteState {
			id := b.StateDB.GetResourceID(d.ResourceKey)
			if id == "" {
				logdiag.LogError(ctx, fmt.Errorf("%s: internal error: missing entry in state after deploy", errorPrefix))
				return false
			}

			err = d.refreshRemoteState(ctx, id)
			if err != nil {
				logdiag.LogError(ctx, fmt.Errorf("%s: failed to read remote state: %w", errorPrefix, err))
				return false
			}
			b.RemoteStateCache.Store(resourceKey, d.RemoteState)
		}

		return true
	})

	// Wait for the queued operations before returning: the caller completes the
	// DMS version right after, and a version must not be completed with uploads
	// still in flight.
	if err := opQueue.close(); err != nil {
		logdiag.LogError(ctx, err)
	}
}

func (b *DeploymentBundle) LookupReferencePostDeploy(ctx context.Context, path *structpath.PathNode) (any, error) {
	targetResourceKey, fieldPath := splitResourcePath(path)
	targetGroup := config.GetResourceTypeFromKey(targetResourceKey)

	// Translate Terraform-style field paths to DABs naming before lookup.
	fieldPath, err := terraform_dabs_map.TerraformPathToDABs(targetGroup, fieldPath)
	if err != nil {
		return nil, err
	}
	fieldPathS := fieldPath.String()

	targetEntry, err := b.Plan.ReadLockEntry(targetResourceKey)
	if err != nil {
		return nil, err
	}

	if targetEntry == nil {
		return nil, fmt.Errorf("internal error: %s: missing entry in the plan", targetResourceKey)
	}

	defer b.Plan.ReadUnlockEntry(targetResourceKey)

	targetAction := targetEntry.Action
	if targetAction == deployplan.Undefined {
		return nil, fmt.Errorf("internal error: %s: missing action in the plan", targetResourceKey)
	}

	if fieldPathS == "id" {
		id := b.StateDB.GetResourceID(targetResourceKey)
		if id == "" {
			return nil, errors.New("internal error: no db entry")
		}
		return id, nil
	}

	remoteState, ok := b.RemoteStateCache.Load(targetResourceKey)
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
