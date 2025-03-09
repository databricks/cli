package files

import (
	"context"
	"errors"
	"fmt"
	"io/fs"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/permissions"
	"github.com/databricks/cli/clis"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/sync"
)

type upload struct {
	outputHandler sync.OutputHandler
	cliType       clis.CLIType
}

func (m *upload) Name() string {
	return "files.Upload"
}

func (m *upload) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	if config.IsExplicitlyEnabled(b.Config.Presets.SourceLinkedDeployment) {
		cmdio.LogString(ctx, "Source-linked deployment is enabled. Deployed resources reference the source files in your working tree instead of separate copies.")
		return nil
	}

	if m.cliType != clis.DLT {
		cmdio.LogString(ctx, fmt.Sprintf("Uploading files to %s...", b.Config.Workspace.FilePath))
	}
	opts, err := GetSyncOptions(ctx, b)
	if err != nil {
		return diag.FromErr(err)
	}

	opts.OutputHandler = m.outputHandler
	sync, err := sync.New(ctx, *opts)
	if err != nil {
		return diag.FromErr(err)
	}
	defer sync.Close()

	b.Files, err = sync.RunOnce(ctx)
	if err != nil {
		if errors.Is(err, fs.ErrPermission) {
			return permissions.ReportPossiblePermissionDenied(ctx, b, b.Config.Workspace.FilePath)
		}
		return diag.FromErr(err)
	}

	log.Infof(ctx, "Uploaded bundle files")
	return nil
}

func Upload(outputHandler sync.OutputHandler, cliType clis.CLIType) bundle.Mutator {
	return &upload{outputHandler, cliType}
}
