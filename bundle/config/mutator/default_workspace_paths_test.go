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
				RootPath: "/",
			},
		},
	}
	_, err := mutator.DefineDefaultWorkspacePaths().Apply(context.Background(), bundle)
	require.NoError(t, err)
	assert.Equal(t, "/files", bundle.Config.Workspace.FilePath)
	assert.Equal(t, "/artifacts", bundle.Config.Workspace.ArtifactPath)
	assert.Equal(t, "/state", bundle.Config.Workspace.StatePath)
}

func TestDefineDefaultWorkspacePathsAlreadySet(t *testing.T) {
	bundle := &bundle.Bundle{
		Config: config.Root{
			Workspace: config.Workspace{
				RootPath:     "/",
				FilePath:     "/foo/bar",
				ArtifactPath: "/foo/bar",
				StatePath:    "/foo/bar",
			},
		},
	}
	_, err := mutator.DefineDefaultWorkspacePaths().Apply(context.Background(), bundle)
	require.NoError(t, err)
	assert.Equal(t, "/foo/bar", bundle.Config.Workspace.FilePath)
	assert.Equal(t, "/foo/bar", bundle.Config.Workspace.ArtifactPath)
	assert.Equal(t, "/foo/bar", bundle.Config.Workspace.StatePath)
}
