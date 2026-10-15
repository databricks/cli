package apiclient_test

import (
	"net/http"
	"testing"

	"github.com/databricks/cli/libs/apiclient"
	"github.com/databricks/cli/libs/apiclient/apiclienttest"
	"github.com/databricks/cli/libs/apierrs"
	"github.com/stretchr/testify/assert"
)

func TestNonJSONErrorBodies(t *testing.T) {
	for _, tc := range []struct {
		status    int
		body      string
		message   string
		errorCode string
	}{
		{http.StatusNotFound, "", "Not Found", ""},
		{http.StatusInternalServerError, "INJECTED", "INJECTED", "UNKNOWN"},
		{http.StatusBadRequest, "INVALID_STATE: cluster is terminated", "cluster is terminated", "INVALID_STATE"},
		{http.StatusServiceUnavailable, "<html><pre>Worker environment not ready.</pre></html>", "Worker environment not ready", "SERVICE_UNAVAILABLE"},
		{http.StatusBadRequest, `{"error_code":"BAD","message":"json"}`, "json", "BAD"},
	} {
		c := apiclienttest.New(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(tc.status)
			_, _ = w.Write([]byte(tc.body))
		})
		ctx := apiclient.WithMaxAttempts(t.Context(), 1)
		err := c.Do(ctx, http.MethodGet, "/api/2.0/x", nil, nil, nil, nil)
		assert.Equal(t, tc.message, apierrs.Message(err), tc.body)
		assert.Equal(t, tc.errorCode, apierrs.ErrorCode(err), tc.body)
	}
}
