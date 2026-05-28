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

// DeploymentLock manages the deployment lock lifecycle.
type DeploymentLock interface {
	// Acquire acquires the deployment lock.
	Acquire(ctx context.Context) error

	// Release releases the deployment lock with the given deployment status.
	Release(ctx context.Context, status DeploymentStatus) error
}

// NewDeploymentLock returns a DeploymentLock backed by the workspace
// filesystem. Captures everything the lock needs from the bundle at
// construction time so the lock implementation itself does not retain a
// *bundle.Bundle reference.
func NewDeploymentLock(ctx context.Context, b *bundle.Bundle, goal Goal) DeploymentLock {
	return &workspaceFilesystemLock{
		client:    b.WorkspaceClient(ctx),
		user:      b.Config.Workspace.CurrentUser.UserName,
		statePath: b.Config.Workspace.StatePath,
		enabled:   b.Config.Bundle.Deployment.Lock.IsEnabled(),
		force:     b.Config.Bundle.Deployment.Lock.Force,
		goal:      goal,
		reportPermissionError: func(ctx context.Context, path string) diag.Diagnostics {
			return permissions.ReportPossiblePermissionDenied(ctx, b, path)
		},
	}
}
