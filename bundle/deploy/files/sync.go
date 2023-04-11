package files

import (
	"context"
	"fmt"

	"github.com/databricks/bricks/bundle"
	"github.com/databricks/bricks/libs/sync"
)

func getSync(ctx context.Context, b *bundle.Bundle) (*sync.Sync, error) {
	cacheDir, err := b.CacheDir()
	if err != nil {
		return nil, fmt.Errorf("cannot get bundle cache directory: %w", err)
	}

	opts := sync.SyncOptions{
		LocalPath:  b.Config.Path,
		RemotePath: b.Config.Workspace.FilePath.Workspace,
		Full:       false,

		SnapshotBasePath: cacheDir,
		WorkspaceClient:  b.WorkspaceClient(),
	}
	return sync.New(ctx, opts)
}
