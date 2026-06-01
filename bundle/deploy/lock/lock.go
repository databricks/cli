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
// The workspace-filesystem lock serializes concurrent deployments.
// DMS version tracking will be added additively — see deployment_metadata_service.go.
type DeploymentLock struct {
	wfs workspaceFilesystemLock
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
	return l
}

// Acquire acquires the deployment lock.
func (l *DeploymentLock) Acquire(ctx context.Context) error {
	return l.wfs.acquire(ctx)
}

// Release releases the deployment lock.
func (l *DeploymentLock) Release(ctx context.Context, status DeploymentStatus) error {
	return l.wfs.release(ctx, status)
}
