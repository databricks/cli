package resources

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestComputeVolumePath(t *testing.T) {
	v := &Volume{
		CatalogName: new("main"),
		SchemaName:  new("myschema"),
		Name:        new("myvol"),
	}
	require.Equal(t, "/Volumes/main/myschema/myvol", v.ComputeVolumePath())
}

func TestComputeVolumePath_PureReferenceEmbedded(t *testing.T) {
	v := &Volume{
		CatalogName: new("main"),
		SchemaName:  new("${resources.schemas.my.name}"),
		Name:        new("myvol"),
	}
	// A pure reference is embedded verbatim so it can be resolved later (plan/deploy).
	require.Equal(t, "/Volumes/main/${resources.schemas.my.name}/myvol", v.ComputeVolumePath())
}

func TestVolumeNotFound(t *testing.T) {
	ctx := t.Context()

	w := newTestClients(t, map[string]any{
		"GET /api/2.1/unity-catalog/volumes/non-existent-volume": notFound(),
	})

	s := &Volume{}
	exists, err := s.Exists(ctx, w, "non-existent-volume")

	require.Falsef(t, exists, "Exists should return false when getting a 404 response from Workspace")
	require.NoErrorf(t, err, "Exists should not return an error when getting a 404 response from Workspace")
}
