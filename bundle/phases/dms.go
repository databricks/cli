package phases

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"path"
	"strconv"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/internal/build"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/dms"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/retry"
	"github.com/databricks/cli/libs/workspaceurls"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/common/types/fieldmask"
	"github.com/databricks/databricks-sdk-go/service/bundledeployments"
	"github.com/databricks/databricks-sdk-go/service/workspace"
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
func createOrUpdateDeployment(ctx context.Context, b *bundle.Bundle, current *bundledeployments.Deployment) error {
	db := &b.DeploymentBundle
	w := b.WorkspaceClient(ctx)
	metadata, err := deploymentMetadata(b)
	if err != nil {
		return fmt.Errorf("failed to compute deployment metadata: %w", err)
	}
	deploymentID := db.StateDB.DeploymentID
	if deploymentID == "" {
		dep := metadata.Deployment()
		dep.InitialParentPath = b.Config.Workspace.StatePath
		created, err := w.BundleDeployments.CreateDeployment(ctx, bundledeployments.CreateDeploymentRequest{Deployment: dep})
		if err != nil {
			return fmt.Errorf("failed to create deployment: %w", err)
		}
		// The id is the workspace node the server makes under the parent path.
		deploymentID, err = dms.DeploymentIDFromName(created.Name)
		if err != nil {
			return fmt.Errorf("failed to create deployment: %w", err)
		}
		db.StateDB.DeploymentID = deploymentID
	} else if mask := metadata.StaleFields(current); mask != "" {
		_, err := retry.With(ctx, retry.IsTransient, func() (*bundledeployments.Deployment, error) {
			return w.BundleDeployments.UpdateDeployment(ctx, bundledeployments.UpdateDeploymentRequest{
				Name:       dms.DeploymentName(deploymentID),
				Deployment: dms.DeploymentUpdate(metadata, mask),
				UpdateMask: fieldmask.FieldMask{Paths: strings.Split(mask, ",")},
			})
		})
		if err != nil {
			return fmt.Errorf("failed to update deployment: %w", err)
		}
	}

	// A first deploy had no deployment to read at startup, so its id enters the history here.
	bundle.ApplyFuncContext(ctx, b, func(_ context.Context, b *bundle.Bundle) {
		b.Config.Bundle.Deployment.DeploymentID = deploymentID
	})
	return nil
}

// deleteDeployment deletes the deployment record. A retry that finds it missing succeeds: the
// earlier attempt may have deleted it before its response was lost.
func deleteDeployment(ctx context.Context, w *databricks.WorkspaceClient, deploymentID string) error {
	retried := false
	_, err := retry.With(ctx, retry.IsTransient, func() (struct{}, error) {
		err := w.BundleDeployments.DeleteDeployment(ctx, bundledeployments.DeleteDeploymentRequest{Name: dms.DeploymentName(deploymentID)})
		if retried && apierr.IsMissing(err) {
			return struct{}{}, nil
		}
		retried = true
		return struct{}{}, err
	})
	return err
}

// prepareDeploymentHistory creates history metadata and a version after approval.
func prepareDeploymentHistory(ctx context.Context, b *bundle.Bundle, plan *deployplan.Plan, current *bundledeployments.Deployment) error {
	firstDeploy := b.DeploymentBundle.StateDB.DeploymentID == ""
	if err := createOrUpdateDeployment(ctx, b, current); err != nil {
		return err
	}
	if firstDeploy {
		if err := b.DeploymentBundle.StampDeploymentIdForFirstVersion(b.DeploymentBundle.StateDB.DeploymentID); err != nil {
			return err
		}
	}
	staged, err := stagedOperations(plan)
	if err != nil {
		return err
	}
	if len(staged) == 0 {
		return nil
	}
	if err := startVersion(ctx, b, dms.VersionTypeDeploy, staged); err != nil {
		return err
	}
	logDeploymentVersion(ctx, b)
	return nil
}

// prepareDestroyHistory records the approved destroy without changing its state source.
func prepareDestroyHistory(ctx context.Context, b *bundle.Bundle, plan *deployplan.Plan) error {
	if !b.DeploymentBundle.StateDB.IsDeploymentMetadataService() && b.DeploymentBundle.StateDB.DeploymentID == "" {
		if err := createOrUpdateDeployment(ctx, b, nil); err != nil {
			return err
		}
	}
	staged, err := stagedOperations(plan)
	if err != nil {
		return err
	}
	return startVersion(ctx, b, dms.VersionTypeDestroy, staged)
}

