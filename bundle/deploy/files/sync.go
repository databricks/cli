package files

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/sync"
)

func GetSync(ctx context.Context, rb bundle.ReadOnlyBundle) (*sync.Sync, error) {
	opts, err := GetSyncOptions(ctx, rb)
	if err != nil {
		return nil, fmt.Errorf("cannot get sync options: %w", err)
	}
	return sync.New(ctx, *opts)
}

func GetSyncOptions(ctx context.Context, rb bundle.ReadOnlyBundle) (*sync.SyncOptions, error) {
	cacheDir, err := rb.CacheDir(ctx)
	if err != nil {
		return nil, fmt.Errorf("cannot get bundle cache directory: %w", err)
	}

	includes, err := rb.GetSyncIncludePatterns(ctx)
	if err != nil {
		return nil, fmt.Errorf("cannot get list of sync includes: %w", err)
	}

	opts := &sync.SyncOptions{
		WorktreeRoot: rb.WorktreeRoot(),
		LocalRoot:    rb.SyncRoot(),
		Paths:        rb.Config().Sync.Paths,
		Include:      includes,
		Exclude:      rb.Config().Sync.Exclude,

		RemotePath: rb.Config().Workspace.FilePath,
		Host:       rb.WorkspaceClient().Config.Host,

		Full: false,

		SnapshotBasePath: cacheDir,
		WorkspaceClient:  rb.WorkspaceClient(),
	}

	if rb.Config().Workspace.CurrentUser != nil {
		opts.CurrentUser = rb.Config().Workspace.CurrentUser.User
	}

	return opts, nil
}
