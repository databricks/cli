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
			// SDK v0.131.0 deprecated RequestedMinQps/MinQps in favor of TargetQps. Test fake still mirrors the legacy field.
			RequestedMinQps: createReq.MinQps, //nolint:staticcheck
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
	// SDK v0.131.0 deprecated RequestedMinQps/MinQps in favor of TargetQps. Test fake still mirrors the legacy field.
	endpoint.ScalingInfo.RequestedMinQps = patchReq.MinQps //nolint:staticcheck
	endpoint.LastUpdatedTimestamp = nowMilli()
	endpoint.LastUpdatedUser = s.CurrentUser().UserName

	s.VectorSearchEndpoints[endpointName] = endpoint

	return Response{
		Body: endpoint,
	}
}
