package deploy

import (
	"context"
	"os"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/cli/libs/log"
)

const MaxStateFileSize = 10 * 1024 * 1024 // 10MB

type statePush struct {
	filerFactory FilerFactory
}

func (s *statePush) Name() string {
	return "deploy:state-push"
}

func (s *statePush) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	f, err := s.filerFactory(b)
	if err != nil {
		return diag.FromErr(err)
	}

	statePath, err := getPathToStateFile(ctx, b)
	if err != nil {
		return diag.FromErr(err)
	}

	local, err := os.Open(statePath)
	if err != nil {
		return diag.FromErr(err)
	}
	defer local.Close()

	if !b.Config.Bundle.Force {
		state, err := local.Stat()
		if err != nil {
			return diag.FromErr(err)
		}

		if state.Size() > MaxStateFileSize {
			return diag.Errorf("Deployment state file size exceeds the maximum allowed size of %d bytes. Please reduce the number of resources in your bundle, split your bundle into multiple or re-run the command with --force flag.", MaxStateFileSize)
		}
	}

	log.Infof(ctx, "Writing local deployment state file to remote state directory")
	err = f.Write(ctx, DeploymentStateFileName, local, filer.CreateParentDirectories, filer.OverwriteIfExists)
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}

// StatePush returns a mutator that pushes the deployment state file to Databricks workspace.
func StatePush() bundle.Mutator {
	return &statePush{StateFiler}
}
