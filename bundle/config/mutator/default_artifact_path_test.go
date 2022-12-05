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

func TestDefaultArtifactPath(t *testing.T) {
	bundle := &bundle.Bundle{
		Config: config.Root{
			Bundle: config.Bundle{
				Name:        "name",
				Environment: "environment",
			},
			Workspace: config.Workspace{
				CurrentUser: &scim.User{
					UserName: "foo@bar.com",
				},
			},
		},
	}
	_, err := mutator.DefaultArtifactPath().Apply(context.Background(), bundle)
	require.NoError(t, err)
	assert.Equal(t, "/Users/foo@bar.com/.bundle/name/environment/artifacts", bundle.Config.Workspace.ArtifactPath.Workspace)
}

func TestDefaultArtifactPathAlreadySet(t *testing.T) {
	bundle := &bundle.Bundle{
		Config: config.Root{
			Workspace: config.Workspace{
				ArtifactPath: config.PathLike{
					Workspace: "/foo/bar",
				},
				CurrentUser: &scim.User{
					UserName: "foo@bar.com",
				},
			},
		},
	}
	_, err := mutator.DefaultArtifactPath().Apply(context.Background(), bundle)
	require.NoError(t, err)
	assert.Equal(t, "/foo/bar", bundle.Config.Workspace.ArtifactPath.Workspace)
}
