package testserver_test

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/databricks/cli/libs/testserver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newCountingServer returns a server whose handler counts the requests it applied.
func newCountingServer(t *testing.T) (*testserver.Server, *int) {
	s := testserver.New(t)
	applied := 0
	s.Handle("POST", "/api/2.0/things", func(req testserver.Request) any {
		applied++
		return map[string]string{"id": "thing-with-a-long-enough-id"}
	})
	return s, &applied
}

// postThing returns the status of the response, after reading its body.
func postThing(s *testserver.Server) (int, error) {
	// A fresh connection per request, so a fault on one does not affect the next.
	client := &http.Client{Transport: &http.Transport{DisableKeepAlives: true}}
	resp, err := client.Post(s.URL+"/api/2.0/things", "application/json", strings.NewReader("{}"))
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	_, err = io.ReadAll(resp.Body)
	return resp.StatusCode, err
}

func TestRequestFaultStatusKinds(t *testing.T) {
	for kind, wantStatus := range map[string]int{
		testserver.FaultTooManyRequests:    429,
		testserver.FaultInternalError:      500,
		testserver.FaultBadGateway:         502,
		testserver.FaultUnavailable:        503,
		testserver.FaultGatewayTimeout:     504,
		testserver.FaultGatewayTimeoutLate: 504,
	} {
		t.Run(kind, func(t *testing.T) {
			s, applied := newCountingServer(t)
			require.NoError(t, s.SetRequestFault(2, kind))

			status, err := postThing(s)
			require.NoError(t, err)
			assert.Equal(t, 200, status)

			status, err = postThing(s)
			require.NoError(t, err)
			assert.Equal(t, wantStatus, status)

			status, err = postThing(s)
			require.NoError(t, err)
			assert.Equal(t, 200, status)

			// Only the -after kind lets the handler apply the failed request.
			wantApplied := 2
			if kind == testserver.FaultGatewayTimeoutLate {
				wantApplied = 3
			}
			assert.Equal(t, wantApplied, *applied)
			assert.Equal(t, 3, s.RequestCount())
			assert.Equal(t, &testserver.FiredRequestFault{At: 2, Kind: kind, Method: "POST", Path: "/api/2.0/things"}, s.FiredRequestFault())
		})
	}
}

func TestRequestFaultConnectionKinds(t *testing.T) {
	for kind, wantErr := range map[string]string{
		testserver.FaultReset:         "connection reset by peer",
		testserver.FaultResetLate:     "connection reset by peer",
		testserver.FaultTruncatedLate: "unexpected EOF",
	} {
		t.Run(kind, func(t *testing.T) {
			s, applied := newCountingServer(t)
			require.NoError(t, s.SetRequestFault(1, kind))

			_, err := postThing(s)
			require.Error(t, err)
			// Depending on timing, a reset surfaces as EOF when the client had not started reading.
			if kind == testserver.FaultTruncatedLate {
				assert.ErrorContains(t, err, wantErr)
			} else {
				assert.Regexp(t, wantErr+"|EOF", err.Error())
			}

			status, err := postThing(s)
			require.NoError(t, err)
			assert.Equal(t, 200, status)

			wantApplied := 1
			if kind != testserver.FaultReset {
				wantApplied = 2
			}
			assert.Equal(t, wantApplied, *applied)
		})
	}
}

func TestRequestFaultSkipsTestserverEndpoints(t *testing.T) {
	s, _ := newCountingServer(t)
	require.NoError(t, s.SetRequestFault(1, testserver.FaultUnavailable))

	resp, err := http.Post(s.URL+"/__testserver/kill", "application/json", strings.NewReader(`{"pattern": "GET /none", "times": 0}`))
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, 0, s.RequestCount())
	assert.Nil(t, s.FiredRequestFault())

	status, err := postThing(s)
	require.NoError(t, err)
	assert.Equal(t, 503, status)
}

func TestSetRequestFaultValidates(t *testing.T) {
	s := testserver.New(t)
	assert.ErrorContains(t, s.SetRequestFault(0, testserver.FaultUnavailable), "at least 1")
	assert.ErrorContains(t, s.SetRequestFault(1, "418"), `unknown request fault kind "418"`)
}
