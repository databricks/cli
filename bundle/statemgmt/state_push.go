package statemgmt

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/deploy"
	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/logdiag"
)

// PushResourcesState uploads the local state file to the remote location.
func PushResourcesState(ctx context.Context, b *bundle.Bundle) {
	f, err := deploy.StateFiler(ctx, b)
	if err != nil {
		logdiag.LogError(ctx, err)
		return
	}

	remotePath, localPath := b.StateFilenameDirect(ctx)

	local, err := os.Open(localPath)
	if errors.Is(err, fs.ErrNotExist) {
		// The state file can be absent if terraform apply is skipped because
		// there are no changes to apply in the plan.
		log.Debugf(ctx, "Local state file does not exist: %s", localPath)
		return
	}
	if err != nil {
		logdiag.LogError(ctx, err)
		return
	}
	defer local.Close()

	// Upload state file from local cache directory to filer.
	err = f.Write(ctx, remotePath, local, filer.CreateParentDirectories, filer.OverwriteIfExists)
	if err != nil {
		logdiag.LogError(ctx, err)
	}
}

// BackupRemoteResourcesState writes the selected direct state to resources.json.backup without
// changing the active state. Migration treats any failure as fatal because this is its rollback
// point once DMS has a complete copy.
func BackupRemoteResourcesState(ctx context.Context, b *bundle.Bundle) error {
	f, err := deploy.StateFiler(ctx, b)
	if err != nil {
		return err
	}

	remotePath, localPath := b.StateFilenameDirect(ctx)
	local, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("opening resources state for backup: %w", err)
	}
	defer local.Close()

	backupPath := remotePath + ".backup"
	if err := f.Write(ctx, backupPath, local, filer.CreateParentDirectories, filer.OverwriteIfExists); err != nil {
		return fmt.Errorf("writing resources state backup to %s: %w", backupPath, err)
	}
	return nil
}

func BackupRemoteTerraformState(ctx context.Context, b *bundle.Bundle) {
	f, err := deploy.StateFiler(ctx, b)
	if err != nil {
		logdiag.LogError(ctx, err)
		return
	}

	remotePath, _ := b.StateFilenameTerraform(ctx)
	reader, err := f.Read(ctx, remotePath)

	if errors.Is(err, fs.ErrNotExist) {
		return
	}

	if err != nil {
		log.Warnf(ctx, "backing up terraform state: could not read %s: %s", remotePath, err)
		return
	}

	backupPath := remotePath + ".backup"
	err = f.Write(ctx, backupPath, reader, filer.OverwriteIfExists)
	if err != nil {
		log.Warnf(ctx, "backing up terraform state: could not write %s: %s", backupPath, err)
		return
	}

	err = f.Delete(ctx, remotePath)
	if err != nil {
		log.Warnf(ctx, "backing up terraform state: could not delete %s: %s", remotePath, err)
	}
}
