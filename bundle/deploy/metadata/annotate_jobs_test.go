package metadata

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAnnotateJobsMutator(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Workspace: config.Workspace{
				StatePath: "/a/b/c",
			},
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"my-job-1": {
						JobSettings: jobs.JobSettings{
							Name: "My Job One",
						},
					},
					"my-job-2": {
						JobSettings: jobs.JobSettings{
							Name: "My Job Two",
						},
					},
				},
			},
		},
	}

	diags := AnnotateJobs().Apply(context.Background(), b)
	require.NoError(t, diags.Error())

	assert.Equal(t,
		&jobs.JobDeployment{
			Kind:             jobs.JobDeploymentKindBundle,
			MetadataFilePath: "/a/b/c/metadata.json",
		},
		b.Config.Resources.Jobs["my-job-1"].JobSettings.Deployment)
	assert.Equal(t, jobs.JobEditModeUiLocked, b.Config.Resources.Jobs["my-job-1"].EditMode)
	assert.Equal(t, jobs.FormatMultiTask, b.Config.Resources.Jobs["my-job-1"].Format)

	assert.Equal(t,
		&jobs.JobDeployment{
			Kind:             jobs.JobDeploymentKindBundle,
			MetadataFilePath: "/a/b/c/metadata.json",
		},
		b.Config.Resources.Jobs["my-job-2"].JobSettings.Deployment)
	assert.Equal(t, jobs.JobEditModeUiLocked, b.Config.Resources.Jobs["my-job-2"].EditMode)
	assert.Equal(t, jobs.FormatMultiTask, b.Config.Resources.Jobs["my-job-2"].Format)
}

func TestAnnotateJobsMutatorJobWithoutSettings(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"my-job-1": {},
				},
			},
		},
	}

	diags := AnnotateJobs().Apply(context.Background(), b)
	require.NoError(t, diags.Error())
}
