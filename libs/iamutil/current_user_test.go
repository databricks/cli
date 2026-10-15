package iamutil

import (
	"net/http"
	"testing"

	"github.com/databricks/cli/libs/apierrs"
	"github.com/databricks/cli/libs/httpfixtures"
	"github.com/databricks/cli/libs/sdkclients"
	"github.com/databricks/cli/libs/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const mePath = "/api/2.0/preview/scim/v2/Me"

func TestGetCurrentUserRetriesOn500(t *testing.T) {
	cfg, _ := httpfixtures.Fixtures{
		{Method: "GET", Resource: mePath, Status: http.StatusInternalServerError, Response: `{"error_code":"TEMPORARILY_UNAVAILABLE"}`},
		{Method: "GET", Resource: mePath, Response: `{"userName":"test-user"}`},
	}.Config(t)
	w, err := sdkclients.New(t.Context(), cfg)
	require.NoError(t, err)

	user, err := GetCurrentUser(t.Context(), w)
	require.NoError(t, err)
	assert.Equal(t, "test-user", utils.Deref(user.UserName))
}

func TestGetCurrentUserDoesNotRetryOn403(t *testing.T) {
	// A non-500 error must fail immediately; a second request would match no fixture.
	cfg, _ := httpfixtures.Fixtures{
		{Method: "GET", Resource: mePath, Status: http.StatusForbidden, Response: `{"message":"denied"}`},
	}.Config(t)
	w, err := sdkclients.New(t.Context(), cfg)
	require.NoError(t, err)

	_, err = GetCurrentUser(t.Context(), w)
	require.Error(t, err)
	assert.Equal(t, http.StatusForbidden, apierrs.StatusCode(err))
}
