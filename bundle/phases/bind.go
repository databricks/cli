package phases

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/engine"
	"github.com/databricks/cli/bundle/deploy/lock"
	"github.com/databricks/cli/bundle/deploy/snapshot"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/direct/dstate"
	bundleenv "github.com/databricks/cli/bundle/env"
	"github.com/databricks/cli/bundle/statemgmt"
	"github.com/databricks/cli/internal/build"
	"github.com/databricks/cli/libs/agent"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/dms"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/libs/retry"
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
func createBindingDeployment(ctx context.Context, b *bundle.Bundle) (string, error) {
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

	var historyVersion string
	var historyErr error
	defer func() {
		completeBindingHistory(ctx, b, historyVersion, historyErr == nil && !logdiag.HasError(ctx))
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

	historyVersion, historyErr = recordBindHistory(ctx, b, resourceKey)
	if historyErr != nil {
		log.Warnf(ctx, "Failed to record bind history: %s", historyErr)
	}

	statemgmt.PushResourcesState(ctx, b)
}

// completeBindingHistory completes the recorded version after the remote state upload.
func completeBindingHistory(ctx context.Context, b *bundle.Bundle, versionName string, success bool) {
	if versionName == "" {
		return
	}
	reason := bundledeployments.VersionCompleteVersionCompleteFailure
	if success {
		reason = bundledeployments.VersionCompleteVersionCompleteSuccess
	}
	_, err := b.WorkspaceClient(ctx).BundleDeployments.CompleteVersion(ctx, bundledeployments.CompleteVersionRequest{
		Name:             versionName,
		CompletionReason: reason,
	})
	if err != nil {
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

// recordBindHistory publishes a committed binding to DMS without changing local state.
func recordBindHistory(ctx context.Context, b *bundle.Bundle, resourceKey string) (versionName string, err error) {
	db := &b.DeploymentBundle.StateDB
	_, statePath := b.StateFilenameDirect(ctx)
	if err := db.Open(ctx, statePath, dstate.ReadingFile, dstate.WithRecovery(true), dstate.OpenDmsArgs{}); err != nil {
		return "", err
	}
	defer func() {
		_, finalizeErr := db.Finalize(ctx)
		err = errors.Join(err, finalizeErr)
	}()

	var configured *bool
	if b.Config.Experimental != nil {
		configured = b.Config.Experimental.DeploymentHistory
	}
	if !db.RecordsHistory() && !bundleenv.RecordsDeploymentHistory(ctx, configured) {
		return "", nil
	}

	deploymentID, current, err := dms.LookupDeployment(ctx, b.WorkspaceClient(ctx), b.Config.Workspace.StatePath)
	if err != nil {
		return "", err
	}
	previous, err := dms.ParseLastVersionID(current)
	if err != nil {
		return "", err
	}
	if current == nil {
		deploymentID, err = createBindingDeployment(ctx, b)
		if err != nil {
			return "", err
		}
	}
	versionID := max(previous, db.Data.Serial) + 1
	previousVersionID := ""
	if previous > 0 {
		previousVersionID = strconv.Itoa(previous)
	}
	var gitInfo *bundledeployments.GitInfo
	if git := b.Config.Bundle.Git; git.Branch != "" || git.Commit != "" || git.OriginURL != "" {
		gitInfo = &bundledeployments.GitInfo{Branch: git.Branch, Commit: git.Commit, OriginUrl: git.OriginURL}
	}
	version, err := b.WorkspaceClient(ctx).BundleDeployments.CreateVersion(ctx, bundledeployments.CreateVersionRequest{
		Parent:    dms.DeploymentName(deploymentID),
		VersionId: strconv.Itoa(versionID),
		Version: bundledeployments.Version{
			CliVersion:        build.GetInfo().Version,
			VersionType:       dms.VersionTypeDeploy,
			PreviousVersionId: previousVersionID,
			GitInfo:           gitInfo,
			Operations: []bundledeployments.StagedOperation{{
				ResourceKey: strings.TrimPrefix(resourceKey, dms.StatePrefix),
				ActionType:  bundledeployments.OperationActionTypeOperationActionTypeBind,
			}},
		},
	})
	if err != nil {
		return "", fmt.Errorf("failed to create deployment version: %w", err)
	}
	versionName = version.Name
	entry, ok := db.GetResourceEntry(resourceKey)
	if !ok {
		return versionName, fmt.Errorf("bound resource %s is missing from deployment state", resourceKey)
	}
	return versionName, recordBindingOperation(ctx, b, versionName, resourceKey, &entry)
}

// recordBindingOperation sends an operation's saved state, or its removal, directly to DMS.
func recordBindingOperation(ctx context.Context, b *bundle.Bundle, versionName, resourceKey string, entry *dstate.ResourceEntry) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	var state json.RawMessage
	var resourceID string
	if entry != nil {
		var err error
		state, err = json.Marshal(dstate.RecordedState{State: entry.State, DependsOn: entry.DependsOn})
		if err != nil {
			return err
		}
		resourceID = entry.ID
	}
	update, err := dms.NewStateUpdate(resourceID, state, false)
	if err != nil {
		return err
	}
	_, err = retry.With(ctx, retry.IsTransient, func() (*bundledeployments.Operation, error) {
		return b.WorkspaceClient(ctx).BundleDeployments.UpdateOperation(ctx, bundledeployments.UpdateOperationRequest{
			Name: versionName + "/operations/" + strings.TrimPrefix(resourceKey, dms.StatePrefix),
			Operation: bundledeployments.Operation{
				State:           string(update.State),
				ResourceId:      update.ResourceID,
				Status:          update.Status,
				ForceSendFields: []string{"SequenceId", "ErrorMessage"},
			},
			UpdateMask: update.Fields.Mask(),
		})
	})
	return err
}
