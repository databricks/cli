package testserver

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/databricks/databricks-sdk-go/service/vectorsearch"
)

func (s *FakeWorkspace) VectorSearchEndpointCreate(req Request) Response {
	defer s.LockUnlock()()

	var createReq vectorsearch.CreateEndpoint
	if err := json.Unmarshal(req.Body, &createReq); err != nil {
		return Response{
			Body:       fmt.Sprintf("cannot unmarshal request body: %s", err),
			StatusCode: http.StatusBadRequest,
		}
	}

	if _, exists := s.VectorSearchEndpoints[createReq.Name]; exists {
		return Response{
			StatusCode: http.StatusConflict,
			Body:       map[string]string{"error_code": "RESOURCE_ALREADY_EXISTS", "message": fmt.Sprintf("Vector search endpoint with name %s already exists", createReq.Name)},
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

	s.VectorSearchEndpoints[createReq.Name] = endpoint

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

	endpoint, exists := s.VectorSearchEndpoints[endpointName]
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

	s.VectorSearchEndpoints[endpointName] = endpoint

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

	endpoint, exists := s.VectorSearchEndpoints[endpointName]
	if !exists {
		return Response{
			StatusCode: http.StatusNotFound,
			Body:       map[string]string{"error_code": "RESOURCE_DOES_NOT_EXIST", "message": fmt.Sprintf("Vector search endpoint %s not found", endpointName)},
		}
	}

	if endpoint.ScalingInfo == nil {
		endpoint.ScalingInfo = &vectorsearch.EndpointScalingInfo{}
	}
	// target_qps is omitempty, so a config that clears it drops it from the body and the backend
	// keeps the value it had (aws, 2026-09: the clear is accepted and has no effect). Assigning the
	// zero value here instead made the field look freely clearable.
	if patchReq.TargetQps != 0 {
		endpoint.ScalingInfo.RequestedTargetQps = patchReq.TargetQps
	}
	endpoint.LastUpdatedTimestamp = nowMilli()
	endpoint.LastUpdatedUser = s.CurrentUser().UserName

	s.VectorSearchEndpoints[endpointName] = endpoint

	return Response{
		Body: endpoint,
	}
}
