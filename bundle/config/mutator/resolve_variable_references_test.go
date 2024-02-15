package mutator

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/config/variable"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/require"
)

func TestResolveVariableReferences(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Bundle: config.Bundle{
				Name: "example",
			},
			Workspace: config.Workspace{
				RootPath: "${bundle.name}/bar",
				FilePath: "${workspace.root_path}/baz",
			},
		},
	}

	// Apply with an invalid prefix. This should not change the workspace root path.
	err := bundle.Apply(context.Background(), b, ResolveVariableReferences("doesntexist"))
	require.NoError(t, err)
	require.Equal(t, "${bundle.name}/bar", b.Config.Workspace.RootPath)
	require.Equal(t, "${workspace.root_path}/baz", b.Config.Workspace.FilePath)

	// Apply with a valid prefix. This should change the workspace root path.
	err = bundle.Apply(context.Background(), b, ResolveVariableReferences("bundle", "workspace"))
	require.NoError(t, err)
	require.Equal(t, "example/bar", b.Config.Workspace.RootPath)
	require.Equal(t, "example/bar/baz", b.Config.Workspace.FilePath)
}

func TestResolveVariableReferencesToBundleVariables(t *testing.T) {
	s := func(s string) *string {
		return &s
	}

	b := &bundle.Bundle{
		Config: config.Root{
			Bundle: config.Bundle{
				Name: "example",
			},
			Workspace: config.Workspace{
				RootPath: "${bundle.name}/${var.foo}",
			},
			Variables: map[string]*variable.Variable{
				"foo": {
					Value: s("bar"),
				},
			},
		},
	}

	// Apply with a valid prefix. This should change the workspace root path.
	err := bundle.Apply(context.Background(), b, ResolveVariableReferences("bundle", "variables"))
	require.NoError(t, err)
	require.Equal(t, "example/bar", b.Config.Workspace.RootPath)
}

func TestResolveVariableReferencesToEmptyFields(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Bundle: config.Bundle{
				Name: "example",
				Git: config.Git{
					Branch: "",
				},
			},
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"job1": {
						JobSettings: &jobs.JobSettings{
							Tags: map[string]string{
								"git_branch": "${bundle.git.branch}",
							},
						},
					},
				},
			},
		},
	}

	// Apply for the bundle prefix.
	err := bundle.Apply(context.Background(), b, ResolveVariableReferences("bundle"))
	require.NoError(t, err)

	// The job settings should have been interpolated to an empty string.
	require.Equal(t, "", b.Config.Resources.Jobs["job1"].JobSettings.Tags["git_branch"])
}
