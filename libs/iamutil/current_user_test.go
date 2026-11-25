package iamutil

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"testing"

	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/retry"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestGetCurrentUserRetriesOn500(t *testing.T) {
	m := mocks.NewMockWorkspaceClient(t)
	api := m.GetMockCurrentUserAPI()
	// Separate expectations: a second Return on the same call would replace the first.
	api.EXPECT().Me(mock.Anything, mock.Anything).Return(nil, &apierr.APIError{StatusCode: http.StatusInternalServerError, ErrorCode: "TEMPORARILY_UNAVAILABLE"}).Once()
	api.EXPECT().Me(mock.Anything, mock.Anything).Return(&iam.User{UserName: "test-user"}, nil).Once()

	user, err := GetCurrentUser(testContext(t), m.WorkspaceClient)
	require.NoError(t, err)
	assert.Equal(t, "test-user", user.UserName)
}

func TestGetCurrentUserRetriesOnEOF(t *testing.T) {
	m := mocks.NewMockWorkspaceClient(t)
	api := m.GetMockCurrentUserAPI()
	// Separate expectations: a second Return on the same call would replace the first.
	api.EXPECT().Me(mock.Anything, mock.Anything).Return(nil, &url.Error{Op: "Get", URL: "https://example.test/api/2.0/preview/scim/v2/Me", Err: io.EOF}).Once()
	api.EXPECT().Me(mock.Anything, mock.Anything).Return(&iam.User{UserName: "test-user"}, nil).Once()

	user, err := GetCurrentUser(testContext(t), m.WorkspaceClient)
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

	_, err := GetCurrentUser(testContext(t), m.WorkspaceClient)
	require.Error(t, err)
	var apiErr *apierr.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, http.StatusForbidden, apiErr.StatusCode)
}

// testContext shortens the pause between attempts.
func testContext(t *testing.T) context.Context {
	return env.Set(t.Context(), retry.IntervalMsVariable, "1")
}

func TestGetCurrentUserStopsAfterMaxRetries(t *testing.T) {
	m := mocks.NewMockWorkspaceClient(t)
	eof := &url.Error{Op: "Get", URL: "https://example.test/api/2.0/preview/scim/v2/Me", Err: io.EOF}
	m.GetMockCurrentUserAPI().EXPECT().Me(mock.Anything, mock.Anything).Return(nil, eof).Times(retry.MaxRetries + 1)

	_, err := GetCurrentUser(testContext(t), m.WorkspaceClient)
	assert.ErrorIs(t, err, io.EOF)
}
