package direct

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/terraform_dabs_map"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/libs/structs/structaccess"
	"github.com/databricks/cli/libs/structs/structdiff"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/databricks-sdk-go"
)

// Apply deploys every node in plan, respecting dependency order. When reportApplied
// is set, each resource is reported as soon as it is applied, so a long deploy shows
// what it has done and a failing one still reports the resources it did apply. Nodes
// run in parallel, so the lines come out in completion order, which varies per run.
func (b *DeploymentBundle) Apply(ctx context.Context, client *databricks.WorkspaceClient, plan *deployplan.Plan, reportApplied bool) {
	if plan == nil {
		panic("Planning is not done")
	}

	// Read before the early return below so a malformed value is reported even when there is
	// nothing to deploy.
	maxWait, err := resourceMaxWait(ctx)
	if err != nil {
		logdiag.LogError(ctx, err)
		return
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

	// The state DB records every write with DMS from here on (via the buffer InitializeOperationBuffer opened
	// in the deploy phase), so the service mirrors the WAL. Writes go out on one background
	// goroutine, off the apply path, and are drained below once every worker has finished recording.
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

		// Stop resource CRUD once recording state with DMS has failed.
		if err := b.StateDB.RecordingError(); err != nil {
			logdiag.LogError(ctx, fmt.Errorf("%s: %w", errorPrefix, err))
			return false
		}

		adapter, err := b.getAdapterForKey(resourceKey)
		if adapter == nil {
			logdiag.LogError(ctx, fmt.Errorf("%s: internal error: cannot get adapter: %w", errorPrefix, err))
			return false
		}

		// Deletes are capped even with dependents: state is dropped before the wait, so a
		// cut-short delete leaves the resource untracked while it tears down, and a dependency
		// deleted after it may be rejected for still having a child. Accepted deliberately.
		// Recreate's internal delete-wait is never routed through the cap at all, because it
		// releases the name for the create that follows.
		unitWait := maxWait
		if action != deployplan.Delete && hasBlockingDependents(g, resourceKey) {
			unitWait = maxWaitUnset
			if maxWait != maxWaitUnset {
				log.Debugf(ctx, "Not capping wait for %s: other resources depend on it", resourceKey)
			}
		}

		d := &DeploymentUnit{
			ResourceKey: resourceKey,
			Adapter:     adapter,
			DependsOn:   entry.DependsOn,
			MaxWait:     unitWait,
		}

		if action == deployplan.Delete {
			if entry.IsStateOnlyDelete() {
				// The resource is already deleted remotely (Gone) or has no delete
				// operation (StateOnly); either way only remove it from the state,
				// without calling the delete API.
				err = b.StateDB.DeleteState(ctx, resourceKey, false)
			} else {
				err = d.Destroy(ctx, &b.StateDB)
			}
			if err != nil {
				logdiag.LogError(ctx, fmt.Errorf("%s: %w", errorPrefix, err))
				return false
			}
			// A state-only delete performs no backend operation, so don't report it,
			// consistent with the summary (CountActions excludes it) and the terraform
			// path in logDeploySummary.
			if reportApplied && !entry.IsStateOnlyDelete() {
				cmdio.LogString(ctx, deployplan.AppliedLine(resourceKey, action))
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

			// References are now resolved, so re-check the planned changes and drop any that were
			// only a phantom of an unresolved reference. A field that references a resource being
			// recreated could not be read at plan time (a recreate does not preserve the id), so it
			// diffed against the literal "${...}" placeholder (New) and inflated the action. A change
			// is a phantom when it was a local edit -- New differs from the saved value Old -- yet the
			// field's now-resolved value equals Old: the reference resolved back to an unchanged value.
			// Recompute the action over what remains and downgrade.
			//
			// This only ever lowers the action: a genuine local edit resolves to a value that still
			// differs from Old and is kept, and a remote-only change (New == Old, remote drift -- e.g.
			// a permissions child re-applying its ACL) is never a phantom, so those are kept too.
			// Starting from the plan's own Changes rather than re-diffing from scratch also preserves
			// changes a struct diff here would not surface, such as a cluster's libraries.
			//
			// Child resources (permissions, grants) are excluded: their diff is specialized -- a
			// stale object_id/full_name echoed by DoRead drives re-application -- which the
			// value-level phantom test here would misread.
			if action != deployplan.Create && len(entry.Changes) > 0 && !deployplan.IsChildResourceKey(resourceKey) {
				remaining := make(deployplan.Changes, len(entry.Changes))
				for pathStr, ch := range entry.Changes {
					if path, perr := structpath.ParsePath(pathStr); perr == nil {
						newVal, gerr := structaccess.Get(sv.Value, path)
						if gerr == nil && !structdiff.IsEqual(ch.New, ch.Old) && structdiff.IsEqual(newVal, ch.Old) {
							// A local edit that an unresolved reference stood in for; the reference
							// resolved back to the saved value, so there is no change. Drop it.
							continue
						}
					}
					remaining[pathStr] = ch
				}
				if recomputed := getMaxAction(remaining); deployplan.GetHigherAction(recomputed, action) == action && recomputed != action {
					log.Infof(ctx, "%s: downgrading %s to %s after resolving references", resourceKey, action, recomputed)
					action = recomputed
					entry.Action = action
				}
			}

			if action != deployplan.Skip {
				// Success is recorded by the state writes inside Deploy, so a recreate reports
				// each of its steps.
				err = d.Deploy(ctx, &b.StateDB, sv.Value, action, entry)
				if err != nil {
					// Empty for a create that never got an ID, and for a recreate whose delete
					// step already dropped it.
					failedID := b.StateDB.GetResourceID(resourceKey)
					b.StateDB.RecordFailure(resourceKey, failedID, err)
					logdiag.LogError(ctx, fmt.Errorf("%s: %w", errorPrefix, err))
					return false
				}

				// Reported before the remote-state refresh below: the resource is already
				// deployed at this point, so the line is accurate even if the refresh fails.
				if reportApplied {
					cmdio.LogString(ctx, deployplan.AppliedLine(resourceKey, action))
				}
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
