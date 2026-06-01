package lock

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/permissions"
	"github.com/databricks/cli/libs/diag"
)

// Goal describes the purpose of a deployment operation.
type Goal string

const (
	GoalBind    = Goal("bind")
	GoalUnbind  = Goal("unbind")
	GoalDeploy  = Goal("deploy")
	GoalDestroy = Goal("destroy")
)

// DeploymentStatus indicates whether the deployment operation succeeded or failed.
type DeploymentStatus int

const (
	DeploymentSuccess DeploymentStatus = iota
	DeploymentFailure
)

// DeploymentLock manages the lifecycle of a bundle deployment.
// The workspace-filesystem lock always serializes concurrent deployments.
// DMS version tracking is additive: CreateVersion is called after acquiring
// the file lock, CompleteVersion before releasing it.
type DeploymentLock struct {
	wfs workspaceFilesystemLock
	// dms is nil until wired in via NewDeploymentLock.
	dms *metadataServiceLock
}

// NewDeploymentLock returns a DeploymentLock for the bundle.
// Captures everything it needs from the bundle at construction time
// so the lock does not retain a *bundle.Bundle reference. The
// workspace client is only initialized when locking is enabled.
func NewDeploymentLock(ctx context.Context, b *bundle.Bundle, goal Goal) *DeploymentLock {
	enabled := b.Config.Bundle.Deployment.Lock.IsEnabled()
	l := &DeploymentLock{
		wfs: workspaceFilesystemLock{
			user:      b.Config.Workspace.CurrentUser.UserName,
			statePath: b.Config.Workspace.StatePath,
			enabled:   enabled,
			force:     b.Config.Bundle.Deployment.Lock.Force,
			goal:      goal,
			reportPermissionError: func(ctx context.Context, path string) diag.Diagnostics {
				return permissions.ReportPossiblePermissionDenied(ctx, b, path)
			},
		},
	}
	if enabled {
		l.wfs.client = b.WorkspaceClient(ctx)
	}
	if b.Config.Experimental != nil && b.Config.Experimental.RecordDeploymentHistory {
		l.dms = newMetadataServiceLock(b, goal)
	}
	return l
}

// Acquire acquires the deployment lock.
// The workspace-filesystem lock is always acquired first.
// When DMS is enabled, CreateVersion is called after the file lock so the
// deployment is also registered server-side.
//
// Optimization: once managed_service.json exists (deployment already tracked
// by DMS), the file lock could be skipped — DMS CreateVersion provides
// equivalent concurrency control via the server-side version counter.
func (l *DeploymentLock) Acquire(ctx context.Context) error {
	if err := l.wfs.acquire(ctx); err != nil {
		return err
	}
	if l.dms != nil {
		return l.dms.createVersion(ctx)
	}
	return nil
}

// Release releases the deployment lock.
// When DMS is enabled, CompleteVersion is called before releasing the file lock.
func (l *DeploymentLock) Release(ctx context.Context, status DeploymentStatus) error {
	if l.dms != nil {
		if err := l.dms.completeVersion(ctx, status); err != nil {
			return err
		}
	}
	return l.wfs.release(ctx, status)
}
