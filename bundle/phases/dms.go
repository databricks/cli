package phases

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/engine"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/dms"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/workspaceurls"
	"github.com/databricks/databricks-sdk-go/client"
	"github.com/databricks/databricks-sdk-go/service/bundledeployments"
)

// newDeploymentRecorder returns a dms.Recorder for the current deployment, or
// nil when DMS recording does not apply. A nil recorder is a no-op, so callers
// do not need to branch on it.
//
// Recording is enabled only when experimental.record_deployment_history is set
// AND the engine is direct: DMS resource state is tracked per direct-engine
// deployment. Returning nil for terraform leaves those deployments untouched.
//
// The deployment ID is resolved from the workspace, not local state (see
// dms.ResolveDeploymentID). The lookup happens here, after the deployment lock is
// held, so it sees any deployment a concurrent deploy created. It is empty on the
// first recorded deploy, where the recorder creates the deployment instead.
func newDeploymentRecorder(ctx context.Context, b *bundle.Bundle, eng engine.EngineType, versionType dms.VersionType) (*dms.Recorder, error) {
	if b.Config.Experimental == nil || !b.Config.Experimental.RecordDeploymentHistory {
		return nil, nil
	}
	if !eng.IsDirect() {
		return nil, nil
	}

	statePath := b.Config.Workspace.StatePath
	deploymentID, err := dms.ResolveDeploymentID(ctx, b.WorkspaceClient(ctx), statePath)
	if err != nil {
		return nil, err
	}
	apiClient, err := client.New(b.WorkspaceClient(ctx).Config)
	if err != nil {
		return nil, err
	}
	return dms.NewRecorder(dms.RecorderOptions{
		Service:      b.WorkspaceClient(ctx).BundleDeployments,
		Versions:     dms.NewAPIVersionCreator(apiClient),
		DeploymentID: deploymentID,
		StatePath:    statePath,
		TargetName:   b.Config.Bundle.Target,
		DisplayName:  b.Config.Bundle.Name,
		VersionType:  versionType,
		Provenance:   deploymentProvenance(b),
	}), nil
}

// logDeploymentHistory links to the deployment this deploy was recorded under, so
// the user can open its history without hunting for the ID. A nil recorder means
// recording is off, and a zero version means the version was never created.
//
// The workspace ID is left out of the URL: the page redirects correctly without it,
// and omitting it keeps the line short enough to stay clickable in a terminal.
func logDeploymentHistory(ctx context.Context, b *bundle.Bundle, recorder *dms.Recorder) {
	if recorder == nil || recorder.Version() == 0 {
		return
	}

	baseURL, err := url.Parse(b.WorkspaceClient(ctx).Config.CanonicalHostName())
	if err != nil {
		// Only the link is lost, so report the deployment without it rather than
		// failing a deploy that already succeeded.
		log.Debugf(ctx, "Not linking to the recorded deployment: %s", err)
		cmdio.LogString(ctx, fmt.Sprintf("Recorded deployment %s version %d", recorder.DeploymentID(), recorder.Version()))
		return
	}

	cmdio.LogString(ctx, "Deployment history: "+workspaceurls.DeploymentURL(*baseURL, recorder.DeploymentID(), recorder.Version()))
}

// deploymentProvenance describes the source this deploy came from and where it
// landed, mirroring what bundle/deploy/metadata computes for the metadata file.
func deploymentProvenance(b *bundle.Bundle) dms.Provenance {
	p := dms.Provenance{Mode: deploymentModeToSDK(b.Config.Bundle.Mode)}

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
