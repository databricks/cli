package testserver_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/databricks/cli/libs/testserver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFaultRulesNoMatch(t *testing.T) {
	fr := testserver.NewFaultRules()
	fr.Set("tok", "GET /foo", 504, "body", 0, 1, false)

	assert.Nil(t, fr.Check("POST", "/foo", "tok"))
	assert.Nil(t, fr.Check("GET", "/bar", "tok"))
	assert.Nil(t, fr.Check("GET", "/foo", "other"))
}

func TestFaultRulesExactMatch(t *testing.T) {
	fr := testserver.NewFaultRules()
	fr.Set("tok", "PUT /api/2.0/jobs/123", 504, "body", 0, 1, false)

	rule := fr.Check("PUT", "/api/2.0/jobs/123", "tok")
	require.NotNil(t, rule)
	assert.Equal(t, 504, rule.StatusCode)
	assert.Equal(t, "body", rule.Body)
}

func TestFaultRulesWildcardMatch(t *testing.T) {
	fr := testserver.NewFaultRules()
	fr.Set("tok", "PUT /api/2.0/permissions/pipelines/*", 504, "body", 0, 2, false)

	assert.NotNil(t, fr.Check("PUT", "/api/2.0/permissions/pipelines/abc", "tok"))
	assert.NotNil(t, fr.Check("PUT", "/api/2.0/permissions/pipelines/xyz", "tok"))
	assert.Nil(t, fr.Check("PUT", "/api/2.0/permissions/pipelines/xyz", "tok")) // exhausted
}

func TestFaultRulesOffset(t *testing.T) {
	fr := testserver.NewFaultRules()
	fr.Set("tok", "GET /foo", 504, "body", 2, 1, false)

	assert.Nil(t, fr.Check("GET", "/foo", "tok"))    // offset 2→1
	assert.Nil(t, fr.Check("GET", "/foo", "tok"))    // offset 1→0
	assert.NotNil(t, fr.Check("GET", "/foo", "tok")) // fires
	assert.Nil(t, fr.Check("GET", "/foo", "tok"))    // exhausted
}

func TestFaultRulesTimes(t *testing.T) {
	fr := testserver.NewFaultRules()
	fr.Set("tok", "GET /foo", 504, "body", 0, 3, false)

	for range 3 {
		assert.NotNil(t, fr.Check("GET", "/foo", "tok"))
	}
	assert.Nil(t, fr.Check("GET", "/foo", "tok")) // exhausted
}

// A plain fault answers instead of the handler, so the request has no effect. An
// after-handler one keeps the effect and replaces only the response.
func TestServerFaultAfterHandlerKeepsTheHandlersEffect(t *testing.T) {
	for _, afterHandler := range []bool{false, true} {
		t.Run(fmt.Sprintf("after_handler=%v", afterHandler), func(t *testing.T) {
			var calls atomic.Int32
			server := testserver.New(t)
			server.Handle("POST", "/create", func(req testserver.Request) any {
				calls.Add(1)
				return map[string]string{"status": "created"}
			})
			setFault(t, server, "POST /create", afterHandler)

			assert.Equal(t, 503, post(t, server, "/create"))
			if afterHandler {
				assert.Equal(t, int32(1), calls.Load())
			} else {
				assert.Zero(t, calls.Load())
			}
		})
	}
}

func TestServerFaultAppliesOnlyForTheGivenTimes(t *testing.T) {
	var calls atomic.Int32
	server := testserver.New(t)
	server.Handle("POST", "/create", func(req testserver.Request) any {
		calls.Add(1)
		return map[string]string{"status": "created"}
	})
	setFault(t, server, "POST /create", true)

	require.Equal(t, 503, post(t, server, "/create"))
	// The rule fired once, so the resend is answered by the handler as usual.
	assert.Equal(t, 200, post(t, server, "/create"))
	assert.Equal(t, int32(2), calls.Load())
}

const faultTestToken = "dbapi-fault-test"

// setFault registers a one-shot 503 through the endpoint fault.py posts to.
func setFault(t *testing.T, server *testserver.Server, pattern string, afterHandler bool) {
	t.Helper()
	body, err := json.Marshal(map[string]any{
		"pattern":       pattern,
		"status_code":   503,
		"body":          `{"error_code": "INJECTED", "message": "Fault injected by test."}`,
		"offset":        0,
		"times":         1,
		"after_handler": afterHandler,
	})
	require.NoError(t, err)
	require.Equal(t, 200, postBody(t, server, "/__testserver/fault", body))
}

// post sends an empty request to path and reports the status code.
func post(t *testing.T, server *testserver.Server, path string) int {
	t.Helper()
	return postBody(t, server, path, []byte("{}"))
}

func postBody(t *testing.T, server *testserver.Server, path string, body []byte) int {
	t.Helper()
	request, err := http.NewRequest(http.MethodPost, server.URL+path, bytes.NewReader(body))
	require.NoError(t, err)
	request.Header.Set("Authorization", "Bearer "+faultTestToken)

	response, err := http.DefaultClient.Do(request)
	require.NoError(t, err)
	defer response.Body.Close()
	return response.StatusCode
}
