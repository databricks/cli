package resources

import (
	"testing"

	"github.com/databricks/cli/libs/testserver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExternalLocationExists(t *testing.T) {
	ctx := t.Context()
	w := newTestClients(t, map[string]any{
		"GET /api/2.1/unity-catalog/external-locations/test_location": map[string]any{"name": "test_location"},
		"GET /api/2.1/unity-catalog/external-locations/nonexistent": testserver.Response{
			StatusCode: 404,
			Body:       map[string]string{"error_code": "RESOURCE_DOES_NOT_EXIST", "message": "not found"},
		},
	})

	t.Run("exists", func(t *testing.T) {
		el := &ExternalLocation{}
		exists, err := el.Exists(ctx, w, "test_location")
		require.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("does not exist", func(t *testing.T) {
		el := &ExternalLocation{}
		exists, err := el.Exists(ctx, w, "nonexistent")
		require.NoError(t, err)
		assert.False(t, exists)
	})
}

func TestExternalLocationResourceDescription(t *testing.T) {
	el := &ExternalLocation{}
	desc := el.ResourceDescription()

	assert.Equal(t, "external_location", desc.SingularName)
	assert.Equal(t, "external_locations", desc.PluralName)
	assert.Equal(t, "External Location", desc.SingularTitle)
	assert.Equal(t, "External Locations", desc.PluralTitle)
}

func TestExternalLocationGetName(t *testing.T) {
	el := &ExternalLocation{
		Name: new("my_location"),
	}

	assert.Equal(t, "my_location", el.GetName())
}
