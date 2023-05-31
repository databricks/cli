package terraform

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/cli/libs/log"
)

type statePull struct{}

func (l *statePull) Name() string {
	return "terraform:state-pull"
}

func (l *statePull) Apply(ctx context.Context, b *bundle.Bundle) error {
	f, err := filer.NewWorkspaceFilesClient(b.WorkspaceClient(), b.Config.Workspace.StatePath)
	if err != nil {
		return err
	}

	dir, err := Dir(b)
	if err != nil {
		return err
	}

	// Download state file from filer to local cache directory.
	log.Infof(ctx, "Opening remote state file")
	remote, err := f.Read(ctx, TerraformStateFileName)
	if err != nil {
		// On first deploy this state file doesn't yet exist.
		if errors.Is(err, fs.ErrNotExist) {
			log.Infof(ctx, "Remote state file does not exist")
			return nil
		}
		return err
	}

	// Expect the state file to live under dir.
	local, err := os.OpenFile(filepath.Join(dir, TerraformStateFileName), os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return err
	}
	defer local.Close()

	if !IsLocalStateStale(local, remote) {
		log.Infof(ctx, "Local state is the same or newer, ignoring remote state")
		return nil
	}

	// Truncating the file before writing
	local.Truncate(0)
	local.Seek(0, 0)

	// Write file to disk.
	log.Infof(ctx, "Writing remote state file to local cache directory")
	_, err = io.Copy(local, remote)
	if err != nil {
		return err
	}

	return nil
}

func StatePull() bundle.Mutator {
	return &statePull{}
}
