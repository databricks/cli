package qualitymonitors

import (
	"net/http"
	"testing"

	"github.com/databricks/cli/libs/apiclient/apiclienttest"
	"github.com/databricks/cli/libs/apierrs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetMissingMonitor(t *testing.T) {
	c := New(apiclienttest.New(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/2.1/unity-catalog/tables/main.default.t/monitor", r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error_code":"RESOURCE_DOES_NOT_EXIST","message":"no monitor"}`))
	}))
	_, err := c.Get(t.Context(), GetQualityMonitorRequest{TableName: "main.default.t"})
	require.Error(t, err)
	assert.True(t, apierrs.Is(err, apierrs.ErrResourceDoesNotExist))
}
