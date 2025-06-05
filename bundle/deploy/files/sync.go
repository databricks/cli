package files

import (
	"context"
	"fmt"
	"slices"

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

var defaultExcludes = []string{
	"__pycache__/",
	"^build/",
	"^dist/",
}

func GetSyncOptions(ctx context.Context, b *bundle.Bundle) (*sync.SyncOptions, error) {
	cacheDir, err := b.CacheDir(ctx)
	if err != nil {
		return nil, fmt.Errorf("cannot get bundle cache directory: %w", err)
	}

	includes, err := b.GetSyncIncludePatterns(ctx)
	if err != nil {
		return nil, fmt.Errorf("cannot get list of sync includes: %w", err)
	}

	// We used to delete __pycache__ and build and most of the dist, so now we're excluding it manually

	var excludes []string

	for _, defaultExclude := range defaultExcludes {
		if slices.Contains(includes, defaultExclude) {
			continue
		}
		if slices.Contains(b.Config.Sync.Exclude, defaultExclude) {
			continue
		}
		excludes = append(excludes, defaultExclude)
	}

	excludes = append(b.Config.Sync.Exclude, excludes...)

	// TODO: if users include those manually, then we should not exclude it?

	opts := &sync.SyncOptions{
		WorktreeRoot: b.WorktreeRoot,
		LocalRoot:    b.SyncRoot,
		Paths:        b.Config.Sync.Paths,
		Include:      includes,
		Exclude:      excludes,

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
