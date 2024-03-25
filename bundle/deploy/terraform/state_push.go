package terraform

import (
	"context"
	"os"
	"path/filepath"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/deploy"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/cli/libs/log"
)

type statePush struct {
	filerFactory deploy.FilerFactory
}

func (l *statePush) Name() string {
	return "terraform:state-push"
}

func (l *statePush) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	f, err := l.filerFactory(b)
	if err != nil {
		return diag.FromErr(err)
	}

	dir, err := Dir(ctx, b)
	if err != nil {
		return diag.FromErr(err)
	}

	// Expect the state file to live under dir.
	local, err := os.Open(filepath.Join(dir, TerraformStateFileName))
	if err != nil {
		return diag.FromErr(err)
	}
	defer local.Close()

	// Upload state file from local cache directory to filer.
	cmdio.LogString(ctx, "Updating deployment state...")
	log.Infof(ctx, "Writing local state file to remote state directory")
	err = f.Write(ctx, TerraformStateFileName, local, filer.CreateParentDirectories, filer.OverwriteIfExists)
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func StatePush() bundle.Mutator {
	return &statePush{deploy.StateFiler}
}
