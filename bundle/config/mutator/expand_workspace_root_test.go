package mutator_test

import (
	"context"
	"testing"

	"github.com/databricks/bricks/bundle"
	"github.com/databricks/bricks/bundle/config"
	"github.com/databricks/bricks/bundle/config/mutator"
	"github.com/databricks/databricks-sdk-go/service/scim"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExpandWorkspaceRoot(t *testing.T) {
	bundle := &bundle.Bundle{
		Config: config.Root{
			Workspace: config.Workspace{
				CurrentUser: &scim.User{
					UserName: "jane@doe.com",
				},
				Root: "~/foo",
			},
		},
	}
	_, err := mutator.ExpandWorkspaceRoot().Apply(context.Background(), bundle)
	require.NoError(t, err)
	assert.Equal(t, "/Users/jane@doe.com/foo", bundle.Config.Workspace.Root)
}

func TestExpandWorkspaceRootDoesNothing(t *testing.T) {
	bundle := &bundle.Bundle{
		Config: config.Root{
			Workspace: config.Workspace{
				CurrentUser: &scim.User{
					UserName: "jane@doe.com",
				},
				Root: "/Users/charly@doe.com/foo",
			},
		},
	}
	_, err := mutator.ExpandWorkspaceRoot().Apply(context.Background(), bundle)
	require.NoError(t, err)
	assert.Equal(t, "/Users/charly@doe.com/foo", bundle.Config.Workspace.Root)
}

func TestExpandWorkspaceRootWithoutRoot(t *testing.T) {
	bundle := &bundle.Bundle{
		Config: config.Root{
			Workspace: config.Workspace{
				CurrentUser: &scim.User{
					UserName: "jane@doe.com",
				},
			},
		},
	}
	_, err := mutator.ExpandWorkspaceRoot().Apply(context.Background(), bundle)
	require.Error(t, err)
}

func TestExpandWorkspaceRootWithoutCurrentUser(t *testing.T) {
	bundle := &bundle.Bundle{
		Config: config.Root{
			Workspace: config.Workspace{
				Root: "~/foo",
			},
		},
	}
	_, err := mutator.ExpandWorkspaceRoot().Apply(context.Background(), bundle)
	require.Error(t, err)
}
