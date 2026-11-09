package phases

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/engine"
	"github.com/databricks/cli/bundle/deploy/lock"
	"github.com/databricks/cli/bundle/deploy/snapshot"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/direct/dstate"
	"github.com/databricks/cli/bundle/statemgmt"
	"github.com/databricks/cli/libs/agent"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/dms"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/databricks-sdk-go/service/bundledeployments"
)

type BindOptions struct {
	AutoApprove bool
	// Group is the bundle resource group, e.g. "jobs".
	Group       string
	ResourceKey string
	ResourceId  string
}

// createBindingDeployment registers a first binding without changing metadata from an existing deploy.
func createBindingDeployment(ctx context.Context, b *bundle.Bundle, current *bundledeployments.Deployment) error {
	if current != nil {
		return nil
	}
	// Bind does not upload files or stage an immutable snapshot, so it cannot report a deployed file path.
	created, err := b.WorkspaceClient(ctx).BundleDeployments.CreateDeployment(ctx, bundledeployments.CreateDeploymentRequest{
		Deployment: bundledeployments.Deployment{
			DisplayName:       b.Config.Bundle.Name,
			TargetName:        b.Config.Bundle.Target,
			DeploymentMode:    deploymentModeToSDK(b.Config.Bundle.Mode),
			WorkspaceInfo:     &bundledeployments.WorkspaceInfo{RootPath: b.Config.Workspace.RootPath},
			InitialParentPath: b.Config.Workspace.StatePath,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create deployment: %w", err)
	}
	deploymentID, err := dms.DeploymentIDFromName(created.Name)
	if err != nil {
		return fmt.Errorf("failed to create deployment: %w", err)
	}
	b.DeploymentBundle.StateDB.DeploymentID = deploymentID
	return nil
}

func Bind(ctx context.Context, b *bundle.Bundle, opts *BindOptions, stateDesc *statemgmt.StateDesc) {
	log.Info(ctx, "Phase: bind")
	engine := stateDesc.Engine

	bundle.ApplyContext(ctx, b, lock.Acquire(lock.GoalBind))
	if logdiag.HasError(ctx) {
		return
	}

	historySucceeded := true
	defer func() {
		completeBindingHistory(ctx, b, historySucceeded && !logdiag.HasError(ctx))
		bundle.ApplyContext(ctx, b, lock.Release(lock.GoalBind))
	}()

	// The terraform engine was removed in v1.20.0. bind does not migrate on its own, so a
	// terraform state means the user must migrate first (via "bundle deploy").
	if !engine.IsDirect() {
		logdiag.LogError(ctx, errors.New(TerraformStateRemovedMessage))
		return
	}

	// In immutable sync mode, translated paths reference the snapshot (an internal resource,
	// see resources.SnapshotFullPathRef) that only snapshot.PlanUpload registers in the config.
	// The plans below cannot resolve those references without it, so register it as plan and
	// deploy do.
	if b.IsImmutableFolder() {
		bundle.ApplyContext(ctx, b, snapshot.PlanUpload(snapshot.PlanUploadOptions{Clean: true}))
		if logdiag.HasError(ctx) {
			return
		}
	}

	// Direct engine: import into temp state, run plan, check for changes.
	resourceKey := fmt.Sprintf("resources.%s.%s", opts.Group, opts.ResourceKey)
	_, statePath := b.StateFilenameDirect(ctx)

	result, err := b.DeploymentBundle.Bind(ctx, b.WorkspaceClient(ctx), &b.Config, statePath, resourceKey, opts.ResourceId)
	if err != nil {
		logdiag.LogError(ctx, err)
		return
	}

	// If there are changes and auto-approve is not set, show plan and ask for confirmation
	if result.HasChanges && !opts.AutoApprove {
		// Display the planned changes for the bound resource
		cmdio.LogString(ctx, fmt.Sprintf("Plan: %s %s", result.Action, resourceKey))

		// Show details of what will change
		if result.Plan != nil {
			if entry, ok := result.Plan.Plan[resourceKey]; ok && entry != nil && len(entry.Changes) > 0 {
				cmdio.LogString(ctx, "\nChanges detected:")
				for _, field := range slices.Sorted(maps.Keys(entry.Changes)) {
					change := entry.Changes[field]
					if change.Action != deployplan.Skip {
						cmdio.LogString(ctx, fmt.Sprintf("  ~ %s: %v -> %v", field, jsonDump(ctx, change.Remote, field), jsonDump(ctx, change.New, field)))
					}
				}
				cmdio.LogString(ctx, "")
			}
		}

		if !cmdio.IsPromptSupported(ctx) {
			result.Cancel()
			logdiag.LogError(ctx, fmt.Errorf("this bind operation requires user confirmation, but the current console does not support prompting.\nTo proceed, use --auto-approve after reviewing the plan above.%s", agent.AgentNotice()))
			return
		}

		ans, err := cmdio.AskYesOrNo(ctx, "Confirm import changes? Changes will be remotely applied only after running 'bundle deploy'.")
		if err != nil {
			result.Cancel()
			logdiag.LogError(ctx, err)
			return
		}
		if !ans {
			result.Cancel()
			logdiag.LogError(ctx, errors.New("import aborted"))
			return
		}
	}

	// Finalize: rename temp state to final location
	err = result.Finalize()
	if err != nil {
		logdiag.LogError(ctx, err)
		return
	}

	if err := recordBindHistory(ctx, b, resourceKey); err != nil {
		historySucceeded = false
		log.Warnf(ctx, "Failed to record bind history: %s", err)
	}

	statemgmt.PushResourcesState(ctx, b)
}

// completeBindingHistory completes the recorded version after the remote state upload.
func completeBindingHistory(ctx context.Context, b *bundle.Bundle, success bool) {
	if !b.DeploymentBundle.StateDB.RecordsHistory() {
		return
	}
	completionCtx := cmdctx.SetWorkspaceClient(ctx, b.WorkspaceClient(ctx))
	if _, err := b.DeploymentBundle.StateDB.CompleteVersion(completionCtx, success); err != nil {
		log.Warnf(ctx, "Failed to complete deployment history version: %s", err)
	}
}

func jsonDump(ctx context.Context, v any, field string) string {
	b, err := json.Marshal(v)
	if err != nil {
		log.Warnf(ctx, "Cannot marshal %s: %s", field, err)
		return "??"
	}
	return string(b)
}

func Unbind(ctx context.Context, b *bundle.Bundle, group, resourceKey string, engine engine.EngineType) {
	log.Info(ctx, "Phase: unbind")

	bundle.ApplyContext(ctx, b, lock.Acquire(lock.GoalUnbind))
	if logdiag.HasError(ctx) {
		return
	}

	defer func() {
		bundle.ApplyContext(ctx, b, lock.Release(lock.GoalUnbind))
	}()

	// The terraform engine was removed in v1.20.0. unbind does not migrate on its own, so a
	// terraform state means the user must migrate first (via "bundle deploy").
	if !engine.IsDirect() {
		logdiag.LogError(ctx, errors.New(TerraformStateRemovedMessage))
		return
	}

	fullResourceKey := fmt.Sprintf("resources.%s.%s", group, resourceKey)
	_, statePath := b.StateFilenameDirect(ctx)
	err := b.DeploymentBundle.Unbind(ctx, statePath, fullResourceKey)
	if err != nil {
		logdiag.LogError(ctx, err)
		return
	}

	statemgmt.PushResourcesState(ctx, b)
}

// recordBindHistory records a committed binding's saved state when history recording is enabled.
func recordBindHistory(ctx context.Context, b *bundle.Bundle, resourceKey string) (err error) {
	db := &b.DeploymentBundle.StateDB
	_, statePath := b.StateFilenameDirect(ctx)
	if err := db.Open(ctx, statePath, dstate.ReadingFile, dstate.WithRecovery(true), dstate.OpenDmsArgs{}); err != nil {
		return err
	}
	defer func() {
		_, finalizeErr := db.Finalize(ctx)
		err = errors.Join(err, finalizeErr)
	}()

	if !db.RecordsHistory() {
		return nil
	}

	deploymentID, current, err := dms.LookupDeployment(ctx, b.WorkspaceClient(ctx), b.Config.Workspace.StatePath)
	if err != nil {
		return err
	}
	db.DeploymentID = deploymentID
	db.VersionID, err = dms.ParseLastVersionID(current)
	if err != nil {
		return err
	}

	if err := db.UpgradeToWrite(); err != nil {
		return err
	}
	if !cmdctx.HasWorkspaceClient(ctx) {
		ctx = cmdctx.SetWorkspaceClient(ctx, b.WorkspaceClient(ctx))
	}
	if err := createBindingDeployment(ctx, b, current); err != nil {
		return err
	}
	if err := startVersion(ctx, b, dms.VersionTypeDeploy, []bundledeployments.StagedOperation{{
		ResourceKey: strings.TrimPrefix(resourceKey, dms.StatePrefix),
		ActionType:  bundledeployments.OperationActionTypeOperationActionTypeBind,
	}}); err != nil {
		return err
	}
	entry, ok := db.GetResourceEntry(resourceKey)
	if !ok {
		return fmt.Errorf("bound resource %s is missing from deployment state", resourceKey)
	}
	return db.RecordResourceHistory(ctx, resourceKey, &entry)
}
