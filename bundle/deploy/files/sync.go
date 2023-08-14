package files

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/sync"
)

func getSync(ctx context.Context, b *bundle.Bundle) (*sync.Sync, error) {
	cacheDir, err := b.CacheDir()
	if err != nil {
		return nil, fmt.Errorf("cannot get bundle cache directory: %w", err)
	}

	internalDir, err := b.InternalDir()
	if err != nil {
		return nil, fmt.Errorf("cannot get bundle internal directory: %w", err)
	}

	opts := sync.SyncOptions{
		LocalPath:  b.Config.Path,
		RemotePath: b.Config.Workspace.FilesPath,
		Include:    b.Config.Sync.Include,
		Exclude:    b.Config.Sync.Exclude,

		Full:        false,
		CurrentUser: b.Config.Workspace.CurrentUser.User,

		InternalDir:      internalDir,
		SnapshotBasePath: cacheDir,
		WorkspaceClient:  b.WorkspaceClient(),
	}
	return sync.New(ctx, opts)
}
