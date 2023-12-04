package metadata

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/paths"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/metadata"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestComputeMetadataMutator(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Workspace: config.Workspace{
				RootPath:     "/Users/shreyas.goenka@databricks.com",
				ArtifactPath: "/Users/shreyas.goenka@databricks.com/artifacts",
				FilePath:     "/Users/shreyas.goenka@databricks.com/files",
			},
			Bundle: config.Bundle{
				Name:   "my-bundle",
				Target: "development",
				Git: config.Git{
					Branch:         "my-branch",
					OriginURL:      "www.host.com",
					Commit:         "abcd",
					BundleRootPath: "a/b/c/d",
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

	expectedMetadata := metadata.Metadata{
		Version: metadata.Version,
		Config: metadata.Config{
			Workspace: metadata.Workspace{
				FilePath: "/Users/shreyas.goenka@databricks.com/files",
			},
			Bundle: metadata.Bundle{
				Git: config.Git{
					Branch:         "my-branch",
					OriginURL:      "www.host.com",
					Commit:         "abcd",
					BundleRootPath: "a/b/c/d",
				},
			},
			Resources: metadata.Resources{
				Jobs: map[string]*metadata.Job{
					"my-job-1": {
						RelativePath: "a/b/c",
						ID:           "1111",
					},
					"my-job-2": {
						RelativePath: "d/e/f",
						ID:           "2222",
					},
				},
			},
		},
	}

	err := bundle.Apply(context.Background(), b, Compute())
	require.NoError(t, err)

	assert.Equal(t, expectedMetadata, b.Metadata)
}
