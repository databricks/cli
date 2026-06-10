package genie

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/databricks/cli/experimental/genie/agentstream"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildRequest_WireFormat(t *testing.T) {
	// The backend expects camelCase warehouseId; this pins the wire format,
	// not just the Go field values.
	b, err := json.Marshal(BuildRequest("What tables exist?", "abc123"))
	require.NoError(t, err)
	assert.JSONEq(t, `{
		"input": [{
			"type": "message",
			"role": "user",
			"content": [{"type": "input_text", "text": "What tables exist?"}]
		}],
		"warehouseId": "abc123"
	}`, string(b))
}

func TestBuildRequest_OmitsEmptyWarehouse(t *testing.T) {
	b, err := json.Marshal(BuildRequest("q", ""))
	require.NoError(t, err)
	assert.NotContains(t, string(b), "warehouseId")
}

func TestPostStream(t *testing.T) {
	var gotMethod, gotPath, gotAccept, gotContentType string
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotAccept = r.Header.Get("Accept")
		gotContentType = r.Header.Get("Content-Type")
		gotBody, _ = io.ReadAll(r.Body)

		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: {\"type\":\"response.completed\"}\n\n")
	}))
	defer srv.Close()

	cfg := &config.Config{Host: srv.URL, Token: "dummy"}
	body, err := PostStream(t.Context(), cfg, BuildRequest("What tables exist?", "wh1"))
	require.NoError(t, err)
	defer body.Close()

	assert.Equal(t, "POST", gotMethod)
	assert.Equal(t, genieResponsesPath, gotPath)
	assert.Equal(t, "text/event-stream", gotAccept)
	assert.Equal(t, "application/json", gotContentType)
	assert.Contains(t, string(gotBody), `"warehouseId":"wh1"`)
	assert.Equal(t, StreamingTimeoutSeconds, cfg.HTTPTimeoutSeconds)

	ev, err := agentstream.NewSSEReader(body).Next()
	require.NoError(t, err)
	assert.JSONEq(t, `{"type":"response.completed"}`, ev.Data)
}

func TestPostStream_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, `{"error_code":"INTERNAL_ERROR","message":"backend exploded"}`)
	}))
	defer srv.Close()

	cfg := &config.Config{Host: srv.URL, Token: "dummy"}
	_, err := PostStream(t.Context(), cfg, BuildRequest("q", ""))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "backend exploded")
}
