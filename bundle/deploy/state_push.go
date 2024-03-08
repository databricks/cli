package deploy

import (
	"context"
	"os"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/cli/libs/log"
)

type statePush struct {
	filerFactory
}

func (s *statePush) Name() string {
	return "deploy:state-push"
}

func (s *statePush) Apply(ctx context.Context, b *bundle.Bundle) error {
	f, err := s.filerFactory(b)
	if err != nil {
		return err
	}

	statePath, err := getPathToStateFile(ctx, b)
	if err != nil {
		return err
	}

	local, err := os.Open(statePath)
	if err != nil {
		return err
	}
	defer local.Close()

	log.Infof(ctx, "Writing local deployment state file to remote state directory")
	err = f.Write(ctx, DeploymentStateFileName, local, filer.CreateParentDirectories, filer.OverwriteIfExists)
	if err != nil {
		return err
	}

	return nil
}

func StatePush() bundle.Mutator {
	return &statePush{stateFiler}
}
