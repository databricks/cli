package files

import (
	"context"
	"fmt"

	"github.com/databricks/bricks/bundle"
	sync "github.com/databricks/bricks/libs/sync"
)

type upload struct{}

func (m *upload) Name() string {
	return "files.Upload"
}

func (m *upload) Apply(ctx context.Context, b *bundle.Bundle) ([]bundle.Mutator, error) {
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

	sync, err := sync.New(ctx, opts)
	if err != nil {
		return nil, err
	}

	err = sync.RunOnce(ctx)
	if err != nil {
		return nil, err
	}

	return nil, nil
}

func Upload() bundle.Mutator {
	return &upload{}
}
