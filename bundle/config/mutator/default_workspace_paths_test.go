package mutator_test

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/mutator"
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
	assert.Equal(t, "/files", bundle.Config.Workspace.FilesPath)
	assert.Equal(t, "/artifacts", bundle.Config.Workspace.ArtifactsPath)
	assert.Equal(t, "/state", bundle.Config.Workspace.StatePath)
}

func TestDefineDefaultWorkspacePathsAlreadySet(t *testing.T) {
	bundle := &bundle.Bundle{
		Config: config.Root{
			Workspace: config.Workspace{
				RootPath:      "/",
				FilesPath:     "/foo/bar",
				ArtifactsPath: "/foo/bar",
				StatePath:     "/foo/bar",
			},
		},
	}
	_, err := mutator.DefineDefaultWorkspacePaths().Apply(context.Background(), bundle)
	require.NoError(t, err)
	assert.Equal(t, "/foo/bar", bundle.Config.Workspace.FilesPath)
	assert.Equal(t, "/foo/bar", bundle.Config.Workspace.ArtifactsPath)
	assert.Equal(t, "/foo/bar", bundle.Config.Workspace.StatePath)
}
