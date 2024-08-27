package mutator_test

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExpandWorkspaceRoot(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Workspace: config.Workspace{
				CurrentUser: &config.User{
					User: &iam.User{
						UserName: "jane@doe.com",
					},
				},
				RootPath: "~/foo",
			},
		},
	}
	diags := bundle.Apply(context.Background(), b, mutator.ExpandWorkspaceRoot())
	require.NoError(t, diags.Error())
	assert.Equal(t, "/Workspace/Users/jane@doe.com/foo", b.Config.Workspace.RootPath)
}

func TestExpandWorkspaceRootDoesNothing(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Workspace: config.Workspace{
				CurrentUser: &config.User{
					User: &iam.User{
						UserName: "jane@doe.com",
					},
				},
				RootPath: "/Users/charly@doe.com/foo",
			},
		},
	}
	diags := bundle.Apply(context.Background(), b, mutator.ExpandWorkspaceRoot())
	require.NoError(t, diags.Error())
	assert.Equal(t, "/Users/charly@doe.com/foo", b.Config.Workspace.RootPath)
}

func TestExpandWorkspaceRootWithoutRoot(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Workspace: config.Workspace{
				CurrentUser: &config.User{
					User: &iam.User{
						UserName: "jane@doe.com",
					},
				},
			},
		},
	}
	diags := bundle.Apply(context.Background(), b, mutator.ExpandWorkspaceRoot())
	require.True(t, diags.HasError())
}

func TestExpandWorkspaceRootWithoutCurrentUser(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Workspace: config.Workspace{
				RootPath: "~/foo",
			},
		},
	}
	diags := bundle.Apply(context.Background(), b, mutator.ExpandWorkspaceRoot())
	require.True(t, diags.HasError())
}
