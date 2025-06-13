package statemgmt

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/deploy"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/cli/libs/log"
)

// statePush uploads the local state file to the remote filer location.
// A filer implementation is injected to keep the mutator independent from
// the storage backend.
//
// The mutator is a no-op when the local state file does not exist – this is
// the expected scenario when the previous terraform apply was skipped because
// there were no changes.
//
// The mutator intentionally does not attempt to merge state. That logic lives
// in the complimentary statePull mutator.
// A successful push is therefore the last step in the state management
// workflow.

type statePush struct {
	filerFactory deploy.FilerFactory
}

func (l *statePush) Name() string {
	return "state:push"
}

func (l *statePush) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	f, err := l.filerFactory(b)
	if err != nil {
		return diag.FromErr(err)
	}

	dir, err := b.CacheDir(ctx, "terraform")
	if err != nil {
		return diag.FromErr(err)
	}

	const legacyStateFile = "terraform.tfstate"

	localPath := filepath.Join(dir, b.StateFileName())
	local, err := os.Open(localPath)
	if errors.Is(err, fs.ErrNotExist) {
		// Fallback to legacy local file name.
		localPath = filepath.Join(dir, legacyStateFile)
		local, err = os.Open(localPath)
	}
	if errors.Is(err, fs.ErrNotExist) {
		// Nothing to push – this is perfectly fine.
		log.Debugf(ctx, "Local state file does not exist: %s", localPath)
		return nil
	}
	if err != nil {
		return diag.FromErr(err)
	}
	defer local.Close()

	cmdio.LogString(ctx, "Updating deployment state...")
	log.Infof(ctx, "Writing local state file to remote state directory")
	if err = f.Write(ctx, legacyStateFile, local, filer.CreateParentDirectories, filer.OverwriteIfExists); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

// StatePush returns a mutator that pushes the local bundle state to the
// workspace-side storage.
func StatePush() bundle.Mutator {
	return &statePush{deploy.StateFiler}
}
