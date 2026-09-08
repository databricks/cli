package client

import (
	"testing"

	"github.com/databricks/cli/experimental/ssh/internal/proxy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
			want: "ws://127.0.0.1:8080/driver-proxy-api/o/900800700600/1234-567890-abc/7772/ssh?delivered=0&id=conn-1",
		},
		{
			name: "a reattach states its intent and its offset",
			host: "http://127.0.0.1:8080",
			dial: proxy.DialRequest{ConnID: "conn-1", ResumeCapable: true, Delivered: 4096, Reattach: true},
			want: "ws://127.0.0.1:8080/driver-proxy-api/o/900800700600/1234-567890-abc/7772/ssh?delivered=4096&id=conn-1&reattach=1",
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
