package qa

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/databricks/databricks-sdk-go/databricks"
	"github.com/databricks/databricks-sdk-go/workspaces"
	"github.com/stretchr/testify/assert"
)

// HTTPFixture defines request structure for test
type HTTPFixture struct {
	Method       string
	RequestURI   string
	ReuseRequest bool
	Response     any
}

func getFixtureServer(t *testing.T, fixtures []HTTPFixture) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		found := false
		for i, fixture := range fixtures {
			if req.Method == fixture.Method && req.RequestURI == fixture.RequestURI {
				found = true
				// Reset the request if it is already used
				if !fixture.ReuseRequest {
					fixtures[i] = HTTPFixture{}
				}
				if fixture.Response != nil {
					if alreadyJSON, ok := fixture.Response.(string); ok {
						_, err := rw.Write([]byte(alreadyJSON))
						assert.NoError(t, err, err)
					} else {
						responseBytes, err := json.Marshal(fixture.Response)
						if err != nil {
							assert.NoError(t, err, err)
							t.FailNow()
						}
						_, err = rw.Write(responseBytes)
						assert.NoError(t, err, err)
					}
				}
				break
			}
		}
		if !found {
			stub := fmt.Sprintf(`{
				Method:   "%s",
				RequestURI: "%s",
				Response: XXX {
					// fill in specific fields...
				},
			},`, req.Method, req.RequestURI)
			assert.Fail(t, fmt.Sprintf("Missing stub, please add: %s", stub))
			t.FailNow()
		}
	}))
}

// GeFixtureDatabricksClient creates client for emulated HTTP server
func GeFixtureWorkspacesClient(t *testing.T, fixtures []HTTPFixture) (*workspaces.WorkspacesClient, *httptest.Server) {
	server := getFixtureServer(t, fixtures)
	return workspaces.New(&databricks.Config{Host: server.URL, Token: "..."}), server
}
