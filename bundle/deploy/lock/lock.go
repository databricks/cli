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

// DeploymentManager controls the versioned lifecycle of deployment operations.
//
// DMS semantics: CreateVersion atomically succeeds only if no other deployment
// is in progress and the returned version is exactly +1 to the latest closed
// version, providing serialized optimistic concurrency control. CloseVersion
// records the outcome.
//
// Workspace-filesystem semantics: CreateVersion acquires the workspace lock
// file; CloseVersion releases it. The returned version number is a placeholder
// (the lock file does not track a monotonic counter today).
type DeploymentManager interface {
	// CreateVersion begins a new deployment for the given goal.
	// Returns the version number assigned by the backend.
	CreateVersion(ctx context.Context, goal Goal) (int64, error)

	// CloseVersion finalizes the deployment version created by CreateVersion.
	CloseVersion(ctx context.Context, version int64, status DeploymentStatus) error
}

// NewDeploymentManager returns a DeploymentManager backed by the workspace
// filesystem. Captures everything it needs from the bundle at construction time
// so the implementation does not retain a *bundle.Bundle reference. The
// workspace client is only initialized when locking is enabled to match the
// original lazy-init behavior.
func NewDeploymentManager(ctx context.Context, b *bundle.Bundle) DeploymentManager {
	enabled := b.Config.Bundle.Deployment.Lock.IsEnabled()
	l := &workspaceFilesystemLock{
		user:      b.Config.Workspace.CurrentUser.UserName,
		statePath: b.Config.Workspace.StatePath,
		enabled:   enabled,
		force:     b.Config.Bundle.Deployment.Lock.Force,
		reportPermissionError: func(ctx context.Context, path string) diag.Diagnostics {
			return permissions.ReportPossiblePermissionDenied(ctx, b, path)
		},
	}
	if enabled {
		l.client = b.WorkspaceClient(ctx)
	}
	return l
}
