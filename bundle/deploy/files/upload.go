package files

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/log"
)

type upload struct{}

func (m *upload) Name() string {
	return "files.Upload"
}

func (m *upload) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	cmdio.LogString(ctx, fmt.Sprintf("Uploading bundle files to %s...", b.Config.Workspace.FilePath))
	sync, err := GetSync(ctx, b)
	if err != nil {
		return err
	}

	err = sync.RunOnce(ctx)
	if err != nil {
		return err
	}

	log.Infof(ctx, "Uploaded bundle files")
	return nil
}

func Upload() bundle.Mutator {
	return &upload{}
}
