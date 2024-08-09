package artifacts

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/libraries"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/cli/libs/log"
)

func CleanUp() bundle.Mutator {
	return &cleanUp{}
}

type cleanUp struct{}

func (m *cleanUp) Name() string {
	return "artifacts.CleanUp"
}

func (m *cleanUp) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	uploadPath, err := libraries.GetUploadBasePath(b)
	if err != nil {
		return diag.FromErr(err)
	}

	client, err := libraries.GetFilerForLibraries(b.WorkspaceClient(), uploadPath)
	if err != nil {
		return diag.FromErr(err)
	}

	// We intentionally ignore the error because it is not critical to the deployment
	err = client.Delete(ctx, ".", filer.DeleteRecursively)
	if err != nil {
		log.Errorf(ctx, "failed to delete %s: %v", uploadPath, err)
	}

	err = client.Mkdir(ctx, ".")
	if err != nil {
		return diag.Errorf("unable to create directory for %s: %v", uploadPath, err)
	}

	return nil
}
