package statemgmt

import (
	"context"
	"errors"
	"io/fs"
	"os"

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
	return "statemgmt.Push"
}

func (l *statePush) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	if b.DirectDeployment == nil {
		return diag.Errorf("internal error: statemgmt.Load() called without statemgmt.PullResourcesState()")
	}

	f, err := l.filerFactory(b)
	if err != nil {
		return diag.FromErr(err)
	}

	var remotePath, localPath string

	if *b.DirectDeployment {
		remotePath, localPath = b.StateFilenameDirect(ctx)
	} else {
		remotePath, localPath = b.StateFilenameTerraform(ctx)
	}

	local, err := os.Open(localPath)
	if errors.Is(err, fs.ErrNotExist) {
		// The state file can be absent if terraform apply is skipped because
		// there are no changes to apply in the plan.
		log.Debugf(ctx, "Local state file does not exist: %s", localPath)
		return nil
	}
	if err != nil {
		return diag.FromErr(err)
	}
	defer local.Close()

	// Upload state file from local cache directory to filer.
	cmdio.LogString(ctx, "Updating deployment state...")
	err = f.Write(ctx, remotePath, local, filer.CreateParentDirectories, filer.OverwriteIfExists)
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func StatePush() bundle.Mutator {
	return &statePush{deploy.StateFiler}
}
