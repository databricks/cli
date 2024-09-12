package lock

import (
	"context"
	"errors"
	"io/fs"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/permissions"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/cli/libs/locker"
	"github.com/databricks/cli/libs/log"
)

type acquire struct{}

func Acquire() bundle.Mutator {
	return &acquire{}
}

func (m *acquire) Name() string {
	return "lock:acquire"
}

func (m *acquire) init(b *bundle.Bundle) error {
	user := b.Config.Workspace.CurrentUser.UserName
	dir := b.Config.Workspace.StatePath
	l, err := locker.CreateLocker(user, dir, b.WorkspaceClient())
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

	err := m.init(b)
	if err != nil {
		return diag.FromErr(err)
	}

	force := b.Config.Bundle.Deployment.Lock.Force
	log.Infof(ctx, "Acquiring deployment lock (force: %v)", force)
	err = b.Locker.Lock(ctx, force)
	if err != nil {
		log.Errorf(ctx, "Failed to acquire deployment lock: %v", err)

		if errors.Is(err, fs.ErrPermission) {
			return permissions.ReportPossiblePermissionDenied(ctx, b, b.Config.Workspace.StatePath)
		}

		notExistsError := filer.NoSuchDirectoryError{}
		if errors.As(err, &notExistsError) {
			// If we get a "doesn't exist" error from the API this indicates
			// we either don't have permissions or the path is invalid.
			return permissions.ReportPossiblePermissionDenied(ctx, b, b.Config.Workspace.StatePath)
		}

		return diag.FromErr(err)
	}

	return nil
}
