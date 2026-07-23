package aircmd

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/databricks/databricks-sdk-go/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetBricklensLogsQuerySerialization(t *testing.T) {
	var got url.Values
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/oidc/.well-known/oauth-authorization-server" {
			gotPath = r.URL.Path
			got = r.URL.Query()
		}
		_, _ = w.Write([]byte(`{"log_records": [], "next_page_token": ""}`))
	}))
	t.Cleanup(srv.Close)

	w := newTestWorkspaceClient(t, srv.URL)
	apiClient, err := client.New(w.Config)
	require.NoError(t, err)
	_, err = getBricklensLogs(t.Context(), apiClient, 42, bricklensLogsQuery{
		fromSeconds:   100,
		toSeconds:     200,
		pageToken:     "tok",
		pageSize:      500,
		attemptNumber: 1,
		nodeIndex:     3,
		ascending:     true,
	})
	require.NoError(t, err)

	assert.Equal(t, "/api/2.0/ai-training/workflows/by-run-id/42/logs", gotPath)
	assert.Equal(t, "100", got.Get("from"))
	assert.Equal(t, "200", got.Get("to"))
	assert.Equal(t, "tok", got.Get("page_token"))
	assert.Equal(t, "500", got.Get("page_size"))
	assert.Equal(t, "1", got.Get("ref.attempt_number"))
	assert.Equal(t, "3", got.Get("filter.node_index"))
	assert.Equal(t, "true", got.Get("ascending"))
}

func TestGetBricklensLogsOmitsOptionals(t *testing.T) {
	var got url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/oidc/.well-known/oauth-authorization-server" {
			got = r.URL.Query()
		}
		_, _ = w.Write([]byte(`{"log_records": []}`))
	}))
	t.Cleanup(srv.Close)

	w := newTestWorkspaceClient(t, srv.URL)
	apiClient, err := client.New(w.Config)
	require.NoError(t, err)
	// attempt -1 (latest) and node 0 are the default request; from/to/page unset.
	_, err = getBricklensLogs(t.Context(), apiClient, 7, bricklensLogsQuery{attemptNumber: -1, nodeIndex: 0})
	require.NoError(t, err)

	assert.False(t, got.Has("from"))
	assert.False(t, got.Has("to"))
	assert.False(t, got.Has("page_token"))
	assert.False(t, got.Has("page_size"))
	// -1 attempt means "latest" — the field is omitted so the endpoint defaults.
	assert.False(t, got.Has("ref.attempt_number"))
	// node 0 is a real filter value and must be sent.
	assert.Equal(t, "0", got.Get("filter.node_index"))
	// ascending is always sent so the tail path can force newest-first.
	assert.Equal(t, "false", got.Get("ascending"))
}

func TestLogRecordNano(t *testing.T) {
	assert.Equal(t, int64(123), logRecord{TimeUnixNano: "123"}.nano())
	assert.Equal(t, int64(0), logRecord{TimeUnixNano: ""}.nano())
	assert.Equal(t, int64(0), logRecord{TimeUnixNano: "notanumber"}.nano())
}
