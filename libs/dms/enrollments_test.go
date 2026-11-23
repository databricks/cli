package dms

import (
	"net/http"
	"testing"

	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func enrollmentsClient(t *testing.T, server *testserver.Server) *databricks.WorkspaceClient {
	t.Helper()
	w, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "token"})
	require.NoError(t, err)
	return w
}

func TestCheckEnrollmentsOmitsUnknownFeatures(t *testing.T) {
	server := testserver.New(t)
	testserver.AddDefaultHandlers(server)

	got, err := CheckEnrollments(t.Context(), enrollmentsClient(t, server), FeatureRecordHistoryAuto, "not_a_feature")
	require.NoError(t, err)
	assert.Equal(t, map[string]bool{FeatureRecordHistoryAuto: false}, got)
}

func TestCheckEnrollmentsSendsEachFeatureAsItsOwnParameter(t *testing.T) {
	server := testserver.New(t)
	var features []string
	server.Handle("GET", checkEnrollmentsPath, func(req testserver.Request) any {
		features = req.URL.Query()["features"]
		return map[string]any{"enrollments": map[string]bool{FeatureRecordHistoryAuto: true}}
	})

	got, err := CheckEnrollments(t.Context(), enrollmentsClient(t, server), FeatureRecordHistoryAuto, "another")
	require.NoError(t, err)
	assert.Equal(t, []string{FeatureRecordHistoryAuto, "another"}, features)
	assert.Equal(t, map[string]bool{FeatureRecordHistoryAuto: true}, got)
}

func TestCheckEnrollmentsWithNoFeaturesMakesNoRequest(t *testing.T) {
	server := testserver.New(t)
	server.Handle("GET", checkEnrollmentsPath, func(testserver.Request) any {
		t.Error("unexpected request")
		return nil
	})

	got, err := CheckEnrollments(t.Context(), enrollmentsClient(t, server))
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestCheckEnrollmentsReturnsTheServiceError(t *testing.T) {
	server := testserver.New(t)
	server.Handle("GET", checkEnrollmentsPath, func(testserver.Request) any {
		return testserver.Response{
			StatusCode: http.StatusForbidden,
			Headers:    map[string][]string{"Content-Type": {"application/json"}},
			Body:       map[string]string{"error_code": "PERMISSION_DENIED", "message": "not opted in"},
		}
	})

	_, err := CheckEnrollments(t.Context(), enrollmentsClient(t, server), FeatureRecordHistoryAuto)
	assert.ErrorIs(t, err, apierr.ErrPermissionDenied)
}
