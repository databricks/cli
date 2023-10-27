package metadata

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/assert"
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
						JobSettings: &jobs.JobSettings{
							Name: "My Job One",
						},
					},
					"my-job-2": {
						JobSettings: &jobs.JobSettings{
							Name: "My Job Two",
						},
					},
				},
			},
		},
	}

	err := AnnotateJobs().Apply(context.Background(), b)
	assert.NoError(t, err)

	assert.Equal(t,
		&jobs.JobDeployment{
			Kind:             jobs.JobDeploymentKindBundle,
			MetadataFilePath: "/a/b/c/metadata.json",
		},
		b.Config.Resources.Jobs["my-job-1"].JobSettings.Deployment)
	assert.Equal(t,
		&jobs.JobDeployment{
			Kind:             jobs.JobDeploymentKindBundle,
			MetadataFilePath: "/a/b/c/metadata.json",
		},
		b.Config.Resources.Jobs["my-job-2"].JobSettings.Deployment)
}
