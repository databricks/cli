package testserver_test

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/cli/libs/testserver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// recordingT wraps a real TestingT but records Errorf calls instead of failing,
// so a test can assert whether the server would have failed the run.
type recordingT struct {
	testutil.TestingT
	mu       sync.Mutex
	errCount int
}

func (r *recordingT) Errorf(format string, args ...any) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.errCount++
}

func (r *recordingT) errors() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.errCount
}

func TestIgnoreUnhandledRequests(t *testing.T) {
	for _, ignore := range []bool{false, true} {
		rt := &recordingT{TestingT: t}
		s := testserver.New(rt)
		s.IgnoreUnhandledRequests = ignore

		resp, err := http.Get(s.URL + "/api/2.0/no-such-endpoint")
		require.NoError(t, err)
		assert.Equal(t, http.StatusNotImplemented, resp.StatusCode)
		require.NoError(t, resp.Body.Close())

		if ignore {
			assert.Zero(t, rt.errors(), "unhandled request must not fail the test when ignored")
		} else {
			assert.Positive(t, rt.errors(), "unhandled request must fail the test by default")
		}
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
