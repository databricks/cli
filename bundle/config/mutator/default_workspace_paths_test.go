package mutator_test

import (
	"context"
	"testing"

	"github.com/databricks/bricks/bundle"
	"github.com/databricks/bricks/bundle/config"
	"github.com/databricks/bricks/bundle/config/mutator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefineDefaultWorkspacePaths(t *testing.T) {
	bundle := &bundle.Bundle{
		Config: config.Root{
			Workspace: config.Workspace{
				Root: "/",
			},
		},
	}
	_, err := mutator.DefineDefaultWorkspacePaths().Apply(context.Background(), bundle)
	require.NoError(t, err)
	assert.Equal(t, "/files", bundle.Config.Workspace.FilePath.Workspace)
	assert.Equal(t, "/artifacts", bundle.Config.Workspace.ArtifactPath.Workspace)
	assert.Equal(t, "/state", bundle.Config.Workspace.StatePath.Workspace)
}

func TestDefineDefaultWorkspacePathsAlreadySet(t *testing.T) {
	bundle := &bundle.Bundle{
		Config: config.Root{
			Workspace: config.Workspace{
				Root: "/",
				FilePath: config.PathLike{
					Workspace: "/foo/bar",
				},
				ArtifactPath: config.PathLike{
					Workspace: "/foo/bar",
				},
				StatePath: config.PathLike{
					Workspace: "/foo/bar",
				},
			},
		},
	}
	_, err := mutator.DefineDefaultWorkspacePaths().Apply(context.Background(), bundle)
	require.NoError(t, err)
	assert.Equal(t, "/foo/bar", bundle.Config.Workspace.FilePath.Workspace)
	assert.Equal(t, "/foo/bar", bundle.Config.Workspace.ArtifactPath.Workspace)
	assert.Equal(t, "/foo/bar", bundle.Config.Workspace.StatePath.Workspace)
}
