package iamutil

import (
	"net/http"
	"testing"

	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestGetCurrentUserRetriesOn500(t *testing.T) {
	m := mocks.NewMockWorkspaceClient(t)
	me := m.GetMockCurrentUserAPI().EXPECT().Me(mock.Anything, mock.Anything)
	me.Return(nil, &apierr.APIError{StatusCode: http.StatusInternalServerError, ErrorCode: "TEMPORARILY_UNAVAILABLE"}).Once()
	me.Return(&iam.User{UserName: "test-user"}, nil).Once()

	user, err := GetCurrentUser(t.Context(), m.WorkspaceClient)
	require.NoError(t, err)
	assert.Equal(t, "test-user", user.UserName)
}

func TestGetCurrentUserDoesNotRetryOn403(t *testing.T) {
	m := mocks.NewMockWorkspaceClient(t)
	// A non-500 error must fail immediately, without a retry (Once enforces it).
	m.GetMockCurrentUserAPI().EXPECT().
		Me(mock.Anything, mock.Anything).
		Return(nil, &apierr.APIError{StatusCode: http.StatusForbidden, Message: "denied"}).
		Once()

	_, err := GetCurrentUser(t.Context(), m.WorkspaceClient)
	require.Error(t, err)
	var apiErr *apierr.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, http.StatusForbidden, apiErr.StatusCode)
}
