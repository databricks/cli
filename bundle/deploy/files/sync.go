package files

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/sync"
)

func GetSync(ctx context.Context, b *bundle.Bundle) (*sync.Sync, error) {
	cacheDir, err := b.CacheDir(ctx)
	if err != nil {
		return nil, fmt.Errorf("cannot get bundle cache directory: %w", err)
	}

	includes, err := b.GetSyncIncludePatterns(ctx)
	if err != nil {
		return nil, fmt.Errorf("cannot get list of sync includes: %w", err)
	}

	opts := sync.SyncOptions{
		LocalPath:  b.Config.Path,
		RemotePath: b.Config.Workspace.FilePath,
		Include:    includes,
		Exclude:    b.Config.Sync.Exclude,

		Full:        false,
		CurrentUser: b.Config.Workspace.CurrentUser.User,

		SnapshotBasePath: cacheDir,
		WorkspaceClient:  b.WorkspaceClient(),
	}
	return sync.New(ctx, opts)
}
