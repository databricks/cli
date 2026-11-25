package testserver

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/databricks/databricks-sdk-go/service/vectorsearch"
)

func init() {
	registerRoutes("vector_search_endpoints", addVectorSearchEndpointsRoutes)
	registerStore("vector_search_endpoints", newVectorSearchEndpointsStore)
}

type vectorSearchEndpointsStore struct {
	VectorSearchEndpoints map[string]vectorsearch.EndpointInfo
}

func newVectorSearchEndpointsStore(*FakeWorkspace) *vectorSearchEndpointsStore {
	return &vectorSearchEndpointsStore{
		VectorSearchEndpoints: map[string]vectorsearch.EndpointInfo{},
	}
}

func (s *FakeWorkspace) vectorSearchEndpoints() *vectorSearchEndpointsStore {
	return stateOf[vectorSearchEndpointsStore](s)
}

func addVectorSearchEndpointsRoutes(server *Server) {
	// Vector Search Endpoints:

	server.Handle("POST", "/api/2.0/vector-search/endpoints", func(req Request) any {
		return req.Workspace.VectorSearchEndpointCreate(req)
	})

	server.Handle("GET", "/api/2.0/vector-search/endpoints", func(req Request) any {
		return MapList(req.Workspace, req.Workspace.vectorSearchEndpoints().VectorSearchEndpoints, "endpoints")
	})

	server.Handle("GET", "/api/2.0/vector-search/endpoints/{endpoint_name}", func(req Request) any {
		return MapGet(req.Workspace, req.Workspace.vectorSearchEndpoints().VectorSearchEndpoints, req.Vars["endpoint_name"])
	})

	server.Handle("PATCH", "/api/2.0/vector-search/endpoints/{endpoint_name}", func(req Request) any {
		return req.Workspace.VectorSearchEndpointUpdate(req, req.Vars["endpoint_name"])
	})

	server.Handle("DELETE", "/api/2.0/vector-search/endpoints/{endpoint_name}", func(req Request) any {
		return MapDelete(req.Workspace, req.Workspace.vectorSearchEndpoints().VectorSearchEndpoints, req.Vars["endpoint_name"])
	})

	server.Handle("PATCH", "/api/2.0/vector-search/endpoints/{endpoint_name}/budget-policy", func(req Request) any {
		return req.Workspace.VectorSearchEndpointUpdateBudgetPolicy(req, req.Vars["endpoint_name"])
	})
}

func (s *FakeWorkspace) VectorSearchEndpointCreate(req Request) Response {
	defer s.LockUnlock()()

	var createReq vectorsearch.CreateEndpoint
	if err := json.Unmarshal(req.Body, &createReq); err != nil {
		return Response{
			Body:       fmt.Sprintf("cannot unmarshal request body: %s", err),
			StatusCode: http.StatusBadRequest,
		}
	}

	if _, exists := s.vectorSearchEndpoints().VectorSearchEndpoints[createReq.Name]; exists {
		return Response{
			StatusCode: http.StatusConflict,
			Body:       map[string]string{"error_code": "ALREADY_EXISTS", "message": fmt.Sprintf("AI Search endpoint %s already exists. Please try again with a different endpoint name.", createReq.Name)},
		}
	}

	endpoint := vectorsearch.EndpointInfo{
		BudgetPolicyId:          createReq.BudgetPolicyId,
		EffectiveBudgetPolicyId: createReq.BudgetPolicyId,
		Creator:                 s.CurrentUser().UserName,
		CreationTimestamp:       nowMilli(),
		EndpointType:            createReq.EndpointType,
		Id:                      nextUUID(),
		LastUpdatedUser:         s.CurrentUser().UserName,
		Name:                    createReq.Name,
		EndpointStatus: &vectorsearch.EndpointStatus{
			State: vectorsearch.EndpointStatusStateOnline, // initial create is no-op, returns ONLINE immediately
		},
		ScalingInfo: &vectorsearch.EndpointScalingInfo{
			RequestedTargetQps: createReq.TargetQps,
		},
	}
	endpoint.LastUpdatedTimestamp = endpoint.CreationTimestamp

	s.vectorSearchEndpoints().VectorSearchEndpoints[createReq.Name] = endpoint

	return Response{
		Body: endpoint,
	}
}

func (s *FakeWorkspace) VectorSearchEndpointUpdateBudgetPolicy(req Request, endpointName string) Response {
	defer s.LockUnlock()()

	var patchReq vectorsearch.PatchEndpointBudgetPolicyRequest
	if err := json.Unmarshal(req.Body, &patchReq); err != nil {
		return Response{
			Body:       fmt.Sprintf("cannot unmarshal request body: %s", err),
			StatusCode: http.StatusBadRequest,
		}
	}

	endpoint, exists := s.vectorSearchEndpoints().VectorSearchEndpoints[endpointName]
	if !exists {
		return Response{
			StatusCode: http.StatusNotFound,
			Body:       map[string]string{"error_code": "RESOURCE_DOES_NOT_EXIST", "message": fmt.Sprintf("Vector search endpoint %s not found", endpointName)},
		}
	}

	endpoint.BudgetPolicyId = patchReq.BudgetPolicyId
	endpoint.EffectiveBudgetPolicyId = patchReq.BudgetPolicyId // assume it always becomes the effective policy
	endpoint.LastUpdatedTimestamp = nowMilli()
	endpoint.LastUpdatedUser = s.CurrentUser().UserName

	s.vectorSearchEndpoints().VectorSearchEndpoints[endpointName] = endpoint

	return Response{
		Body: vectorsearch.PatchEndpointBudgetPolicyResponse{
			EffectiveBudgetPolicyId: endpoint.EffectiveBudgetPolicyId,
		},
	}
}

func (s *FakeWorkspace) VectorSearchEndpointUpdate(req Request, endpointName string) Response {
	defer s.LockUnlock()()

	var patchReq vectorsearch.PatchEndpointRequest
	if err := json.Unmarshal(req.Body, &patchReq); err != nil {
		return Response{
			Body:       fmt.Sprintf("cannot unmarshal request body: %s", err),
			StatusCode: http.StatusBadRequest,
		}
	}

	endpoint, exists := s.vectorSearchEndpoints().VectorSearchEndpoints[endpointName]
	if !exists {
		return Response{
			StatusCode: http.StatusNotFound,
			Body:       map[string]string{"error_code": "RESOURCE_DOES_NOT_EXIST", "message": fmt.Sprintf("Vector search endpoint %s not found", endpointName)},
		}
	}

	if endpoint.ScalingInfo == nil {
		endpoint.ScalingInfo = &vectorsearch.EndpointScalingInfo{}
	}
	endpoint.ScalingInfo.RequestedTargetQps = patchReq.TargetQps
	endpoint.LastUpdatedTimestamp = nowMilli()
	endpoint.LastUpdatedUser = s.CurrentUser().UserName

	s.vectorSearchEndpoints().VectorSearchEndpoints[endpointName] = endpoint

	return Response{
		Body: endpoint,
	}
}
