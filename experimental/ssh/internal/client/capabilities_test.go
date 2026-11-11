package client

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/databricks/databricks-sdk-go"
	"github.com/stretchr/testify/require"
)

// The capabilities probe runs before the tunnel is dialled, on every proxy-mode connect. A driver
// proxy that accepts the request and never answers must not hold the connect path open: resume is
// an optimisation, and failing to learn about it has to cost bounded time, not the whole session.
func TestServerSupportsResumeGivesUpOnAStalledServer(t *testing.T) {
	restore := capabilitiesProbeTimeout
	capabilitiesProbeTimeout = 100 * time.Millisecond
	defer func() { capabilitiesProbeTimeout = restore }()

	stalled := make(chan struct{})

	mux := http.NewServeMux()
	mux.HandleFunc("GET /driver-proxy-api/o/123/cluster/7772/capabilities", func(w http.ResponseWriter, r *http.Request) {
		<-stalled
	})
	server := httptest.NewServer(mux)
	// LIFO: release the stalled handler first, so Close does not wait on it.
	defer server.Close()
	defer close(stalled)

	client, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "test-token", WorkspaceID: "123", AuthType: "pat"})
	require.NoError(t, err)

	done := make(chan bool, 1)
	go func() {
		done <- serverSupportsResume(t.Context(), client, "cluster", 7772, "")
	}()

	select {
	case resumable := <-done:
		require.False(t, resumable, "a server that never answered cannot be assumed to support resume")
	case <-time.After(time.Second):
		t.Fatal("the capabilities probe never returned: it has no timeout of its own, so a stalled driver proxy blocks the connect path indefinitely")
	}
}
