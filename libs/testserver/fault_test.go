package testserver_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/databricks/cli/libs/testserver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServerFaultAfterHandlerKeepsTheHandlersEffect(t *testing.T) {
	var calls atomic.Int32
	server := testserver.New(t)
	server.Handle("POST", "/create", func(req testserver.Request) any {
		calls.Add(1)
		return map[string]string{"status": "created"}
	})

	body, err := json.Marshal(map[string]any{
		"pattern":       "POST /create",
		"status_code":   503,
		"body":          `{"error_code": "INJECTED", "message": "Fault injected by test."}`,
		"offset":        0,
		"times":         1,
		"after_handler": true,
	})
	require.NoError(t, err)

	setReq, err := http.NewRequest(http.MethodPost, server.URL+"/__testserver/fault", bytes.NewReader(body))
	require.NoError(t, err)
	setReq.Header.Set("Authorization", "Bearer dbapi-fault-test")
	setResp, err := http.DefaultClient.Do(setReq)
	require.NoError(t, err)
	require.Equal(t, 200, setResp.StatusCode)
	setResp.Body.Close()

	createReq, err := http.NewRequest(http.MethodPost, server.URL+"/create", bytes.NewReader([]byte("{}")))
	require.NoError(t, err)
	createReq.Header.Set("Authorization", "Bearer dbapi-fault-test")
	createResp, err := http.DefaultClient.Do(createReq)
	require.NoError(t, err)
	assert.Equal(t, 503, createResp.StatusCode)
	createResp.Body.Close()

	assert.Equal(t, int32(1), calls.Load())
}
