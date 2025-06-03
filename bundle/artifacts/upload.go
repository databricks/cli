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
	client, uploadPath, diags := libraries.GetFilerForLibrariesCleanup(ctx, b)
	if diags.HasError() {
		return diags
	}

	skipArtifactsCleanup := b.Config.Experimental != nil && b.Config.Experimental.SkipArtifactCleanup
	b.Metrics.AddBoolValue("skip_artifact_cleanup", skipArtifactsCleanup)
	if skipArtifactsCleanup {
		log.Info(ctx, "Skip cleaning up artifacts folder")
	} else {
		// We intentionally ignore the error because it is not critical to the deployment
		err := client.Delete(ctx, libraries.InternalDirName, filer.DeleteRecursively)
		if err != nil {
			log.Debugf(ctx, "failed to delete %s: %v", uploadPath, err)
		}
	}

	err := client.Mkdir(ctx, libraries.InternalDirName)
	if err != nil {
		return diag.Errorf("unable to create directory for %s: %v", uploadPath, err)
	}

	return nil
}
