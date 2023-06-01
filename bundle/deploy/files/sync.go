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

	opts := sync.SyncOptions{
		LocalPath:  b.Config.Path,
		RemotePath: b.Config.Workspace.FilesPath,
		Full:       false,

		SnapshotBasePath: cacheDir,
		WorkspaceClient:  b.WorkspaceClient(),

		PersistSnapshot: true,
		AllowOverwrites: true,
	}
	return sync.New(ctx, opts)
}
