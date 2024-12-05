package bundle

import (
	"context"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/vfs"
	"github.com/databricks/databricks-sdk-go"
)

type ReadOnlyBundle struct {
	b *Bundle
}

func ReadOnly(b *Bundle) ReadOnlyBundle {
	return ReadOnlyBundle{b: b}
}

func (r ReadOnlyBundle) Config() config.Root {
	return r.b.Config
}

func (r ReadOnlyBundle) RootPath() string {
	return r.b.BundleRootPath
}

func (r ReadOnlyBundle) BundleRoot() vfs.Path {
	return r.b.BundleRoot
}

func (r ReadOnlyBundle) SyncRoot() vfs.Path {
	return r.b.SyncRoot
}

func (r ReadOnlyBundle) WorktreeRoot() vfs.Path {
	return r.b.WorktreeRoot
}

func (r ReadOnlyBundle) WorkspaceClient() *databricks.WorkspaceClient {
	return r.b.WorkspaceClient()
}

func (r ReadOnlyBundle) CacheDir(ctx context.Context, paths ...string) (string, error) {
	return r.b.CacheDir(ctx, paths...)
}

func (r ReadOnlyBundle) GetSyncIncludePatterns(ctx context.Context) ([]string, error) {
	return r.b.GetSyncIncludePatterns(ctx)
}
