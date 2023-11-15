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
	b := &bundle.Bundle{
		Config: config.Root{
			Workspace: config.Workspace{
				RootPath: "/",
			},
		},
	}
	err := mutator.DefineDefaultWorkspacePaths().Apply(context.Background(), b)
	require.NoError(t, err)
	assert.Equal(t, "/files", b.Config.Workspace.FilesPath)
	assert.Equal(t, "/artifacts", b.Config.Workspace.ArtifactsPath)
	assert.Equal(t, "/state", b.Config.Workspace.StatePath)
}

func TestDefineDefaultWorkspacePathsAlreadySet(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Workspace: config.Workspace{
				RootPath:      "/",
				FilesPath:     "/foo/bar",
				ArtifactsPath: "/foo/bar",
				StatePath:     "/foo/bar",
			},
		},
	}
	err := mutator.DefineDefaultWorkspacePaths().Apply(context.Background(), b)
	require.NoError(t, err)
	assert.Equal(t, "/foo/bar", b.Config.Workspace.FilesPath)
	assert.Equal(t, "/foo/bar", b.Config.Workspace.ArtifactsPath)
	assert.Equal(t, "/foo/bar", b.Config.Workspace.StatePath)
}
