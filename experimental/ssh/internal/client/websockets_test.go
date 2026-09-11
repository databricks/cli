package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/databricks/cli/experimental/ssh/internal/proxy"
	"github.com/databricks/databricks-sdk-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServerSupportsResume(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status int
		body   string
		want   bool
	}{
		{"old server", http.StatusNotFound, "not found", false},
		{"faulty v1 server", http.StatusOK, `{"resume":true}`, false},
		{"corrected protocol", http.StatusOK, `{"resume_version":2}`, true},
		{"unknown protocol", http.StatusOK, `{"resume_version":3}`, false},
		{"invalid response", http.StatusOK, `{`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mux := http.NewServeMux()
			mux.HandleFunc("GET /driver-proxy-api/o/123/cluster/7772/capabilities", func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			})
			server := httptest.NewServer(mux)
			defer server.Close()
			client, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "test-token", WorkspaceID: "123", AuthType: "pat"})
			require.NoError(t, err)
			assert.Equal(t, tc.want, serverSupportsResume(t.Context(), client, "cluster", 7772, ""))
		})
	}
}

func TestCreateWebsocketConnectionReattachRejected(t *testing.T) {
	for _, status := range []int{http.StatusGone, http.StatusConflict, http.StatusUnauthorized, http.StatusTooManyRequests, http.StatusServiceUnavailable} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			mux := http.NewServeMux()
			mux.HandleFunc("GET /driver-proxy-api/o/123/cluster/7772/ssh", func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(status)
			})
			server := httptest.NewServer(mux)
			defer server.Close()
			client, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "test-token", WorkspaceID: "123", AuthType: "pat"})
			require.NoError(t, err)
			ctx, cancel := context.WithTimeout(t.Context(), time.Second)
			defer cancel()
			_, err = createWebsocketConnection(ctx, client, proxy.DialRequest{ConnID: "test", ResumeCapable: true, Reattach: true}, "cluster", 7772, "")
			require.Error(t, err)
			if status < 500 && status != http.StatusTooManyRequests {
				assert.ErrorIs(t, err, proxy.ErrReattachRejected)
			} else {
				assert.NotErrorIs(t, err, proxy.ErrReattachRejected)
			}
		})
	}
}

func TestBuildProxyWebsocketURL(t *testing.T) {
	tests := []struct {
		name string
		host string
		dial proxy.DialRequest
		want string
	}{
		{
			name: "https host is dialed over wss",
			host: "https://my-workspace.cloud.databricks.test",
			dial: proxy.DialRequest{ConnID: "conn-1"},
			want: "wss://my-workspace.cloud.databricks.test/driver-proxy-api/o/900800700600/1234-567890-abc/7772/ssh?id=conn-1",
		},
		{
			name: "plaintext http host is dialed over ws",
			host: "http://127.0.0.1:8080",
			dial: proxy.DialRequest{ConnID: "conn-1"},
			want: "ws://127.0.0.1:8080/driver-proxy-api/o/900800700600/1234-567890-abc/7772/ssh?id=conn-1",
		},
		{
			// A server that does not speak the resume protocol must see the URL it always saw,
			// so no resume parameters leak out when the capability probe said no.
			name: "a non-resumable dial carries no resume parameters",
			host: "http://127.0.0.1:8080",
			dial: proxy.DialRequest{ConnID: "conn-1", Delivered: 4096, Reattach: true},
			want: "ws://127.0.0.1:8080/driver-proxy-api/o/900800700600/1234-567890-abc/7772/ssh?id=conn-1",
		},
		{
			// The offset travels on every dial, not just a reattach: its presence is what tells
			// the server to start buffering its own output for replay.
			name: "a resumable dial always carries the delivered offset",
			host: "http://127.0.0.1:8080",
			dial: proxy.DialRequest{ConnID: "conn-1", ResumeCapable: true},
			want: "ws://127.0.0.1:8080/driver-proxy-api/o/900800700600/1234-567890-abc/7772/ssh?delivered=0&id=conn-1&resume_version=2",
		},
		{
			name: "a reattach states its intent and its offset",
			host: "http://127.0.0.1:8080",
			dial: proxy.DialRequest{ConnID: "conn-1", ResumeCapable: true, Delivered: 4096, Reattach: true},
			want: "ws://127.0.0.1:8080/driver-proxy-api/o/900800700600/1234-567890-abc/7772/ssh?delivered=4096&id=conn-1&reattach=1&resume_version=2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := buildProxyWebsocketURL(tt.host, "900800700600", "1234-567890-abc", 7772, tt.dial)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
