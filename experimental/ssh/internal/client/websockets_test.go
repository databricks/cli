package client

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildProxyWebsocketURL(t *testing.T) {
	tests := []struct {
		name string
		host string
		want string
	}{
		{
			name: "https host is dialed over wss",
			host: "https://my-workspace.cloud.databricks.test",
			want: "wss://my-workspace.cloud.databricks.test/driver-proxy-api/o/900800700600/1234-567890-abc/7772/ssh?id=conn-1",
		},
		{
			name: "plaintext http host is dialed over ws",
			host: "http://127.0.0.1:8080",
			want: "ws://127.0.0.1:8080/driver-proxy-api/o/900800700600/1234-567890-abc/7772/ssh?id=conn-1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := buildProxyWebsocketURL(tt.host, "900800700600", "1234-567890-abc", 7772, "conn-1")
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
