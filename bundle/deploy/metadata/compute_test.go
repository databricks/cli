package metadata

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/internal/bundletest"
	"github.com/databricks/cli/bundle/metadata"
	"github.com/databricks/cli/libs/dyn"
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
						ID: "1111",
						JobSettings: jobs.JobSettings{
							Name: "My Job One",
						},
					},
					"my-job-2": {
						ID: "2222",
						JobSettings: jobs.JobSettings{
							Name: "My Job Two",
						},
					},
				},
				Pipelines: map[string]*resources.Pipeline{
					"my-pipeline": {},
				},
			},
		},
	}

	bundletest.SetLocation(b, "resources.jobs.my-job-1", []dyn.Location{{File: "a/b/c"}})
	bundletest.SetLocation(b, "resources.jobs.my-job-2", []dyn.Location{{File: "d/e/f"}})
	bundletest.SetLocation(b, "resources.pipelines.my-pipeline", []dyn.Location{{File: "abc"}})

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

	diags := bundle.Apply(context.Background(), b, Compute())
	require.NoError(t, diags.Error())

	assert.Equal(t, expectedMetadata, b.Metadata)
}

func TestComputeMetadataMutatorSourceLinked(t *testing.T) {
	syncRootPath := "/Users/shreyas.goenka@databricks.com/source"
	enabled := true
	b := &bundle.Bundle{
		SyncRootPath: syncRootPath,
		Config: config.Root{
			Presets: config.Presets{
				SourceLinkedDeployment: &enabled,
			},
			Workspace: config.Workspace{
				FilePath: "/Users/shreyas.goenka@databricks.com/files",
			},
		},
	}

	diags := bundle.Apply(context.Background(), b, Compute())
	require.NoError(t, diags.Error())

	assert.Equal(t, syncRootPath, b.Metadata.Config.Workspace.FilePath)
}
