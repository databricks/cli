package fuzz

import (
	"encoding/json"
	"sync"

	"github.com/databricks/cli/libs/testserver"
)

// jobsCreatePath is the Jobs API route the deploy must hit on create. The
// testserver registers only this version, so posting to a different one surfaces
// as a capture failure ("did not POST").
const jobsCreatePath = "/api/2.2/jobs/create"

// capturedRequest is a single mutating API request observed by the testserver.
type capturedRequest struct {
	Method string
	Path   string
	Body   json.RawMessage
}

// recorder collects request bodies sent to a testserver. It is safe for
// concurrent use because the deploy may issue requests from multiple goroutines.
type recorder struct {
	mu       sync.Mutex
	requests []capturedRequest
}

func (r *recorder) callback(req *testserver.Request) {
	r.mu.Lock()
	defer r.mu.Unlock()

	var body json.RawMessage
	if json.Valid(req.Body) {
		// Copy: testserver reuses the underlying buffer across requests.
		body = append(json.RawMessage(nil), req.Body...)
	}

	r.requests = append(r.requests, capturedRequest{
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
