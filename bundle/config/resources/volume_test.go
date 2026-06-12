package resources

import (
	"testing"

	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestComputeVolumePath(t *testing.T) {
	v := &Volume{
		CreateVolumeRequestContent: catalog.CreateVolumeRequestContent{
			CatalogName: "main",
			SchemaName:  "myschema",
			Name:        "myvol",
		},
	}
	require.Equal(t, "/Volumes/main/myschema/myvol", v.ComputeVolumePath())
}

func TestComputeVolumePath_UnresolvedReference(t *testing.T) {
	v := &Volume{
		CreateVolumeRequestContent: catalog.CreateVolumeRequestContent{
			CatalogName: "main",
			SchemaName:  "${resources.schemas.my.name}",
			Name:        "myvol",
		},
	}
	require.Empty(t, v.ComputeVolumePath())
}

func TestComputeVolumePath_MissingField(t *testing.T) {
	v := &Volume{
		CreateVolumeRequestContent: catalog.CreateVolumeRequestContent{
			CatalogName: "main",
			Name:        "myvol",
		},
	}
	require.Empty(t, v.ComputeVolumePath())
}

func TestVolumeNotFound(t *testing.T) {
	ctx := t.Context()

	m := mocks.NewMockWorkspaceClient(t)
	m.GetMockVolumesAPI().On("Read", mock.Anything, mock.Anything).Return(nil, &apierr.APIError{
		StatusCode: 404,
	})

	s := &Volume{}
	exists, err := s.Exists(ctx, m.WorkspaceClient, "non-existent-volume")

	require.Falsef(t, exists, "Exists should return false when getting a 404 response from Workspace")
	require.NoErrorf(t, err, "Exists should not return an error when getting a 404 response from Workspace")
}
