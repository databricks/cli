package testserver_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/databricks/cli/libs/testserver"
	"github.com/stretchr/testify/assert"
)

func TestIsLocalhostProbe(t *testing.T) {
	tests := []struct {
		name   string
		method string
		target string
		host   string
		want   bool
	}{
		{"localhost probe", http.MethodHead, "/", "localhost", true},
		{"localhost probe with port", http.MethodHead, "/", "localhost:8080", true},
		{"cli request to loopback ip", http.MethodGet, "/api/2.0/jobs/list", "127.0.0.1:12345", false},
		{"head to loopback ip", http.MethodHead, "/", "127.0.0.1:12345", false},
		{"get to localhost root", http.MethodGet, "/", "localhost", false},
		{"head to localhost non-root", http.MethodHead, "/api/2.0/jobs/list", "localhost", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(tt.method, tt.target, nil)
			r.Host = tt.host
			assert.Equal(t, tt.want, testserver.IsLocalhostProbe(r))
		})
	}
}
