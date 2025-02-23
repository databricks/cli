package variable

import (
	"context"
	"errors"
	"testing"

	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/databricks/databricks-sdk-go/service/settings"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestResolveNotificationDestination_ResolveSuccess(t *testing.T) {
	m := mocks.NewMockWorkspaceClient(t)

	api := m.GetMockNotificationDestinationsAPI()
	api.EXPECT().
		ListAll(mock.Anything, mock.Anything).
		Return([]settings.ListNotificationDestinationsResult{
			{Id: "1234", DisplayName: "destination"},
		}, nil)

	ctx := context.Background()
	l := resolveNotificationDestination{name: "destination"}
	result, err := l.Resolve(ctx, m.WorkspaceClient)
	require.NoError(t, err)
	assert.Equal(t, "1234", result)
}

func TestResolveNotificationDestination_ResolveError(t *testing.T) {
	m := mocks.NewMockWorkspaceClient(t)

	api := m.GetMockNotificationDestinationsAPI()
	api.EXPECT().
		ListAll(mock.Anything, mock.Anything).
		Return(nil, errors.New("bad"))

	ctx := context.Background()
	l := resolveNotificationDestination{name: "destination"}
	_, err := l.Resolve(ctx, m.WorkspaceClient)
	assert.ErrorContains(t, err, "bad")
}

func TestResolveNotificationDestination_ResolveNotFound(t *testing.T) {
	m := mocks.NewMockWorkspaceClient(t)

	api := m.GetMockNotificationDestinationsAPI()
	api.EXPECT().
		ListAll(mock.Anything, mock.Anything).
		Return([]settings.ListNotificationDestinationsResult{}, nil)

	ctx := context.Background()
	l := resolveNotificationDestination{name: "destination"}
	_, err := l.Resolve(ctx, m.WorkspaceClient)
	require.Error(t, err)
	assert.ErrorContains(t, err, `notification destination named "destination" does not exist`)
}

func TestResolveNotificationDestination_ResolveMultiple(t *testing.T) {
	m := mocks.NewMockWorkspaceClient(t)

	api := m.GetMockNotificationDestinationsAPI()
	api.EXPECT().
		ListAll(mock.Anything, mock.Anything).
		Return([]settings.ListNotificationDestinationsResult{
			{Id: "1234", DisplayName: "destination"},
			{Id: "5678", DisplayName: "destination"},
		}, nil)

	ctx := context.Background()
	l := resolveNotificationDestination{name: "destination"}
	_, err := l.Resolve(ctx, m.WorkspaceClient)
	require.Error(t, err)
	assert.ErrorContains(t, err, `there are 2 instances of clusters named "destination"`)
}

func TestResolveNotificationDestination_String(t *testing.T) {
	l := resolveNotificationDestination{name: "name"}
	assert.Equal(t, "notification-destination: name", l.String())
}
