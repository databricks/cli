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

	removeBundleNameDir(ctx, b)

	// Clean up sync snapshot file
	err = deleteSnapshotFile(ctx, b)
	if err != nil {
		return diag.FromErr(err)
	}
	return nil
}

// removeBundleNameDir removes the directory named after the bundle now that the
// deployment under it is gone. Not recursive, so it fails harmlessly while another
// target is still deployed there.
func removeBundleNameDir(ctx context.Context, b *bundle.Bundle) {
	if !b.RootPathIsNameTargetScoped {
		return
	}

	dir := path.Dir(b.Config.Workspace.RootPath)
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
