package testserver

import (
	"encoding/json"

	"github.com/databricks/databricks-sdk-go/service/pipelines"
)

// pipelineTestingRequest holds the testing field of a Create or Edit request, which is not in the
// Go SDK yet.
type pipelineTestingRequest struct {
	Testing json.RawMessage `json:"testing"`
}

// setPipelineTesting records the testing field of a Create or Edit body, so that Get returns it.
// Like the rest of the spec, an Edit without testing removes it.
func (s *FakeWorkspace) setPipelineTesting(pipelineID string, body []byte) {
	var request pipelineTestingRequest
	_ = json.Unmarshal(body, &request)
	if len(request.Testing) == 0 || string(request.Testing) == "null" {
		delete(s.pipelineTesting, pipelineID)
		return
	}
	s.pipelineTesting[pipelineID] = request.Testing
}

// withPipelineTesting adds spec.testing to a Get Pipeline response, which the Go SDK type cannot
// hold. The JSON is merged as raw values, so numbers and the other fields are kept as they are.
func withPipelineTesting(value pipelines.GetPipelineResponse, testing json.RawMessage) (map[string]json.RawMessage, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	var body map[string]json.RawMessage
	if err := json.Unmarshal(data, &body); err != nil {
		return nil, err
	}
	var spec map[string]json.RawMessage
	if err := json.Unmarshal(body["spec"], &spec); err != nil {
		return nil, err
	}
	spec["testing"] = testing
	body["spec"], err = json.Marshal(spec)
	return body, err
}
