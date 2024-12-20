package libraries

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/filer"
	sdkconfig "github.com/databricks/databricks-sdk-go/config"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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

	client, uploadPath, diags := GetFilerForLibraries(context.Background(), b)
	require.NoError(t, diags.Error())
	assert.Equal(t, "/foo/bar/artifacts/.internal", uploadPath)

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
	m.WorkspaceClient.Config = &sdkconfig.Config{}
	m.GetMockFilesAPI().EXPECT().GetDirectoryMetadataByDirectoryPath(mock.Anything, "/Volumes/main/my_schema/my_volume").Return(nil)
	b.SetWorkpaceClient(m.WorkspaceClient)

	client, uploadPath, diags := GetFilerForLibraries(context.Background(), b)
	require.NoError(t, diags.Error())
	assert.Equal(t, "/Volumes/main/my_schema/my_volume/.internal", uploadPath)

	assert.IsType(t, &filer.FilesClient{}, client)
}

func TestGetFilerForLibrariesRemotePathNotSet(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Workspace: config.Workspace{},
		},
	}

	_, _, diags := GetFilerForLibraries(context.Background(), b)
	require.EqualError(t, diags.Error(), "remote artifact path not configured")
}
