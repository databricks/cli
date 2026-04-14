package lock

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/env"
	"github.com/databricks/cli/libs/tmpdms"
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

// NewDeploymentLock returns a DeploymentLock implementation based on the
// current environment. If managed state is enabled and the goal maps to a
// supported version type, a metadata service lock is returned. Otherwise,
// a workspace filesystem lock is returned.
func NewDeploymentLock(ctx context.Context, b *bundle.Bundle, goal Goal) DeploymentLock {
	useManagedState, _ := env.ManagedState(ctx)
	if useManagedState == "true" {
		versionType, ok := goalToVersionType(goal)
		if ok {
			return newMetadataServiceLock(b, versionType)
		}
	}
	return newWorkspaceFilesystemLock(b, goal)
}

func goalToVersionType(goal Goal) (tmpdms.VersionType, bool) {
	switch goal {
	case GoalDeploy:
		return tmpdms.VersionTypeDeploy, true
	case GoalDestroy:
		return tmpdms.VersionTypeDestroy, true
	default:
		return "", false
	}
}
