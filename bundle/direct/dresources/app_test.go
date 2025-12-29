package dresources

import (
	"context"
	"testing"
	"time"

	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/databricks/databricks-sdk-go/service/apps"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestDoCreate_RetriesOnAlreadyExists(t *testing.T) {
	ctx := context.Background()
	m := mocks.NewMockWorkspaceClient(t)
	appsAPI := m.GetMockAppsAPI()

	callCount := 0
	appsAPI.EXPECT().Create(mock.Anything, mock.Anything).RunAndReturn(
		func(ctx context.Context, req apps.CreateAppRequest) (*apps.WaitGetAppActive[apps.App], error) {
			callCount++
			if callCount == 1 {
				return nil, apierr.ErrResourceAlreadyExists
			}
			return &apps.WaitGetAppActive[apps.App]{Response: &apps.App{Name: "test-app"}}, nil
		},
	)

	r := (&ResourceApp{}).New(m.WorkspaceClient)
	name, _, err := r.DoCreate(ctx, &apps.App{Name: "test-app"})

	require.NoError(t, err)
	assert.Equal(t, "test-app", name)
	assert.Equal(t, 2, callCount, "expected Create to be called twice (1 retry)")
}

func TestDoCreate_FailsAfterTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	m := mocks.NewMockWorkspaceClient(t)
	appsAPI := m.GetMockAppsAPI()

	callCount := 0
	appsAPI.EXPECT().Create(mock.Anything, mock.Anything).RunAndReturn(
		func(ctx context.Context, req apps.CreateAppRequest) (*apps.WaitGetAppActive[apps.App], error) {
			callCount++
			return nil, apierr.ErrResourceAlreadyExists
		},
	)

	r := (&ResourceApp{}).New(m.WorkspaceClient)
	_, _, err := r.DoCreate(ctx, &apps.App{Name: "test-app"})

	require.Error(t, err)
	assert.Greater(t, callCount, 1, "expected Create to be called multiple times before timeout")
}

func TestDoCreate_FailsImmediatelyOnOtherErrors(t *testing.T) {
	ctx := context.Background()
	m := mocks.NewMockWorkspaceClient(t)
	appsAPI := m.GetMockAppsAPI()

	// Return a different error - should not retry
	appsAPI.EXPECT().Create(mock.Anything, mock.Anything).
		Return(nil, apierr.ErrPermissionDenied).Once()

	r := (&ResourceApp{}).New(m.WorkspaceClient)
	_, _, err := r.DoCreate(ctx, &apps.App{Name: "test-app"})

	require.Error(t, err)
	assert.ErrorIs(t, err, apierr.ErrPermissionDenied)
}
