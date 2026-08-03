package testserver_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/databricks/cli/libs/testserver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFaultRulesNoMatch(t *testing.T) {
	fr := testserver.NewFaultRules()
	fr.Set("tok", "GET /foo", 504, "body", 0, 1, "")

	assert.Nil(t, fr.Check("POST", "/foo", "tok", nil))
	assert.Nil(t, fr.Check("GET", "/bar", "tok", nil))
	assert.Nil(t, fr.Check("GET", "/foo", "other", nil))
}

func TestFaultRulesExactMatch(t *testing.T) {
	fr := testserver.NewFaultRules()
	fr.Set("tok", "PUT /api/2.0/jobs/123", 504, "body", 0, 1, "")

	rule := fr.Check("PUT", "/api/2.0/jobs/123", "tok", nil)
	require.NotNil(t, rule)
	assert.Equal(t, 504, rule.StatusCode)
	assert.Equal(t, "body", rule.Body)
}

func TestFaultRulesWildcardMatch(t *testing.T) {
	fr := testserver.NewFaultRules()
	fr.Set("tok", "PUT /api/2.0/permissions/pipelines/*", 504, "body", 0, 2, "")

	assert.NotNil(t, fr.Check("PUT", "/api/2.0/permissions/pipelines/abc", "tok", nil))
	assert.NotNil(t, fr.Check("PUT", "/api/2.0/permissions/pipelines/xyz", "tok", nil))
	assert.Nil(t, fr.Check("PUT", "/api/2.0/permissions/pipelines/xyz", "tok", nil)) // exhausted
}

func TestFaultRulesOffset(t *testing.T) {
	fr := testserver.NewFaultRules()
	fr.Set("tok", "GET /foo", 504, "body", 2, 1, "")

	assert.Nil(t, fr.Check("GET", "/foo", "tok", nil))    // offset 2→1
	assert.Nil(t, fr.Check("GET", "/foo", "tok", nil))    // offset 1→0
	assert.NotNil(t, fr.Check("GET", "/foo", "tok", nil)) // fires
	assert.Nil(t, fr.Check("GET", "/foo", "tok", nil))    // exhausted
}

func TestFaultRulesTimes(t *testing.T) {
	fr := testserver.NewFaultRules()
	fr.Set("tok", "GET /foo", 504, "body", 0, 3, "")

	for range 3 {
		assert.NotNil(t, fr.Check("GET", "/foo", "tok", nil))
	}
	assert.Nil(t, fr.Check("GET", "/foo", "tok", nil)) // exhausted
}

func TestFaultRulesBodyContains(t *testing.T) {
	fr := testserver.NewFaultRules()
	fr.Set("tok", "POST /api/2.0/workspace/import", 403, "body", 0, 1, "state/resources.json")

	// Body without the substring must not match and must not consume the budget.
	assert.Nil(t, fr.Check("POST", "/api/2.0/workspace/import", "tok", []byte("path=state/deploy.lock")))
	// Body with the substring fires.
	assert.NotNil(t, fr.Check("POST", "/api/2.0/workspace/import", "tok", []byte("path=state/resources.json")))
	// Budget was only consumed by the matching request.
	assert.Nil(t, fr.Check("POST", "/api/2.0/workspace/import", "tok", []byte("path=state/resources.json")))
}

func TestServerFaultAfterHandlerKeepsTheHandlersEffect(t *testing.T) {
	var calls atomic.Int32
	server := testserver.New(t)
	server.Handle("POST", "/create", func(req testserver.Request) any {
		calls.Add(1)
		return map[string]string{"status": "created"}
	})

	body, err := json.Marshal(map[string]any{
		"pattern":       "POST /create",
		"status_code":   503,
		"body":          `{"error_code": "INJECTED", "message": "Fault injected by test."}`,
		"offset":        0,
		"times":         1,
		"after_handler": true,
	})
	require.NoError(t, err)

	setReq, err := http.NewRequest(http.MethodPost, server.URL+"/__testserver/fault", bytes.NewReader(body))
	require.NoError(t, err)
	setReq.Header.Set("Authorization", "Bearer dbapi-fault-test")
	setResp, err := http.DefaultClient.Do(setReq)
	require.NoError(t, err)
	require.Equal(t, 200, setResp.StatusCode)
	setResp.Body.Close()

	createReq, err := http.NewRequest(http.MethodPost, server.URL+"/create", bytes.NewReader([]byte("{}")))
	require.NoError(t, err)
	createReq.Header.Set("Authorization", "Bearer dbapi-fault-test")
	createResp, err := http.DefaultClient.Do(createReq)
	require.NoError(t, err)
	assert.Equal(t, 503, createResp.StatusCode)
	createResp.Body.Close()

	assert.Equal(t, int32(1), calls.Load())
}
