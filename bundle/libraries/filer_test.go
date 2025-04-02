package libraries

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/filer"
	databrickscfg "github.com/databricks/databricks-sdk-go/config"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetFilerForLibrariesValidWsfs(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Workspace: config.Workspace{
				ArtifactPath: "/foo/bar/artifacts",
			},
		},
	}

	m := mocks.NewMockWorkspaceClient(t)
	m.WorkspaceClient.Config = &databrickscfg.Config{}
	b.SetWorkpaceClient(m.WorkspaceClient)

	client, uploadPath, diags := GetFilerForLibraries(context.Background(), b)
	require.NoError(t, diags.Error())
	assert.Equal(t, "/foo/bar/artifacts/.internal", uploadPath)

	assert.IsType(t, &filer.WorkspaceFilesClient{}, client)
}

func TestGetFilerForLibrariesCleanupValidWsfs(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Workspace: config.Workspace{
				ArtifactPath: "/foo/bar/artifacts",
			},
		},
	}

	m := mocks.NewMockWorkspaceClient(t)
	m.WorkspaceClient.Config = &databrickscfg.Config{}
	b.SetWorkpaceClient(m.WorkspaceClient)

	client, uploadPath, diags := GetFilerForLibrariesCleanup(context.Background(), b)
	require.NoError(t, diags.Error())
	assert.Equal(t, "/foo/bar/artifacts", uploadPath)

	assert.IsType(t, &filer.WorkspaceFilesClient{}, client)
}

func TestGetFilerForLibrariesValidUcVolume(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Workspace: config.Workspace{
				ArtifactPath: "/Volumes/main/my_schema/my_volume",
			},
		},
	}

	m := mocks.NewMockWorkspaceClient(t)
	m.WorkspaceClient.Config = &databrickscfg.Config{}
	b.SetWorkpaceClient(m.WorkspaceClient)

	client, uploadPath, diags := GetFilerForLibraries(context.Background(), b)
	require.NoError(t, diags.Error())
	assert.Equal(t, "/Volumes/main/my_schema/my_volume/.internal", uploadPath)

	assert.IsType(t, &filer.FilesClient{}, client)
}

func TestGetFilerForLibrariesCleanupValidUcVolume(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Workspace: config.Workspace{
				ArtifactPath: "/Volumes/main/my_schema/my_volume",
			},
		},
	}

	m := mocks.NewMockWorkspaceClient(t)
	m.WorkspaceClient.Config = &databrickscfg.Config{}
	b.SetWorkpaceClient(m.WorkspaceClient)

	client, uploadPath, diags := GetFilerForLibrariesCleanup(context.Background(), b)
	require.NoError(t, diags.Error())
	assert.Equal(t, "/Volumes/main/my_schema/my_volume", uploadPath)

	assert.IsType(t, &filer.FilesClient{}, client)
}

func TestGetFilerForLibrariesRemotePathNotSet(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Workspace: config.Workspace{},
		},
	}

	m := mocks.NewMockWorkspaceClient(t)
	m.WorkspaceClient.Config = &databrickscfg.Config{}
	b.SetWorkpaceClient(m.WorkspaceClient)

	_, _, diags := GetFilerForLibraries(context.Background(), b)
	require.EqualError(t, diags.Error(), "remote artifact path not configured")
}
