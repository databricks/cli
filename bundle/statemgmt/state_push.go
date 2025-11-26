package statemgmt

import (
	"context"
	"errors"
	"io/fs"
	"os"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/engine"
	"github.com/databricks/cli/bundle/deploy"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/logdiag"
)

// PushResourcesState uploads the local state file to the remote location.
func PushResourcesState(ctx context.Context, b *bundle.Bundle, engine engine.EngineType) {
	f, err := deploy.StateFiler(b)
	if err != nil {
		logdiag.LogError(ctx, err)
		return
	}

	var remotePath, localPath string

	if engine.IsDirect() {
		remotePath, localPath = b.StateFilenameDirect(ctx)
	} else {
		remotePath, localPath = b.StateFilenameTerraform(ctx)
	}

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
	cmdio.LogString(ctx, "Updating deployment state...")
	err = f.Write(ctx, remotePath, local, filer.CreateParentDirectories, filer.OverwriteIfExists)
	if err != nil {
		logdiag.LogError(ctx, err)
	}
}

func BackupRemoteTerraformState(ctx context.Context, b *bundle.Bundle) {
	f, err := deploy.StateFiler(b)
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
	err = f.Write(ctx, backupPath, reader)
	if err != nil {
		log.Warnf(ctx, "backing up terraform state: could not write %s: %s", backupPath, err)
		return
	}

	err = f.Delete(ctx, remotePath)
	if err != nil {
		log.Warnf(ctx, "backing up terraform state: could not delete %s: %s", remotePath, err)
	}
}
