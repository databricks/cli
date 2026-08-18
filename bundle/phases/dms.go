package phases

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/engine"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/dms"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/workspaceurls"
	"github.com/databricks/databricks-sdk-go/service/bundledeployments"
)

// newRecording returns what this run records with DMS, or a disabled recording when
// nothing is: recording needs the direct engine and the bundle's opt-in. The deployment ID
// is resolved from the workspace node, and is empty on a first deploy.
func newRecording(ctx context.Context, b *bundle.Bundle, eng engine.EngineType, versionType dms.VersionType) (dms.Recording, error) {
	if !b.RecordsDeploymentHistory(ctx) || !eng.IsDirect() {
		return dms.Disabled(), nil
	}

	w := b.WorkspaceClient(ctx)
	statePath := b.Config.Workspace.StatePath
	deploymentID, err := dms.ResolveDeploymentID(ctx, w, statePath)
	if err != nil {
		return nil, err
	}
	client, err := dms.NewClient(w)
	if err != nil {
		return nil, err
	}
	return dms.NewRecording(dms.RecordingOptions{
		Client:       client,
		DeploymentID: deploymentID,
		StatePath:    statePath,
		VersionType:  versionType,
		Metadata:     deploymentMetadata(b),
	}), nil
}

// stagedOperations lists the resources the plan will touch, for CreateVersion to stage an
// operation each. Skipped and undefined actions are left out: nothing is applied for them, so
// their operations would stay pending and the service would hold no state for them.
func stagedOperations(plan *deployplan.Plan) ([]dms.StagedOperation, error) {
	actions := plan.GetActions()
	staged := make([]dms.StagedOperation, 0, len(actions))
	for _, action := range actions {
		if action.ActionType == deployplan.Skip || action.ActionType == deployplan.Undefined {
			continue
		}
		actionType, err := actionToSDK(action.ActionType)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", action.ResourceKey, err)
		}
		staged = append(staged, dms.StagedOperation{
			ResourceKey: dms.KeyFromState(action.ResourceKey),
			ActionType:  actionType,
		})
	}
	return staged, nil
}

// actionToSDK maps a deployplan action to the DMS action type a staged operation records.
// Only actions that mutate a resource are recordable; Skip and Undefined are rejected
// rather than silently coerced.
func actionToSDK(a deployplan.ActionType) (bundledeployments.OperationActionType, error) {
	switch a {
	case deployplan.Create:
		return bundledeployments.OperationActionTypeOperationActionTypeCreate, nil
	case deployplan.Update:
		return bundledeployments.OperationActionTypeOperationActionTypeUpdate, nil
	case deployplan.UpdateWithID:
		return bundledeployments.OperationActionTypeOperationActionTypeUpdateWithId, nil
	case deployplan.Recreate:
		return bundledeployments.OperationActionTypeOperationActionTypeRecreate, nil
	case deployplan.Resize:
		return bundledeployments.OperationActionTypeOperationActionTypeResize, nil
	case deployplan.Delete:
		return bundledeployments.OperationActionTypeOperationActionTypeDelete, nil
	default:
		return "", fmt.Errorf("cannot record operation: unsupported action %q", a)
	}
}

// setOperationWriter has the state writes during apply recorded under the started version.
// A disabled recording leaves the writer unset, which is also what keeps the state DB from
// serializing an envelope for every write.
func setOperationWriter(b *bundle.Bundle, recording dms.Recording, writer dms.OperationWriter) {
	if !recording.Enabled() {
		return
	}
	b.DeploymentBundle.OpRec = writer
}

// logDeploymentVersion logs the deployment version URL. Workspace ID is omitted
// so the page stays clickable in a terminal and redirects correctly without it.
func logDeploymentVersion(ctx context.Context, b *bundle.Bundle, recording dms.Recording) {
	if recording.Version() == 0 {
		return
	}

	baseURL, err := url.Parse(b.WorkspaceClient(ctx).Config.CanonicalHostName())
	if err != nil {
		// Only the link is lost, so report the version without it rather than failing
		// a deploy over it.
		log.Debugf(ctx, "Not linking to the recorded deployment: %s", err)
		cmdio.LogString(ctx, fmt.Sprintf("Current Deployment Version: %s version %d", recording.DeploymentID(), recording.Version()))
		return
	}

	cmdio.LogString(ctx, "Current Deployment Version: "+workspaceurls.DeploymentURL(*baseURL, recording.DeploymentID(), recording.Version()))
}

// deploymentMetadata describes the bundle this deploy came from and where it
// landed, mirroring what bundle/deploy/metadata computes for the metadata file.
func deploymentMetadata(b *bundle.Bundle) dms.Metadata {
	p := dms.Metadata{
		DisplayName: b.Config.Bundle.Name,
		TargetName:  b.Config.Bundle.Target,
		Mode:        deploymentModeToSDK(b.Config.Bundle.Mode),
	}

	git := b.Config.Bundle.Git
	if git.Branch != "" || git.Commit != "" || git.OriginURL != "" {
		p.Git = &bundledeployments.GitInfo{
			Branch:    git.Branch,
			Commit:    git.Commit,
			OriginUrl: git.OriginURL,
		}
	}

	ws := &bundledeployments.WorkspaceInfo{
		RootPath: b.Config.Workspace.RootPath,
		FilePath: b.Config.Workspace.FilePath,
	}
	// In a source-linked deployment files are not copied, so resources read them
	// from the sync root instead of file_path (see bundle/deploy/metadata.Compute).
	if config.IsExplicitlyEnabled(b.Config.Presets.SourceLinkedDeployment) {
		ws.FilePath = b.SyncRootPath
		ws.SourceLinked = true
	}
	// Only a deploy from a Databricks Git folder has one; a local worktree does not.
	// bundle_root_path is relative to it, so the service requires both or neither.
	if b.WorktreeRoot != nil && strings.HasPrefix(b.WorktreeRoot.Native(), "/Workspace/") {
		ws.GitFolderPath = b.WorktreeRoot.Native()
		ws.BundleRootPath = git.BundleRootPath
	}
	p.Workspace = ws

	return p
}

// deploymentModeToSDK maps the bundle target's mode to the DMS enum. An unset mode
// maps to empty, which the service reads as "not reported".
func deploymentModeToSDK(mode config.Mode) bundledeployments.DeploymentMode {
	switch mode {
	case config.Development:
		return bundledeployments.DeploymentModeDeploymentModeDevelopment
	case config.Production:
		return bundledeployments.DeploymentModeDeploymentModeProduction
	default:
		return ""
	}
}
