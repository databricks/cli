package libraries

import (
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/internal/bundletest"
	"github.com/databricks/cli/libs/filer"
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

	w, _ := bundletest.NewClients(t)
	b.SetWorkpaceClient(w)

	client, uploadPath, diags := GetFilerForLibraries(t.Context(), b)
	require.NoError(t, diags.Error())
	assert.Equal(t, "/Workspace/foo/bar/artifacts/.internal", uploadPath)

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

	w, _ := bundletest.NewClients(t)
	b.SetWorkpaceClient(w)

	client, uploadPath, diags := GetFilerForLibrariesCleanup(t.Context(), b)
	require.NoError(t, diags.Error())
	assert.Equal(t, "/Workspace/foo/bar/artifacts", uploadPath)

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

	w, _ := bundletest.NewClients(t)
	b.SetWorkpaceClient(w)

	client, uploadPath, diags := GetFilerForLibraries(t.Context(), b)
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

	w, _ := bundletest.NewClients(t)
	b.SetWorkpaceClient(w)

	client, uploadPath, diags := GetFilerForLibrariesCleanup(t.Context(), b)
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

	w, _ := bundletest.NewClients(t)
	b.SetWorkpaceClient(w)

	_, _, diags := GetFilerForLibraries(t.Context(), b)
	require.EqualError(t, diags.Error(), "remote artifact path not configured")
}
