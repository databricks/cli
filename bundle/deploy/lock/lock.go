package lock

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/env"
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

// NewDeploymentLock returns a DeploymentLock implementation chosen based on
// the DATABRICKS_BUNDLE_MANAGED_STATE environment variable: when set to a
// truthy value the deployment metadata service backs the lock, otherwise the
// historical workspace-filesystem lock is used.
//
// Note: today the env var alone gates the DMS path. Once the broader managed-
// state feature lands the gate will move behind a richer predicate (e.g.
// statemgmt.IsDmsActive) that also checks server-side opt-in.
func NewDeploymentLock(ctx context.Context, b *bundle.Bundle, goal Goal) (DeploymentLock, error) {
	if env.IsManagedState(ctx) {
		return newMetadataServiceLock(b, goal)
	}
	return newWorkspaceFilesystemLock(b, goal), nil
}
