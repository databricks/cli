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
	return "terraform:state-push"
}

func (l *statePush) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	f, err := l.filerFactory(b)
	if err != nil {
		return diag.FromErr(err)
	}

	path, err := b.StateLocalPath(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	local, err := os.Open(path)
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

	// Upload state file from local cache directory to filer.
	cmdio.LogString(ctx, "Updating deployment state...")
	err = f.Write(ctx, b.StateFilename(), local, filer.CreateParentDirectories, filer.OverwriteIfExists)
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func StatePush() bundle.Mutator {
	return &statePush{deploy.StateFiler}
}
