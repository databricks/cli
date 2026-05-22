package lock

import (
	"context"

	"github.com/databricks/cli/bundle"
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
// filesystem. This factory exists so a future change can swap in alternative
// lock implementations without touching callers.
func NewDeploymentLock(b *bundle.Bundle, goal Goal) DeploymentLock {
	return newWorkspaceFilesystemLock(b, goal)
}
