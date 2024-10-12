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
	diags := bundle.Apply(context.Background(), b, mutator.DefineDefaultWorkspacePaths())
	require.NoError(t, diags.Error())
	assert.Equal(t, "/files", b.Config.Workspace.FilePath)
	assert.Equal(t, "/resources", b.Config.Workspace.ResourcePath)
	assert.Equal(t, "/artifacts", b.Config.Workspace.ArtifactPath)
	assert.Equal(t, "/state", b.Config.Workspace.StatePath)
}

func TestDefineDefaultWorkspacePathsAlreadySet(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Workspace: config.Workspace{
				RootPath:     "/",
				FilePath:     "/foo/bar",
				ResourcePath: "/foo/bar",
				ArtifactPath: "/foo/bar",
				StatePath:    "/foo/bar",
			},
		},
	}
	diags := bundle.Apply(context.Background(), b, mutator.DefineDefaultWorkspacePaths())
	require.NoError(t, diags.Error())
	assert.Equal(t, "/foo/bar", b.Config.Workspace.FilePath)
	assert.Equal(t, "/foo/bar", b.Config.Workspace.ResourcePath)
	assert.Equal(t, "/foo/bar", b.Config.Workspace.ArtifactPath)
	assert.Equal(t, "/foo/bar", b.Config.Workspace.StatePath)
}
