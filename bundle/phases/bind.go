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
	bundleenv "github.com/databricks/cli/bundle/env"
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

// createBindingDeployment creates a deployment record for the first recorded binding change, without a deployed file path.
func createBindingDeployment(ctx context.Context, b *bundle.Bundle) (string, error) {
	// Bind does not upload bundle files, so it cannot report a deployed file path.
	metadata := bundleMetadata(b)
	deployment := metadata.Deployment()
	deployment.InitialParentPath = b.Config.Workspace.StatePath
	created, err := b.WorkspaceClient(ctx).BundleDeployments.CreateDeployment(ctx, bundledeployments.CreateDeploymentRequest{Deployment: deployment})
	if err != nil {
		return "", fmt.Errorf("failed to create deployment: %w", err)
	}
	deploymentID, err := dms.DeploymentIDFromName(created.Name)
	if err != nil {
		return "", fmt.Errorf("failed to create deployment: %w", err)
	}
	return deploymentID, nil
}

func Bind(ctx context.Context, b *bundle.Bundle, opts *BindOptions, stateDesc *statemgmt.StateDesc) {
	log.Info(ctx, "Phase: bind")
	engine := stateDesc.Engine

	bundle.ApplyContext(ctx, b, lock.Acquire(lock.GoalBind))
	if logdiag.HasError(ctx) {
		return
	}

	if !cmdctx.HasWorkspaceClient(ctx) {
		ctx = cmdctx.SetWorkspaceClient(ctx, b.WorkspaceClient(ctx))
	}
	db := &b.DeploymentBundle.StateDB
	defer func() {
		if _, err := db.Finalize(ctx); err != nil {
			logdiag.LogError(ctx, err)
		}
		if _, err := db.CompleteVersion(ctx, !logdiag.HasError(ctx)); err != nil {
			log.Warnf(ctx, "Failed to complete deployment history version: %s", err)
		}
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

	// Prepare the binding without writing state until the plan is approved.
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
			logdiag.LogError(ctx, fmt.Errorf("this bind operation requires user confirmation, but the current console does not support prompting.\nTo proceed, use --auto-approve after reviewing the plan above.%s", agent.AgentNotice()))
			return
		}

		ans, err := cmdio.AskYesOrNo(ctx, "Confirm import changes? Changes will be remotely applied only after running 'bundle deploy'.")
		if err != nil {
			logdiag.LogError(ctx, err)
			return
		}
		if !ans {
			logdiag.LogError(ctx, errors.New("import aborted"))
			return
		}
	}

	var configured *bool
	if b.Config.Experimental != nil {
		configured = b.Config.Experimental.DeploymentHistory
	}
	if db.RecordsHistory() || bundleenv.RecordsDeploymentHistory(ctx, configured) {
		if db.Data.Features == nil {
			db.Data.Features = make(map[string]struct{})
		}
		db.Data.Features[dstate.FeatureDeploymentHistoryShadow] = struct{}{}
	}
	if err := db.UpgradeToWrite(); err != nil {
		logdiag.LogError(ctx, err)
		return
	}
	if db.RecordsHistory() {
		if err := prepareBindingHistory(ctx, b, []string{resourceKey}); err != nil {
			log.Warnf(ctx, "Failed to record bind history: %s", err)
		}
	}
	resource := result.Resource
	if err := db.SaveState(ctx, resourceKey, resource.ID, resource.State, resource.DependsOn); err != nil {
		logdiag.LogError(ctx, err)
		return
	}
	if _, err := db.Finalize(ctx); err != nil {
		logdiag.LogError(ctx, err)
		return
	}
	statemgmt.PushResourcesState(ctx, b)
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

	if !cmdctx.HasWorkspaceClient(ctx) {
		ctx = cmdctx.SetWorkspaceClient(ctx, b.WorkspaceClient(ctx))
	}
	db := &b.DeploymentBundle.StateDB
	defer func() {
		if _, err := db.Finalize(ctx); err != nil {
			logdiag.LogError(ctx, err)
		}
		if _, err := db.CompleteVersion(ctx, !logdiag.HasError(ctx)); err != nil {
			log.Warnf(ctx, "Failed to complete deployment history version: %s", err)
		}
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
	if err := db.Open(ctx, statePath, dstate.ReadingFile, dstate.WithRecovery(true), dstate.OpenDmsArgs{}); err != nil {
		logdiag.LogError(ctx, err)
		return
	}
	keys := b.DeploymentBundle.UnbindResourceKeys(fullResourceKey)
	var configured *bool
	if b.Config.Experimental != nil {
		configured = b.Config.Experimental.DeploymentHistory
	}
	if db.RecordsHistory() || bundleenv.RecordsDeploymentHistory(ctx, configured) {
		if db.Data.Features == nil {
			db.Data.Features = make(map[string]struct{})
		}
		db.Data.Features[dstate.FeatureDeploymentHistoryShadow] = struct{}{}
	}
	if err := db.UpgradeToWrite(); err != nil {
		logdiag.LogError(ctx, err)
		return
	}
	if db.RecordsHistory() {
		if err := prepareBindingHistory(ctx, b, keys); err != nil {
			log.Warnf(ctx, "Failed to record unbind history: %s", err)
		}
	}
	if err := b.DeploymentBundle.Unbind(ctx, fullResourceKey, keys); err != nil {
		logdiag.LogError(ctx, err)
		return
	}
	if _, err := db.Finalize(ctx); err != nil {
		logdiag.LogError(ctx, err)
		return
	}

	statemgmt.PushResourcesState(ctx, b)
}

// prepareBindingHistory stages only binding changes, preserving deployed file metadata.
func prepareBindingHistory(ctx context.Context, b *bundle.Bundle, resourceKeys []string) error {
	if len(resourceKeys) == 0 {
		return nil
	}
	deploymentID, current, err := dms.FetchDeployment(ctx, b.WorkspaceClient(ctx), b.Config.Workspace.StatePath)
	if err != nil {
		return err
	}
	previous, err := dms.ParseLastVersionID(current)
	if err != nil {
		return err
	}
	// Existing metadata is refreshed by deploy, which applies the current configuration.
	if current == nil {
		deploymentID, err = createBindingDeployment(ctx, b)
		if err != nil {
			return err
		}
	}
	db := &b.DeploymentBundle.StateDB
	db.DeploymentID, db.VersionID = deploymentID, previous
	staged := make([]bundledeployments.StagedOperation, 0, len(resourceKeys))
	for _, key := range resourceKeys {
		staged = append(staged, bundledeployments.StagedOperation{
			ResourceKey: strings.TrimPrefix(key, dms.StatePrefix),
			ActionType:  bundledeployments.OperationActionTypeOperationActionTypeUpdate,
		})
	}
	return startVersion(ctx, b, dms.VersionTypeDeploy, staged)
}
