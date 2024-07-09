package destroy

import (
	"context"
	"errors"
	"net/http"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go/apierr"
)

type assertRootPathExists struct{}

func AssertRootPathExists() bundle.Mutator {
	return &assertRootPathExists{}
}

func (m *assertRootPathExists) Name() string {
	return "destroy:assert_root_path_exists"
}

func (m *assertRootPathExists) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	w := b.WorkspaceClient()
	_, err := w.Workspace.GetStatusByPath(ctx, b.Config.Workspace.RootPath)

	if err != nil {
		var aerr *apierr.APIError
		if errors.As(err, &aerr) && aerr.StatusCode == http.StatusNotFound {
			log.Infof(ctx, "No active deployment found. %s does not exist. Skipping destroy.", b.Config.Workspace.RootPath)
			cmdio.LogString(ctx, "No active deployment found to destroy!")
			return bundle.DiagnosticSequenceBreak
		}
	}
	return nil
}
