package phases

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/internal/build"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/dms"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/libs/workspaceurls"
	"github.com/databricks/databricks-sdk-go/common/types/fieldmask"
	"github.com/databricks/databricks-sdk-go/service/bundledeployments"
)

// stagedOperations lists the resources the plan will touch, for CreateVersion to stage an
// operation each. Skipped and undefined actions are left out: nothing is applied for them, so
// their operations would stay pending and the service would hold no state for them.
func stagedOperations(plan *deployplan.Plan) ([]bundledeployments.StagedOperation, error) {
	actions := plan.GetActions()
	staged := make([]bundledeployments.StagedOperation, 0, len(actions))
	for _, action := range actions {
		if action.ActionType == deployplan.Skip || action.ActionType == deployplan.Undefined {
			continue
		}
		actionType, err := actionToSDK(action.ActionType)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", action.ResourceKey, err)
		}
		staged = append(staged, bundledeployments.StagedOperation{
			ResourceKey: strings.TrimPrefix(action.ResourceKey, dms.StatePrefix),
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

// createOrUpdateDeployment creates the deployment on a first deploy, or updates the metadata this
// run changed. current is the record the service holds (nil before the first recorded deploy),
// diffed to mask the update down to what changed. Runs after approval, so a declined deploy leaves
// no deployment behind; a first deploy's new id is then stamped into the plan (StampDeploymentID).
func createOrUpdateDeployment(ctx context.Context, b *bundle.Bundle, current *bundledeployments.Deployment) {
	db := &b.DeploymentBundle
	dmsService := db.StateDB.DmsService()
	metadata := deploymentMetadata(b)
	deploymentID := db.StateDB.DeploymentID
	if deploymentID == "" {
		dep := metadata.Deployment()
		dep.InitialParentPath = b.Config.Workspace.StatePath
		created, err := dmsService.CreateDeployment(ctx, bundledeployments.CreateDeploymentRequest{Deployment: dep})
		if err != nil {
			logdiag.LogError(ctx, fmt.Errorf("failed to create deployment: %w", err))
			return
		}
		// The server assigns the id as the workspace node it creates under the parent path.
		deploymentID, err = dms.DeploymentIDFromName(created.Name)
		if err != nil {
			logdiag.LogError(ctx, fmt.Errorf("failed to create deployment: %w", err))
			return
		}
		db.StateDB.DeploymentID = deploymentID
	} else if mask := metadata.StaleFields(current); mask != "" {
		_, err := dmsService.UpdateDeployment(ctx, bundledeployments.UpdateDeploymentRequest{
			Name:       dms.DeploymentName(deploymentID),
			Deployment: dms.DeploymentUpdate(metadata, mask),
			UpdateMask: fieldmask.FieldMask{Paths: strings.Split(mask, ",")},
		})
		if err != nil {
			logdiag.LogError(ctx, fmt.Errorf("failed to update deployment: %w", err))
			return
		}
	}

	// A first deploy had no deployment to read at startup, so its id enters the history here.
	bundle.ApplyFuncContext(ctx, b, func(_ context.Context, b *bundle.Bundle) {
		b.Config.Bundle.Deployment.DeploymentID = deploymentID
	})
}

// startVersion claims the version the run settled on and opens the buffer that records
// each state write under it. Called after approval, so a declined deploy never claims a number.
// A no-op when the bundle does not record deployment history.
func startVersion(ctx context.Context, b *bundle.Bundle, versionType dms.VersionType, staged []bundledeployments.StagedOperation) error {
	db := &b.DeploymentBundle
	dmsService := db.StateDB.DmsService()
	if dmsService == nil {
		return nil
	}
	deploymentID := db.StateDB.DeploymentID
	// This run's version follows the one the state is anchored to, which is empty for the first.
	// InitializeOperationBuffer moves the anchor below, so callers afterwards read the created
	// version straight off the state.
	previous := db.StateDB.VersionID
	versionID := previous + 1
	previousVersionID := ""
	if previous > 0 {
		previousVersionID = strconv.Itoa(previous)
	}

	// The server rejects this unless the version number exceeds last_version_id and
	// previous_version_id matches it, which is what makes claiming the number up front
	// safe: a deploy that took it in the meantime is reported, not overwritten.
	var gitInfo *bundledeployments.GitInfo
	if git := b.Config.Bundle.Git; git.Branch != "" || git.Commit != "" || git.OriginURL != "" {
		gitInfo = &bundledeployments.GitInfo{
			Branch:    git.Branch,
			Commit:    git.Commit,
			OriginUrl: git.OriginURL,
		}
	}
	version, err := dmsService.CreateVersion(ctx, bundledeployments.CreateVersionRequest{
		Parent:    dms.DeploymentName(deploymentID),
		VersionId: strconv.Itoa(versionID),
		Version: bundledeployments.Version{
			CliVersion:        build.GetInfo().Version,
			VersionType:       versionType,
			PreviousVersionId: previousVersionID,
			GitInfo:           gitInfo,
			Operations:        staged,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create deployment version: %w", err)
	}
	log.Infof(ctx, "Created deployment version: deployment=%s version=%s", deploymentID, version.VersionId)

	db.StateDB.InitializeOperationBuffer(ctx, deploymentID, versionID)
	return nil
}

// logDeploymentVersion logs the deployment version URL. Workspace ID is omitted
// so the page stays clickable in a terminal and redirects correctly without it.
func logDeploymentVersion(ctx context.Context, b *bundle.Bundle) {
	deploymentID := b.DeploymentBundle.StateDB.DeploymentID
	version := b.DeploymentBundle.StateDB.VersionID
	if version == 0 {
		return
	}

	baseURL, err := url.Parse(b.WorkspaceClient(ctx).Config.CanonicalHostName())
	if err != nil {
		// Only the link is lost, so report the version without it rather than failing
		// a deploy over it.
		log.Debugf(ctx, "Not linking to the recorded deployment: %s", err)
		cmdio.LogString(ctx, fmt.Sprintf("Current Deployment Version: %s version %d", deploymentID, version))
		return
	}

	cmdio.LogString(ctx, "Current Deployment Version: "+workspaceurls.DeploymentURL(*baseURL, deploymentID, version))
}

// deploymentMetadata describes the bundle this deploy came from and where it
// landed, mirroring what bundle/deploy/metadata computes for the metadata file.
func deploymentMetadata(b *bundle.Bundle) dms.Metadata {
	p := dms.Metadata{
		DisplayName: b.Config.Bundle.Name,
		TargetName:  b.Config.Bundle.Target,
		Mode:        deploymentModeToSDK(b.Config.Bundle.Mode),
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
		ws.BundleRootPath = b.Config.Bundle.Git.BundleRootPath
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
