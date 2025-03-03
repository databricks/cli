package files

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/sync"
	"github.com/databricks/databricks-sdk-go/service/workspace"
)

type delete struct{}

func (m *delete) Name() string {
	return "files.Delete"
}

func (m *delete) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	cmdio.LogString(ctx, "Deleting files...")

	err := b.WorkspaceClient().Workspace.Delete(ctx, workspace.Delete{
		Path:      b.Config.Workspace.RootPath,
		Recursive: true,
	})
	if err != nil {
		return diag.FromErr(err)
	}

	// Clean up sync snapshot file
	err = deleteSnapshotFile(ctx, b)
	if err != nil {
		return diag.FromErr(err)
	}
	return nil
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
