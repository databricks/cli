// Package httpfixtures serves canned HTTP responses to unit tests that call
// the Databricks API, replacing the previous SDK's qa.HTTPFixtures.
package httpfixtures

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"

	"github.com/databricks/databricks-sdk-go/config"
	"github.com/stretchr/testify/assert"
)

// Fixture is one expected request and the response to send for it.
type Fixture struct {
	Method string
	// Resource is the request path, optionally with a query string. Query
	// parameters are compared regardless of their order.
	Resource string
	// Response is written as is if it is a string and JSON-encoded otherwise.
	Response any
	// Status defaults to 200.
	Status int
	// ExpectedRequest, if set, is compared as JSON against the request body.
	ExpectedRequest any
	// ReuseRequest lets the fixture match more than one request.
	ReuseRequest bool
}

type Fixtures []Fixture

// hostMetadata answers config resolution's host metadata lookup so tests do
// not depend on it unless they stub it themselves.
var hostMetadata = Fixture{
	Method:       "GET",
	Resource:     "/.well-known/databricks-config",
	ReuseRequest: true,
	Status:       404,
	Response:     `{"error_code":"NOT_FOUND","message":"host metadata endpoint auto-stubbed by httpfixtures"}`,
}

func (f Fixture) matches(req *http.Request) bool {
	if f.Method != req.Method {
		return false
	}
	want, err := url.Parse(f.Resource)
	if err != nil || want.Path != req.URL.Path {
		return false
	}
	got := req.URL.Query()
	wantQuery := want.Query()
	if len(got) == 0 && len(wantQuery) == 0 {
		return true
	}
	return reflect.DeepEqual(got, wantQuery)
}

// Config starts a server for the fixtures and returns a PAT config for it.
// Requests that match no fixture fail the test.
func (fixtures Fixtures) Config(t *testing.T) (*config.Config, *httptest.Server) {
	fixtures = append(Fixtures{hostMetadata}, fixtures...)
	used := make([]bool, len(fixtures))
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		for i, fixture := range fixtures {
			if used[i] || !fixture.matches(req) {
				continue
			}
			if !fixture.ReuseRequest {
				used[i] = true
			}
			if fixture.ExpectedRequest != nil {
				body, err := io.ReadAll(req.Body)
				assert.NoError(t, err)
				expected, err := json.Marshal(fixture.ExpectedRequest)
				assert.NoError(t, err)
				assert.JSONEq(t, string(expected), string(body), "request body for %s %s", req.Method, req.URL)
			}
			status := fixture.Status
			if status == 0 {
				status = http.StatusOK
			}
			rw.WriteHeader(status)
			switch r := fixture.Response.(type) {
			case nil:
			case string:
				_, _ = rw.Write([]byte(r))
			default:
				data, err := json.Marshal(r)
				assert.NoError(t, err)
				_, _ = rw.Write(data)
			}
			return
		}
		assert.Fail(t, "missing fixture for "+req.Method+" "+req.URL.String())
		rw.WriteHeader(http.StatusNotImplemented)
	}))
	t.Cleanup(server.Close)
	return &config.Config{
		Credentials: config.PatCredentials{},
		Host:        server.URL,
		Token:       "x",
	}, server
}
