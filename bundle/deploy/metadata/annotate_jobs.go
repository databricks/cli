package metadata

import (
	"context"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/env"
	"github.com/databricks/cli/libs/dbr"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/sdktypes/jobs/v2"
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
			Kind:             jobs.JobDeployment_DeploymentKind_Bundle,
			MetadataFilePath: new(metadataFilePath(b)),
		}

		isDatabricksWorkspace := dbr.RunsOnRuntime(ctx) && strings.HasPrefix(b.SyncRootPath, "/Workspace/")
		_, experimentalYamlSyncEnabled := env.ExperimentalYamlSync(ctx)
		if isDatabricksWorkspace && experimentalYamlSyncEnabled && b.Config.Bundle.Mode == config.Development {
			job.EditMode = jobs.JobEditMode_Editable
		} else {
			job.EditMode = jobs.JobEditMode_UiLocked
		}

		job.Format = jobs.Format_MultiTask
	}

	return nil
}
