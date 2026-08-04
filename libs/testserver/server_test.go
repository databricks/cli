package testserver_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/cli/libs/testserver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// recordingT captures Errorf calls instead of failing, so a test can assert whether
// the server would have failed the run, and on what.
type recordingT struct {
	testutil.TestingT
	mu       sync.Mutex
	errCount int
	lastErr  string
}

func (r *recordingT) Errorf(format string, args ...any) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.errCount++
	r.lastErr = fmt.Sprintf(format, args...)
}

func (r *recordingT) errors() (int, string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.errCount, r.lastErr
}

func TestIgnoreUnhandledRequests(t *testing.T) {
	tests := []struct {
		name   string
		ignore bool
	}{
		{"strict by default", false},
		{"ignored when opted in", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rt := &recordingT{TestingT: t}
			s := testserver.New(rt)
			s.IgnoreUnhandledRequests = tt.ignore

			resp, err := http.Get(s.URL + "/api/2.0/no-such-endpoint")
			require.NoError(t, err)
			assert.Equal(t, http.StatusNotImplemented, resp.StatusCode)
			require.NoError(t, resp.Body.Close())

			count, lastErr := rt.errors()
			if tt.ignore {
				assert.Zero(t, count, "unhandled request must not fail the test when ignored: %s", lastErr)
			} else {
				assert.Positive(t, count, "unhandled request must fail the test by default")
				assert.Contains(t, lastErr, "GET /api/2.0/no-such-endpoint")
			}
		})
	}
}

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
