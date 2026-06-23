package fuzz

import (
	"encoding/json"
	"sync"

	"github.com/databricks/cli/libs/testserver"
)

// jobsCreatePath is the Jobs API route both engines must hit on create. The
// direct engine posts here via the SDK; the terraform provider is expected to
// post here too, and a mismatch (e.g. a different API version) is itself a
// divergence worth surfacing.
const jobsCreatePath = "/api/2.2/jobs/create"

// CapturedRequest is a single mutating API request observed by the testserver.
type CapturedRequest struct {
	Method string
	Path   string
	Body   json.RawMessage
}

// recorder collects request bodies sent to a testserver. It is safe for
// concurrent use because the SDK and terraform may issue requests from multiple
// goroutines.
type recorder struct {
	mu       sync.Mutex
	requests []CapturedRequest
}

func (r *recorder) callback(req *testserver.Request) {
	r.mu.Lock()
	defer r.mu.Unlock()

	var body json.RawMessage
	if json.Valid(req.Body) {
		// Copy: testserver reuses the underlying buffer across requests.
		body = append(json.RawMessage(nil), req.Body...)
	}

	r.requests = append(r.requests, CapturedRequest{
		Method: req.Method,
		Path:   req.URL.Path,
		Body:   body,
	})
}

// find returns the body of the first recorded request matching method and path.
func (r *recorder) find(method, path string) (json.RawMessage, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, req := range r.requests {
		if req.Method == method && req.Path == path {
			return req.Body, true
		}
	}
	return nil, false
}
