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
				FilesPath:     "/Users/shreyas.goenka@databricks.com/files",
			},
			Bundle: config.Bundle{
				Name:   "my-bundle",
				Target: "development",
				Git: config.Git{
					Branch:     "my-branch",
					OriginURL:  "www.host.com",
					Commit:     "abcd",
					BundleRoot: "a/b/c/d",
				},
			},
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"my-job-1": {
						Paths: paths.Paths{
							ConfigFilePath: "a/b/c",
						},
						ID: "1111",
						JobSettings: &jobs.JobSettings{
							Name: "My Job One",
						},
					},
					"my-job-2": {
						Paths: paths.Paths{
							ConfigFilePath: "d/e/f",
						},
						ID: "2222",
						JobSettings: &jobs.JobSettings{
							Name: "My Job Two",
						},
					},
				},
				Pipelines: map[string]*resources.Pipeline{
					"my-pipeline": {
						Paths: paths.Paths{
							ConfigFilePath: "abc",
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
				FilesPath: "/Users/shreyas.goenka@databricks.com/files",
			},
			Bundle: config.Bundle{
				Git: config.Git{
					Branch:     "my-branch",
					OriginURL:  "www.host.com",
					Commit:     "abcd",
					BundleRoot: "a/b/c/d",
				},
			},
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"my-job-1": {
						Paths: paths.Paths{
							RelativePath: "a/b/c",
						},
						ID: "1111",
					},
					"my-job-2": {
						Paths: paths.Paths{
							RelativePath: "d/e/f",
						},
						ID: "2222",
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
