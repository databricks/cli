package direct

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/terraform_dabs_map"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/libs/structs/structaccess"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/databricks-sdk-go"
)

type MigrateMode bool

func (b *DeploymentBundle) Apply(ctx context.Context, client *databricks.WorkspaceClient, plan *deployplan.Plan, migrateMode MigrateMode) {
	if plan == nil {
		panic("Planning is not done")
	}

	// Migrate mode rewrites local state in-place and does not call resource
	// adapters, so there are no operations to report. Reject the combination
	// rather than silently dropping migrations from the DMS view.
	if migrateMode && b.OperationReporter != nil {
		logdiag.LogError(ctx, errors.New("migration is not supported with the deployment metadata service"))
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
			if migrateMode {
				// Resource is in terraform state but not in config. Preserve its ID in
				// direct state so the next direct deploy will plan and execute deletion.
				id := b.StateDB.GetResourceID(resourceKey)
				if id == "" {
					logdiag.LogError(ctx, fmt.Errorf("%s: internal error: no ID in state", errorPrefix))
					return false
				}
				if err = b.StateDB.SaveState(resourceKey, id, json.RawMessage("{}"), entry.DependsOn); err != nil {
					logdiag.LogError(ctx, fmt.Errorf("%s: %w", errorPrefix, err))
					return false
				}
				return true
			}

			// Capture the resource ID before Destroy clears it from state so
			// the DMS report carries a usable resource_id.
			var deleteResourceID string
			if b.OperationReporter != nil {
				deleteResourceID = b.StateDB.GetResourceID(resourceKey)
			}

			err = d.Destroy(ctx, &b.StateDB)
			if reportErr := b.reportOperation(ctx, resourceKey, deleteResourceID, action, err, nil); reportErr != nil {
				logdiag.LogError(ctx, fmt.Errorf("%s: failed to report operation: %w", errorPrefix, reportErr))
				return false
			}
			if err != nil {
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

			if migrateMode {
				// In migration mode we're reading resources in DAG order so that we have fully resolved config snapshots stored
				id := b.StateDB.GetResourceID(resourceKey)
				if id == "" {
					logdiag.LogError(ctx, fmt.Errorf("state entry not found for %q", resourceKey))
					return false
				}
				err = b.StateDB.SaveState(resourceKey, id, sv.Value, entry.DependsOn)
			} else {
				// TODO: redo calcDiff to downgrade planned action if possible (?)
				err = d.Deploy(ctx, &b.StateDB, sv.Value, action, entry)
			}

			// Report the operation outcome to DMS (if enabled). On success
			// we report the post-deploy resource ID + the new local state;
			// on failure we report the action and error message and omit
			// the state payload (matches OPERATION_STATUS_FAILED conventions).
			// migrateMode is rejected up-front above when DMS is enabled, so
			// it's safe to always go through reportOperation here.
			if b.OperationReporter != nil {
				var reportState json.RawMessage
				var reportID string
				if err == nil {
					reportID = b.StateDB.GetResourceID(resourceKey)
					stateBytes, marshalErr := json.Marshal(sv.Value)
					if marshalErr != nil {
						logdiag.LogError(ctx, fmt.Errorf("%s: marshalling state for report: %w", errorPrefix, marshalErr))
						return false
					}
					reportState = stateBytes
				}
				if reportErr := b.reportOperation(ctx, resourceKey, reportID, action, err, reportState); reportErr != nil {
					logdiag.LogError(ctx, fmt.Errorf("%s: failed to report operation: %w", errorPrefix, reportErr))
					return false
				}
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

// reportOperation forwards a single resource operation outcome to the
// configured OperationReporter (if any). Returning nil when the reporter is
// unset lets callers invoke this unconditionally and keeps the DMS-off path
// identical to before.
func (b *DeploymentBundle) reportOperation(
	ctx context.Context,
	resourceKey, resourceID string,
	action deployplan.ActionType,
	operationErr error,
	state json.RawMessage,
) error {
	if b.OperationReporter == nil {
		return nil
	}
	return b.OperationReporter(ctx, resourceKey, resourceID, action, operationErr, state)
}

func jsonDump(obj any) string {
	bytes, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		return err.Error()
	}
	return string(bytes)
}
