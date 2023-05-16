package terraform

import (
	"context"
	"os"
	"path/filepath"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/cli/libs/log"
)

type statePush struct{}

func (l *statePush) Name() string {
	return "terraform:state-push"
}

func (l *statePush) Apply(ctx context.Context, b *bundle.Bundle) ([]bundle.Mutator, error) {
	f, err := filer.NewWorkspaceFilesClient(b.WorkspaceClient(), b.Config.Workspace.StatePath)
	if err != nil {
		return nil, err
	}

	dir, err := Dir(b)
	if err != nil {
		return nil, err
	}

	// Expect the state file to live under dir.
	local, err := os.Open(filepath.Join(dir, TerraformStateFileName))
	if err != nil {
		return nil, err
	}

	// Upload state file from local cache directory to filer.
	log.Infof(ctx, "Writing local state file to remote state directory")
	err = f.Write(ctx, TerraformStateFileName, local, filer.CreateParentDirectories, filer.OverwriteIfExists)
	if err != nil {
		return nil, err
	}

	return nil, nil
}

func StatePush() bundle.Mutator {
	return &statePush{}
}
