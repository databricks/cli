package dresources

import (
	"context"
	"fmt"
	"time"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/direct/internal/retries"
	"github.com/databricks/cli/libs/sdkclients"
	vectorsearch "github.com/databricks/cli/libs/sdktypes/vectorsearch/v1"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/utils"
	sdkvectorsearch "github.com/databricks/sdk-go/vectorsearch/v1"
)

var (
	pathBudgetPolicyId = structpath.MustParsePath("budget_policy_id")
	pathTargetQps      = structpath.MustParsePath("target_qps")
)

// VectorSearchEndpointRemote is remote state for a vector search endpoint. It embeds API response
// fields for drift comparison and adds endpoint_uuid for permissions; deployment state id remains the endpoint name.
type VectorSearchEndpointRemote struct {
	vectorsearch.Endpoint
	EndpointUuid string `json:"endpoint_uuid"`
	// TargetQps is mapped from Endpoint.ScalingInfo.RequestedTargetQps in DoRead
	// so that drift detection can compare it directly against the config field.
	TargetQps *int64 `json:"target_qps,omitempty"`
}

func newVectorSearchEndpointRemote(endpoint *sdkvectorsearch.Endpoint) (*VectorSearchEndpointRemote, error) {
	info, err := vectorsearch.EndpointFromSDK(endpoint)
	if err != nil {
		return nil, err
	}
	var targetQps *int64
	if info.ScalingInfo != nil {
		targetQps = info.ScalingInfo.RequestedTargetQps
	}
	return &VectorSearchEndpointRemote{
		Endpoint:     *info,
		EndpointUuid: utils.Deref(info.Id),
		TargetQps:    targetQps,
	}, nil
}

type ResourceVectorSearchEndpoint struct {
	client *sdkclients.Clients
}

func (*ResourceVectorSearchEndpoint) New(client *sdkclients.Clients) *ResourceVectorSearchEndpoint {
	return &ResourceVectorSearchEndpoint{client: client}
}

func (*ResourceVectorSearchEndpoint) PrepareState(input *resources.VectorSearchEndpoint) *vectorsearch.CreateEndpointRequest {
	return &input.CreateEndpointRequest
}

func (*ResourceVectorSearchEndpoint) RemapState(remote *VectorSearchEndpointRemote) *vectorsearch.CreateEndpointRequest {
	return &vectorsearch.CreateEndpointRequest{
		Name:           remote.Name,
		EndpointType:   remote.EndpointType,
		BudgetPolicyId: remote.BudgetPolicyId,
		UsagePolicyId:  nil, // Missing in remote
		TargetQps:      remote.TargetQps,
	}
}

func (r *ResourceVectorSearchEndpoint) getEndpoint(ctx context.Context, name string) (*sdkvectorsearch.Endpoint, error) {
	return r.client.VectorSearch.GetEndpoint(ctx, sdkvectorsearch.GetEndpointRequest{Name: new(name)})
}

func (r *ResourceVectorSearchEndpoint) DoRead(ctx context.Context, id string) (*VectorSearchEndpointRemote, error) {
	endpoint, err := r.getEndpoint(ctx, id)
	if err != nil {
		return nil, err
	}
	return newVectorSearchEndpointRemote(endpoint)
}

func (r *ResourceVectorSearchEndpoint) DoCreate(ctx context.Context, config *vectorsearch.CreateEndpointRequest) (string, *VectorSearchEndpointRemote, error) {
	request, err := vectorsearch.CreateEndpointRequestToSDK(config)
	if err != nil {
		return "", nil, err
	}
	if _, err := r.client.VectorSearch.CreateEndpoint(ctx, *request); err != nil {
		return "", nil, err
	}
	return utils.Deref(config.Name), nil, nil
}

func (r *ResourceVectorSearchEndpoint) WaitAfterCreate(ctx context.Context, id string, config *vectorsearch.CreateEndpointRequest) (*VectorSearchEndpointRemote, error) {
	name := utils.Deref(config.Name)
	endpoint, err := retries.Poll(ctx, 60*time.Minute, func() (*sdkvectorsearch.Endpoint, *retries.Err) {
		endpoint, err := r.getEndpoint(ctx, name)
		if err != nil {
			return nil, retries.Halt(err)
		}
		var status sdkvectorsearch.EndpointStatus_State
		statusMessage := ""
		if endpoint.EndpointStatus != nil {
			status = endpoint.EndpointStatus.State
			statusMessage = utils.Deref(endpoint.EndpointStatus.Message)
		}
		switch status {
		case sdkvectorsearch.EndpointStatus_State_Online:
			return endpoint, nil
		case sdkvectorsearch.EndpointStatus_State_Offline:
			return nil, retries.Halt(fmt.Errorf("failed to reach %s, got %s: %s", sdkvectorsearch.EndpointStatus_State_Online, status, statusMessage))
		default:
			return nil, retries.Continues(statusMessage)
		}
	})
	if err != nil {
		return nil, err
	}
	return newVectorSearchEndpointRemote(endpoint)
}

func (r *ResourceVectorSearchEndpoint) DoUpdate(ctx context.Context, id string, config *vectorsearch.CreateEndpointRequest, entry *PlanEntry) (*VectorSearchEndpointRemote, error) {
	if entry.Changes.HasChange(pathBudgetPolicyId) {
		_, err := r.client.VectorSearch.PatchEndpointBudgetPolicy(ctx, sdkvectorsearch.PatchEndpointBudgetPolicyRequest{
			Name:           new(id),
			BudgetPolicyId: orZero(config.BudgetPolicyId),
		})
		if err != nil {
			return nil, err
		}
	}

	if entry.Changes.HasChange(pathTargetQps) {
		_, err := r.client.VectorSearch.PatchEndpoint(ctx, sdkvectorsearch.PatchEndpointRequest{
			Name:      new(id),
			TargetQps: config.TargetQps,
		})
		if err != nil {
			return nil, err
		}
	}

	return nil, nil
}

func (r *ResourceVectorSearchEndpoint) DoDelete(ctx context.Context, id string, _ *vectorsearch.CreateEndpointRequest) error {
	_, err := r.client.VectorSearch.DeleteEndpoint(ctx, sdkvectorsearch.DeleteEndpointRequest{Name: new(id)})
	return err
}
