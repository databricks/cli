package lock

import (
	"context"
	"errors"
	"io/fs"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/permissions"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/locker"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go/apierr"
)

// maxChildNodeSizeExceeded is the error_code the Workspace API returns (as a 403)
// when a directory is at its child-node limit and cannot accept a new child.
// The SDK has no sentinel for it, so we match on the code directly.
const maxChildNodeSizeExceeded = "MAX_CHILD_NODE_SIZE_EXCEEDED"

type acquire struct {
	goal Goal
}

func Acquire(goal Goal) bundle.Mutator {
	return &acquire{goal}
}

func (m *acquire) Name() string {
	return "lock:acquire"
}

func (m *acquire) init(ctx context.Context, b *bundle.Bundle) error {
	user := b.Config.Workspace.CurrentUser.UserName
	dir := b.Config.Workspace.StatePath
	l, err := locker.CreateLocker(user, dir, b.WorkspaceClient(ctx))
	if err != nil {
		return err
	}

	b.Locker = l
	return nil
}

func (m *acquire) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	// Return early if locking is disabled.
	if !b.Config.Bundle.Deployment.Lock.IsEnabled() {
		log.Infof(ctx, "Skipping; locking is disabled")
		return nil
	}

	err := m.init(ctx, b)
	if err != nil {
		return diag.FromErr(err)
	}

	force := b.Config.Bundle.Deployment.Lock.Force
	log.Infof(ctx, "Acquiring deployment lock (force: %v)", force)
	err = b.Locker.Lock(ctx, force)
	if err != nil {
		// When destroying with --force-lock, tolerate a full workspace directory
		// that cannot accept the lock file: proceeding lock-less carries the same
		// caveat --force-lock already documents (no guaranteed exclusive access),
		// which is acceptable when the goal is to tear the deployment down.
		// This check must precede the fs.ErrPermission branch below because the API
		// reports the child-node limit as a 403, which the filer maps to that error.
		if m.goal == GoalDestroy && force {
			if aerr, ok := errors.AsType[*apierr.APIError](err); ok && aerr.ErrorCode == maxChildNodeSizeExceeded {
				log.Warnf(ctx, "Proceeding with destroy without a deployment lock: %v", err)
				return nil
			}
		}

		log.Errorf(ctx, "Failed to acquire deployment lock: %v", err)

		if errors.Is(err, fs.ErrPermission) {
			return permissions.ReportPossiblePermissionDenied(ctx, b, b.Config.Workspace.StatePath)
		}

		if errors.Is(err, fs.ErrNotExist) {
			// If we get a "doesn't exist" error from the API this indicates
			// we either don't have permissions or the path is invalid.
			return permissions.ReportPossiblePermissionDenied(ctx, b, b.Config.Workspace.StatePath)
		}

		return diag.FromErr(err)
	}

	return nil
}
