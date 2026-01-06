package metadata

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/dbr"
	"github.com/databricks/cli/libs/env"
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

	ctx := context.Background()
	ctx = dbr.MockRuntime(ctx, dbr.Environment{IsDbr: false, Version: ""})

	diags := AnnotateJobs().Apply(ctx, b)
	require.NoError(t, diags.Error())

	assert.Equal(t,
		&jobs.JobDeployment{
			Kind:             jobs.JobDeploymentKindBundle,
			MetadataFilePath: "/a/b/c/metadata.json",
		},
		b.Config.Resources.Jobs["my-job-1"].Deployment)
	assert.Equal(t, jobs.JobEditModeUiLocked, b.Config.Resources.Jobs["my-job-1"].EditMode)
	assert.Equal(t, jobs.FormatMultiTask, b.Config.Resources.Jobs["my-job-1"].Format)

	assert.Equal(t,
		&jobs.JobDeployment{
			Kind:             jobs.JobDeploymentKindBundle,
			MetadataFilePath: "/a/b/c/metadata.json",
		},
		b.Config.Resources.Jobs["my-job-2"].Deployment)
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

	ctx := context.Background()
	ctx = dbr.MockRuntime(ctx, dbr.Environment{IsDbr: false, Version: ""})

	diags := AnnotateJobs().Apply(ctx, b)
	require.NoError(t, diags.Error())
}

func TestAnnotateJobsWorkspaceWithFlag(t *testing.T) {
	b := &bundle.Bundle{
		SyncRootPath: "/Workspace/Users/user@example.com/project",
		Config: config.Root{
			Workspace: config.Workspace{
				StatePath: "/a/b/c",
			},
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"my-job": {
						JobSettings: jobs.JobSettings{
							Name: "My Job",
						},
					},
				},
			},
		},
	}

	ctx := context.Background()
	ctx = dbr.MockRuntime(ctx, dbr.Environment{IsDbr: true, Version: "14.0"})
	ctx = env.Set(ctx, "DATABRICKS_BUNDLE_ENABLE_EXPERIMENTAL_YAML_SYNC", "1")

	diags := AnnotateJobs().Apply(ctx, b)
	require.NoError(t, diags.Error())

	assert.Equal(t, jobs.JobEditModeEditable, b.Config.Resources.Jobs["my-job"].EditMode)
	assert.Equal(t, jobs.FormatMultiTask, b.Config.Resources.Jobs["my-job"].Format)
}

func TestAnnotateJobsWorkspaceWithoutFlag(t *testing.T) {
	b := &bundle.Bundle{
		SyncRootPath: "/Workspace/Users/user@example.com/project",
		Config: config.Root{
			Workspace: config.Workspace{
				StatePath: "/a/b/c",
			},
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"my-job": {
						JobSettings: jobs.JobSettings{
							Name: "My Job",
						},
					},
				},
			},
		},
	}

	ctx := context.Background()
	ctx = dbr.MockRuntime(ctx, dbr.Environment{IsDbr: true, Version: "14.0"})

	diags := AnnotateJobs().Apply(ctx, b)
	require.NoError(t, diags.Error())

	assert.Equal(t, jobs.JobEditModeUiLocked, b.Config.Resources.Jobs["my-job"].EditMode)
	assert.Equal(t, jobs.FormatMultiTask, b.Config.Resources.Jobs["my-job"].Format)
}

func TestAnnotateJobsNonWorkspaceWithFlag(t *testing.T) {
	b := &bundle.Bundle{
		SyncRootPath: "/some/local/path",
		Config: config.Root{
			Workspace: config.Workspace{
				StatePath: "/a/b/c",
			},
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"my-job": {
						JobSettings: jobs.JobSettings{
							Name: "My Job",
						},
					},
				},
			},
		},
	}

	ctx := context.Background()
	ctx = dbr.MockRuntime(ctx, dbr.Environment{IsDbr: false, Version: ""})
	ctx = env.Set(ctx, "DATABRICKS_BUNDLE_ENABLE_EXPERIMENTAL_YAML_SYNC", "1")

	diags := AnnotateJobs().Apply(ctx, b)
	require.NoError(t, diags.Error())

	assert.Equal(t, jobs.JobEditModeUiLocked, b.Config.Resources.Jobs["my-job"].EditMode)
	assert.Equal(t, jobs.FormatMultiTask, b.Config.Resources.Jobs["my-job"].Format)
}
