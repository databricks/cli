package resources

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSchemaNotFound(t *testing.T) {
	ctx := t.Context()

	w := newTestClients(t, map[string]any{
		"GET /api/2.1/unity-catalog/schemas/non-existent-schema": notFound(),
	})

	s := &Schema{}
	exists, err := s.Exists(ctx, w, "non-existent-schema")

	require.Falsef(t, exists, "Exists should return false when getting a 404 response from Workspace")
	require.NoErrorf(t, err, "Exists should not return an error when getting a 404 response from Workspace")
}
