package files

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/sync"
)

func GetSync(ctx context.Context, b *bundle.Bundle) (*sync.Sync, error) {
	opts, err := GetSyncOptions(ctx, b)
	if err != nil {
		return nil, fmt.Errorf("cannot get sync options: %w", err)
	}
	return sync.New(ctx, *opts)
}

func GetSyncOptions(ctx context.Context, b *bundle.Bundle) (*sync.SyncOptions, error) {
	cacheDir, err := b.LocalStateDir(ctx)
	if err != nil {
		return nil, fmt.Errorf("cannot get bundle cache directory: %w", err)
	}

	includes, err := b.GetSyncIncludePatterns(ctx)
	if err != nil {
		return nil, fmt.Errorf("cannot get list of sync includes: %w", err)
	}

	opts := &sync.SyncOptions{
		WorktreeRoot: b.WorktreeRoot,
		LocalRoot:    b.SyncRoot,
		Paths:        b.Config.Sync.Paths,
		Include:      includes,
		Exclude:      b.Config.Sync.Exclude,

		RemotePath: b.Config.Workspace.FilePath,
		Host:       b.WorkspaceClient().Config.Host,

		Full: false,

		SnapshotBasePath: cacheDir,
		WorkspaceClient:  b.WorkspaceClient(),
	}

	if b.Config.Workspace.CurrentUser != nil {
		opts.CurrentUser = b.Config.Workspace.CurrentUser.User
	}

	return opts, nil
}
