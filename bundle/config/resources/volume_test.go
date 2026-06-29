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

func TestComputeVolumePath_PureReferenceEmbedded(t *testing.T) {
	v := &Volume{
		CreateVolumeRequestContent: catalog.CreateVolumeRequestContent{
			CatalogName: "main",
			SchemaName:  "${resources.schemas.my.name}",
			Name:        "myvol",
		},
	}
	// A pure reference is embedded verbatim so it can be resolved later (plan/deploy).
	require.Equal(t, "/Volumes/main/${resources.schemas.my.name}/myvol", v.ComputeVolumePath())
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
