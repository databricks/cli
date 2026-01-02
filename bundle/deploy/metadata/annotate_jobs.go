package metadata

import (
	"context"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/env"
	"github.com/databricks/cli/libs/dbr"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/databricks-sdk-go/service/jobs"
)

type annotateJobs struct{}

func AnnotateJobs() bundle.Mutator {
	return &annotateJobs{}
}

func (m *annotateJobs) Name() string {
	return "metadata.AnnotateJobs"
}

func (m *annotateJobs) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	for _, job := range b.Config.Resources.Jobs {
		job.Deployment = &jobs.JobDeployment{
			Kind:             jobs.JobDeploymentKindBundle,
			MetadataFilePath: metadataFilePath(b),
		}

		isDatabricksWorkspace := dbr.RunsOnRuntime(ctx) && strings.HasPrefix(b.SyncRootPath, "/Workspace/")
		_, experimentalYamlSyncEnabled := env.ExperimentalYamlSync(ctx)
		if isDatabricksWorkspace && experimentalYamlSyncEnabled {
			job.EditMode = jobs.JobEditModeEditable
		} else {
			job.EditMode = jobs.JobEditModeUiLocked
		}

		job.Format = jobs.FormatMultiTask
	}

	return nil
}
