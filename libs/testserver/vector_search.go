package testserver

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/databricks/databricks-sdk-go/service/vectorsearch"
)

func (s *FakeWorkspace) VectorSearchEndpointCreate(req Request) Response {
	defer s.LockUnlock()()

	var createRequest vectorsearch.CreateEndpoint
	if err := json.Unmarshal(req.Body, &createRequest); err != nil {
		return Response{
			Body:       fmt.Sprintf("internal error: %s", err),
			StatusCode: http.StatusInternalServerError,
		}
	}

	info := vectorsearch.EndpointInfo{
		Name:         createRequest.Name,
		EndpointType: createRequest.EndpointType,
		Creator:      s.CurrentUser().UserName,
		Id:           fmt.Sprintf("vs-ep-%d", nextID()),
		EndpointStatus: &vectorsearch.EndpointStatus{
			State: "ONLINE",
		},
	}
	s.VectorSearchEndpoints[createRequest.Name] = info
	return Response{Body: info}
}

func (s *FakeWorkspace) VectorSearchEndpointPatch(req Request) Response {
	defer s.LockUnlock()()

	endpointName := req.Vars["endpoint_name"]
	info, ok := s.VectorSearchEndpoints[endpointName]
	if !ok {
		return Response{
			StatusCode: http.StatusNotFound,
			Body: map[string]string{
				"error_code": "NOT_FOUND",
				"message":    "resource not found",
			},
		}
	}

	var patchRequest vectorsearch.PatchEndpointRequest
	if err := json.Unmarshal(req.Body, &patchRequest); err != nil {
		return Response{
			Body:       fmt.Sprintf("internal error: %s", err),
			StatusCode: http.StatusInternalServerError,
		}
	}

	if info.ScalingInfo == nil {
		info.ScalingInfo = &vectorsearch.EndpointScalingInfo{}
	}
	info.ScalingInfo.RequestedMinQps = patchRequest.MinQps
	s.VectorSearchEndpoints[endpointName] = info
	return Response{Body: info}
}

func (s *FakeWorkspace) VectorSearchIndexCreate(req Request) Response {
	defer s.LockUnlock()()

	var createRequest vectorsearch.CreateVectorIndexRequest
	if err := json.Unmarshal(req.Body, &createRequest); err != nil {
		return Response{
			Body:       fmt.Sprintf("internal error: %s", err),
			StatusCode: http.StatusInternalServerError,
		}
	}

	info := vectorsearch.VectorIndex{
		Name:         createRequest.Name,
		EndpointName: createRequest.EndpointName,
		PrimaryKey:   createRequest.PrimaryKey,
		IndexType:    createRequest.IndexType,
	}
	s.VectorSearchIndexes[createRequest.Name] = info
	return Response{Body: info}
}
