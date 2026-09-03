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
	"github.com/databricks/databricks-sdk-go/service/bundledeployments"
)

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
			ResourceKey: action.ResourceKey,
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

// deploymentAndNextVersion reads, from the history the recording set, the deployment id and the version
// this run will create (one past the deployment's most recent). Both zero when nothing is recorded
// - the bundle records no history, so version 0 marks "no version" for logDeploymentVersion.
func deploymentAndNextVersion(b *bundle.Bundle) (string, int64) {
	h := b.Config.Bundle.Deployment.History
	if h == nil {
		return "", 0
	}
	// LatestVersionID comes from the service as an integer, so this does not fail in practice.
	version, err := dms.NextVersion(h.LatestVersionID)
	if err != nil {
		version = 1
	}
	return h.DeploymentID, version
}

// createOrUpdateDeployment creates the deployment on a first deploy, or updates the metadata this
// run changed. current is the record the service holds (nil before the first recorded deploy),
// diffed to mask the update down to what changed. Runs before the plan, so the deployment exists
// to be stamped onto the resources; the version it will create is settled in process.go.
func createOrUpdateDeployment(ctx context.Context, b *bundle.Bundle, current *bundledeployments.Deployment) {
	db := &b.DeploymentBundle
	metadata := deploymentMetadata(b)
	deploymentID, _ := deploymentAndNextVersion(b)
	switch mask := metadata.StaleFields(current); {
	case deploymentID == "":
		id, err := db.DmsApiClient.CreateDeployment(ctx, b.Config.Workspace.StatePath, metadata)
		if err != nil {
			logdiag.LogError(ctx, fmt.Errorf("failed to create deployment: %w", err))
			return
		}
		deploymentID = id
	case mask != "":
		if err := db.DmsApiClient.UpdateDeployment(ctx, deploymentID, metadata, mask); err != nil {
			logdiag.LogError(ctx, fmt.Errorf("failed to update deployment: %w", err))
			return
		}
	}

	// A first deploy had no deployment to read at startup, so its id enters the history here.
	bundle.ApplyFuncContext(ctx, b, func(_ context.Context, b *bundle.Bundle) {
		if b.Config.Bundle.Deployment.History == nil {
			b.Config.Bundle.Deployment.History = &config.DeploymentHistory{}
		}
		b.Config.Bundle.Deployment.History.DeploymentID = deploymentID
	})
}

// startVersion claims the version the run settled on and opens the buffer that records
// each state write under it. Called after approval, so a declined deploy never claims a number.
// A no-op when the bundle does not record deployment history.
func startVersion(ctx context.Context, b *bundle.Bundle, versionType dms.VersionType, staged []dms.StagedOperation) error {
	db := &b.DeploymentBundle
	if db.DmsApiClient == nil {
		return nil
	}
	deploymentID, versionID := deploymentAndNextVersion(b)

	// The version this run creates is one past the deployment's most recent, so the previous is
	// one below it - empty for the first version.
	previousVersionID := ""
	if versionID > 1 {
		previousVersionID = strconv.FormatInt(versionID-1, 10)
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
	version, err := db.DmsApiClient.CreateVersion(ctx, deploymentID, versionID, dms.CreateVersionRequest{
		CliVersion:        build.GetInfo().Version,
		VersionType:       versionType,
		PreviousVersionId: previousVersionID,
		Operations:        staged,
		GitInfo:           gitInfo,
	})
	if err != nil {
		return fmt.Errorf("failed to create deployment version: %w", err)
	}
	log.Infof(ctx, "Created deployment version: deployment=%s version=%s", deploymentID, version.VersionId)

	db.DmsAsyncOperationClient = dms.StartOperationBuffer(ctx, db.DmsApiClient, deploymentID, versionID)
	return nil
}

// drainOperationsAndCompleteVersion drains the buffered operations and completes the version.
// It returns whether the version completed successfully, which a destroy uses to decide whether
// to delete the deployment. Idempotent: the first call clears the buffer, so a deferred
// safety-net after an explicit completion is a no-op returning false. Also false when no version
// was created.
func drainOperationsAndCompleteVersion(ctx context.Context, b *bundle.Bundle, success bool) (bool, error) {
	db := &b.DeploymentBundle
	buf := db.DmsAsyncOperationClient
	if buf == nil {
		return false, nil
	}
	db.DmsAsyncOperationClient = nil

	// Drain first; a recording error fails the deploy even when the resources applied. Its error
	// is left to whoever drained (apply), not returned twice.
	if buf.Drain() != nil {
		success = false
	}

	reason := bundledeployments.VersionCompleteVersionCompleteSuccess
	if !success {
		reason = bundledeployments.VersionCompleteVersionCompleteFailure
	}
	deploymentID, versionID := deploymentAndNextVersion(b)
	if err := db.DmsApiClient.CompleteVersion(ctx, deploymentID, versionID, reason); err != nil {
		return false, err
	}
	log.Infof(ctx, "Completed deployment version: deployment=%s version=%d reason=%s", deploymentID, versionID, reason)

	return success, nil
}

// logDeploymentVersion logs the deployment version URL. Workspace ID is omitted
// so the page stays clickable in a terminal and redirects correctly without it.
func logDeploymentVersion(ctx context.Context, b *bundle.Bundle, deploymentID string, version int64) {
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
