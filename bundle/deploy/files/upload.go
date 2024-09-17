package files

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/sync"
)

type upload struct {
	outpuHandler sync.OutputHandler
}

func (m *upload) Name() string {
	return "files.Upload"
}

func (m *upload) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	cmdio.LogString(ctx, fmt.Sprintf("Uploading bundle files to %s...", b.Config.Workspace.FilePath))
	opts, err := GetSyncOptions(ctx, bundle.ReadOnly(b))
	if err != nil {
		return diag.FromErr(err)
	}

	opts.OutputHandler = m.outpuHandler
	sync, err := sync.New(ctx, *opts)
	if err != nil {
		return diag.FromErr(err)
	}

	b.Files, err = sync.RunOnce(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	log.Infof(ctx, "Uploaded bundle files")
	return nil
}

func Upload(outputHandler sync.OutputHandler) bundle.Mutator {
	return &upload{outputHandler}
}
