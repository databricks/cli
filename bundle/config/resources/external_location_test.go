package resources

import (
	"context"
	"testing"

	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestExternalLocationExists(t *testing.T) {
	ctx := context.Background()
	m := mocks.NewMockWorkspaceClient(t)
	api := m.GetMockExternalLocationsAPI()

	t.Run("exists", func(t *testing.T) {
		api.EXPECT().
			GetByName(mock.Anything, "test_location").
			Return(&catalog.ExternalLocationInfo{Name: "test_location"}, nil)

		el := &ExternalLocation{}
		exists, err := el.Exists(ctx, m.WorkspaceClient, "test_location")
		require.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("does not exist", func(t *testing.T) {
		notFoundErr := &apierr.APIError{
			StatusCode: 404,
			ErrorCode:  "RESOURCE_DOES_NOT_EXIST",
		}
		api.EXPECT().
			GetByName(mock.Anything, "nonexistent").
			Return(nil, notFoundErr)

		el := &ExternalLocation{}
		exists, err := el.Exists(ctx, m.WorkspaceClient, "nonexistent")
		require.NoError(t, err)
		assert.False(t, exists)
	})
}

func TestExternalLocationResourceDescription(t *testing.T) {
	el := &ExternalLocation{}
	desc := el.ResourceDescription()

	assert.Equal(t, "external location", desc.SingularName)
	assert.Equal(t, "external_locations", desc.PluralName)
	assert.Equal(t, "External Location", desc.SingularTitle)
	assert.Equal(t, "External Locations", desc.PluralTitle)
}

func TestExternalLocationGetName(t *testing.T) {
	el := &ExternalLocation{
		CreateExternalLocation: catalog.CreateExternalLocation{
			Name: "my_location",
		},
	}

	assert.Equal(t, "my_location", el.GetName())
}