// startVersion opens a history version after approval. File-backed history skips empty versions;
// DMS-only destroy retains its existing behavior. A declined deploy never claims a version.
// A no-op when the bundle does not record deployment history.
func startVersion(ctx context.Context, b *bundle.Bundle, versionType dms.VersionType, staged []bundledeployments.StagedOperation) error {
	db := &b.DeploymentBundle
	if !db.StateDB.RecordsHistory() {
		return nil
	}
	if len(staged) == 0 && !db.StateDB.IsDeploymentMetadataService() {
		return nil
	}
	w := b.WorkspaceClient(ctx)
	deploymentID := db.StateDB.DeploymentID
	// This run's version follows the one the state is anchored to, which is empty for the first.
	// InitializeOperationBuffer moves the anchor below, so callers afterwards read the created
	// version straight off the state.
	previous := db.StateDB.VersionID
	versionID := max(previous, db.StateDB.Data.Serial) + 1
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
	version, err := w.BundleDeployments.CreateVersion(ctx, bundledeployments.CreateVersionRequest{
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

	db.StateDB.InitializeOperationBuffer(ctx, deploymentID, versionID, len(staged))
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
func deploymentMetadata(b *bundle.Bundle) (dms.Metadata, error) {
	p := dms.Metadata{
		DisplayName: b.Config.Bundle.Name,
		TargetName:  b.Config.Bundle.Target,
		Mode:        deploymentModeToSDK(b.Config.Bundle.Mode),
	}

	ws := &bundledeployments.WorkspaceInfo{
		RootPath: b.Config.Workspace.RootPath,
		FilePath: b.Config.Workspace.FilePath,
	}
	// With an immutable folder, file_path is a reference to the snapshot, which only
	// resolves once the snapshot is uploaded. Its path is already known from the staged zip.
	if b.IsImmutableFolder() {
		snapshot, ok := b.Config.Resources.Snapshots[resources.SnapshotResourceKey]
		if !ok {
			return dms.Metadata{}, fmt.Errorf("no %s in the bundle", resources.SnapshotKey)
		}
		ws.FilePath = path.Join(snapshot.FullPath(), "files")
	}
	// In a source-linked deployment files are not copied, so resources read them
	// from the sync root instead of file_path (see bundle/deploy/metadata.Compute).
	if b.IsSourceLinkedDeployment() {
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
	return p, nil
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

// LookupDeployment reads the deployment metadata for a bundle.
//
// Lookup and return the deployment object from ${workspace.state_path}/resources.deployment.json
//
// TODO: a deployment is only usable when both the node and the service's record exist, and a
// half-created one - node present, record missing - blocks the bundle here even though
// CreateDeployment already recovers from it. Move this behind a dms.ReadDeployment(ctx, statePath)
// that returns an empty id and version unless both halves are there, leaving the caller to call
// CreateDeployment to create or finalize it.
//
// TODO: ask the service for a lookup by state path, so this is one round trip rather than two - a
// workspace lookup to turn the node into an id, then a get by that id.
func LookupDeployment(ctx context.Context, w *databricks.WorkspaceClient, statePath string) (string, *bundledeployments.Deployment, error) {
	nodePath := path.Join(statePath, dms.DeploymentNodeName)

	obj, err := retry.With(ctx, retry.IsTransientRead, func() (*workspace.ObjectInfo, error) {
		return w.Workspace.GetStatusByPath(ctx, nodePath)
	})
	if errors.Is(err, apierr.ErrNotFound) || errors.Is(err, apierr.ErrResourceDoesNotExist) {
		return "", nil, nil
	}
	if err != nil {
		return "", nil, fmt.Errorf("looking up deployment at %s: %w", nodePath, err)
	}
	deploymentID := strconv.FormatInt(obj.ObjectId, 10)
	deployment, err := retry.With(ctx, retry.IsTransientRead, func() (*bundledeployments.Deployment, error) {
		return w.BundleDeployments.GetDeployment(ctx, bundledeployments.GetDeploymentRequest{
			Name: dms.DeploymentName(deploymentID),
		})
	})
	if err != nil {
		return "", nil, err
	}
	return deploymentID, deployment, nil
}
