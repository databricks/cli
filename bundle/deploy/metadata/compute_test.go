package metadata

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/paths"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/deploy"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestComputeMetadataMutator(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Workspace: config.Workspace{
				RootPath:      "/Users/shreyas.goenka@databricks.com",
				ArtifactsPath: "/Users/shreyas.goenka@databricks.com/artifacts",
			},
			Bundle: config.Bundle{
				Name:   "my-bundle",
				Target: "development",
				Git: config.Git{
					Branch:    "my-branch",
					OriginURL: "www.host.com",
					Commit:    "abcd",
				},
			},
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"my-job-1": {
						Paths: paths.Paths{
							LocalConfigFilePath: "a/b/c",
						},
						JobSettings: &jobs.JobSettings{
							Name: "My Job One",
						},
					},
					"my-job-2": {
						Paths: paths.Paths{
							LocalConfigFilePath: "d/e/f",
						},
						JobSettings: &jobs.JobSettings{
							Name: "My Job Two",
						},
					},
				},
				Pipelines: map[string]*resources.Pipeline{
					"my-pipeline": {
						Paths: paths.Paths{
							LocalConfigFilePath: "abc",
						},
					},
				},
			},
		},
	}

	expectedMetadata := deploy.Metadata{
		Version: deploy.LatestMetadataVersion,
		Config: config.Root{
			Workspace: config.Workspace{
				RootPath: "/Users/shreyas.goenka@databricks.com",
			},
			Bundle: config.Bundle{
				Git: config.Git{
					Branch:    "my-branch",
					OriginURL: "www.host.com",
					Commit:    "abcd",
				},
			},
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"my-job-1": {
						Paths: paths.Paths{
							LocalConfigFilePath: "a/b/c",
						},
					},
					"my-job-2": {
						Paths: paths.Paths{
							LocalConfigFilePath: "d/e/f",
						},
					},
				},
			},
		},
	}

	err := Compute().Apply(context.Background(), b)
	require.NoError(t, err)

	// Print expected and actual metadata for debugging
	actual, err := json.MarshalIndent(b.Metadata, "		", "	")
	assert.NoError(t, err)
	t.Log("[DEBUG] actual: ", string(actual))
	expected, err := json.MarshalIndent(expectedMetadata, "		", "	")
	assert.NoError(t, err)
	t.Log("[DEBUG] expected: ", string(expected))

	assert.Equal(t, expectedMetadata, b.Metadata)
}
