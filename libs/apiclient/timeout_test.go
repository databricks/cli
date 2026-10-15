package apiclient_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/databricks/cli/libs/apiclient/apiclienttest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHTTPTimeoutIsInactivityTimeout(t *testing.T) {
	c := apiclienttest.New(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/slow" {
			time.Sleep(2 * time.Second)
			return
		}
		// Streams for longer than the timeout, but never stays silent that long.
		flusher := w.(http.Flusher)
		for range 4 {
			_, _ = w.Write([]byte("x"))
			flusher.Flush()
			time.Sleep(500 * time.Millisecond)
		}
	})
	c.Config.HTTPTimeoutSeconds = 1

	var body []byte
	require.NoError(t, c.Do(t.Context(), http.MethodGet, "/stream", nil, nil, nil, &body))
	assert.Equal(t, "xxxx", string(body))

	err := c.Do(t.Context(), http.MethodGet, "/slow", nil, nil, nil, nil)
	assert.ErrorContains(t, err, "request timed out after 1s of inactivity")
}
