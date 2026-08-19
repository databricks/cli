package files

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/sync"
	"github.com/databricks/databricks-sdk-go/service/workspace"
)

const bundleDirName = ".bundle"

type delete struct{}

func (m *delete) Name() string {
	return "files.Delete"
}

func (m *delete) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	err := b.WorkspaceClient(ctx).Workspace.Delete(ctx, workspace.Delete{
		Path:      b.Config.Workspace.RootPath,
		Recursive: true,
	})
	if err != nil {
		return diag.FromErr(err)
	}

	removeEmptyBundleDir(ctx, b)

	// Clean up sync snapshot file
	err = deleteSnapshotFile(ctx, b)
	if err != nil {
		return diag.FromErr(err)
	}
	return nil
}

// removeEmptyBundleDir removes the ~/.bundle/<name> directory left behind once the
// target subdirectory under it is gone. The delete is not recursive, so it only
// succeeds while no other target of the bundle is still deployed there; both a
// remaining sibling target and an already-removed directory surface as an error that
// is expected and ignored.
func removeEmptyBundleDir(ctx context.Context, b *bundle.Bundle) {
	dir := path.Dir(b.Config.Workspace.RootPath)

	// root_path is user-configurable, so only clean up the layout the CLI generates
	// (~/.bundle/<name>/<target>) instead of deleting the parent of an arbitrary path.
	if path.Base(path.Dir(dir)) != bundleDirName {
		return
	}

	err := b.WorkspaceClient(ctx).Workspace.Delete(ctx, workspace.Delete{Path: dir})
	if err != nil {
		log.Debugf(ctx, "Leaving %s in place: %s", dir, err)
	}
}

func deleteSnapshotFile(ctx context.Context, b *bundle.Bundle) error {
	opts, err := GetSyncOptions(ctx, b)
	if err != nil {
		return fmt.Errorf("cannot get sync options: %w", err)
	}
	sp, err := sync.SnapshotPath(opts)
	if err != nil {
		return err
	}
	err = os.Remove(sp)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("failed to destroy sync snapshot file: %s", err)
	}
	return nil
}

func Delete() bundle.Mutator {
	return &delete{}
}
