package deploy

import (
	"context"
	"os"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/cli/libs/log"
)

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
