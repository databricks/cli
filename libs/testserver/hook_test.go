package testserver_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/databricks-sdk-go/service/vectorsearch"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const hookTestToken = "dbapi-hook-test"

// sendTestRequest sends a request with hookTestToken and returns the status and body of the response.
func sendTestRequest(t *testing.T, server *testserver.Server, method, path string, body any) (int, []byte) {
	t.Helper()
	encoded, err := json.Marshal(body)
	require.NoError(t, err)
	req, err := http.NewRequest(method, server.URL+path, bytes.NewReader(encoded))
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+hookTestToken)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	return resp.StatusCode, respBody
}

func TestServerHookSendsItsRequestBeforeTheMatchingRequestIsAnswered(t *testing.T) {
	var events []string
	server := testserver.New(t)
	server.Handle("GET", "/things/{name}", func(req testserver.Request) any {
		events = append(events, "GET "+req.Vars["name"])
		return testserver.Response{StatusCode: 404, Body: map[string]string{"message": "not found"}}
	})
	server.Handle("POST", "/things", func(req testserver.Request) any {
		events = append(events, "POST "+string(req.Body)+" as "+req.Token)
		return map[string]string{"status": "created"}
	})

	status, _ := sendTestRequest(t, server, "POST", "/__testserver/hook", map[string]any{
		"pattern": "GET /things/*",
		"offset":  1,
		"times":   1,
		"method":  "POST",
		"path":    "/things",
		"body":    "x",
	})
	require.Equal(t, 200, status)

	// The offset lets the first matching request through.
	status, _ = sendTestRequest(t, server, "GET", "/things/a", nil)
	assert.Equal(t, 404, status)
	assert.Equal(t, []string{"GET a"}, events)

	// The hook's request is handled after the handler of the matching request and before its
	// response, with the same token; the matching request keeps its own response.
	status, _ = sendTestRequest(t, server, "GET", "/things/b", nil)
	assert.Equal(t, 404, status)
	assert.Equal(t, []string{"GET a", "GET b", "POST x as " + hookTestToken}, events)

	// times is used up.
	sendTestRequest(t, server, "GET", "/things/c", nil)
	assert.Equal(t, []string{"GET a", "GET b", "POST x as " + hookTestToken, "GET c"}, events)
}

// A hook whose request matches its own pattern runs as many times as it was given, each run
// inside the request of the one before, and then stops.
func TestServerHookRunsItsTimesWhenItsRequestMatchesItsOwnPattern(t *testing.T) {
	for _, times := range []int{1, 2} {
		calls := 0
		server := testserver.New(t)
		server.Handle("POST", "/things", func(req testserver.Request) any {
			calls++
			return map[string]string{"status": "created"}
		})

		status, _ := sendTestRequest(t, server, "POST", "/__testserver/hook", map[string]any{
			"pattern": "POST /things",
			"times":   times,
			"method":  "POST",
			"path":    "/things",
		})
		require.Equal(t, 200, status)

		sendTestRequest(t, server, "POST", "/things", nil)
		assert.Equal(t, 1+times, calls)

		sendTestRequest(t, server, "POST", "/things", nil)
		assert.Equal(t, 2+times, calls)
	}
}

func TestServerHookRejectsARuleWithoutARequest(t *testing.T) {
	server := testserver.New(t)
	status, _ := sendTestRequest(t, server, "POST", "/__testserver/hook", map[string]any{"pattern": "GET /things", "times": 1})
	assert.Equal(t, 400, status)
}

// The create times are compared with each other, not with the clock of the test: the testserver
// hands out strictly increasing times.
func TestServerHookShiftsTheTimeOfItsRequest(t *testing.T) {
	const offsetMs = -30_000
	server := testserver.New(t)
	testserver.AddDefaultHandlers(server)

	createEndpoint := func(name string) {
		status, body := sendTestRequest(t, server, "POST", "/api/2.0/vector-search/endpoints", map[string]string{"name": name, "endpoint_type": "STANDARD"})
		require.Equal(t, 200, status, string(body))
	}
	createdAt := func(name string) int64 {
		status, body := sendTestRequest(t, server, "GET", "/api/2.0/vector-search/endpoints/"+name, nil)
		require.Equal(t, 200, status, string(body))
		var info vectorsearch.EndpointInfo
		require.NoError(t, json.Unmarshal(body, &info))
		return info.CreationTimestamp
	}

	createEndpoint("before")
	status, _ := sendTestRequest(t, server, "POST", "/__testserver/hook", map[string]any{
		"pattern":         "GET /api/2.0/vector-search/endpoints/before",
		"times":           1,
		"method":          "POST",
		"path":            "/api/2.0/vector-search/endpoints",
		"body":            `{"name": "shifted", "endpoint_type": "STANDARD"}`,
		"clock_offset_ms": offsetMs,
	})
	require.Equal(t, 200, status)
	before := createdAt("before")
	createEndpoint("after")

	shifted := createdAt("shifted")
	assert.Greater(t, shifted, before+offsetMs)
	assert.Less(t, shifted, createdAt("after")+offsetMs)
}
