package terraform

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
	if errors.Is(err, fs.ErrNotExist) {
		// The state file can be absent if terraform apply is skipped because
		// there are no changes to apply in the plan.
		log.Debugf(ctx, "Local terraform state file does not exist.")
		return nil
	}
	if err != nil {
		return diag.FromErr(err)
	}
	defer local.Close()

	if !b.Config.Bundle.Force {
		state, err := local.Stat()
		if err != nil {
			return diag.FromErr(err)
		}

		if state.Size() > deploy.MaxStateFileSize {
			return diag.Errorf("Terraform state file size exceeds the maximum allowed size of %d bytes. Please reduce the number of resources in your bundle, split your bundle into multiple or re-run the command with --force flag", deploy.MaxStateFileSize)
		}
	}

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
