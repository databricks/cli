package dresources

import (
	"context"
	"time"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/utils"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/vectorsearch"
)

var (
	pathBudgetPolicyId = structpath.MustParsePath("budget_policy_id")
	pathTargetQps      = structpath.MustParsePath("target_qps")
)

// VectorSearchEndpointRemote is remote state for a vector search endpoint. It embeds API response
// fields for drift comparison and adds endpoint_uuid for permissions; deployment state id remains the endpoint name.
type VectorSearchEndpointRemote struct {
	vectorsearch.EndpointInfo
	EndpointUuid string `json:"endpoint_uuid"`
}

func newVectorSearchEndpointRemote(info *vectorsearch.EndpointInfo) *VectorSearchEndpointRemote {
	return &VectorSearchEndpointRemote{
		EndpointInfo: *info,
		EndpointUuid: info.Id,
	}
}

type ResourceVectorSearchEndpoint struct {
	client *databricks.WorkspaceClient
}

func (*ResourceVectorSearchEndpoint) New(client *databricks.WorkspaceClient) *ResourceVectorSearchEndpoint {
	return &ResourceVectorSearchEndpoint{client: client}
}

func (*ResourceVectorSearchEndpoint) PrepareState(input *resources.VectorSearchEndpoint) *vectorsearch.CreateEndpoint {
	return &input.CreateEndpoint
}

func (*ResourceVectorSearchEndpoint) RemapState(remote *VectorSearchEndpointRemote) *vectorsearch.CreateEndpoint {
	var targetQps int64
	if remote.ScalingInfo != nil {
		targetQps = remote.ScalingInfo.RequestedTargetQps
	}
	return &vectorsearch.CreateEndpoint{
		Name:            remote.Name,
		EndpointType:    remote.EndpointType,
		BudgetPolicyId:  remote.BudgetPolicyId,
		UsagePolicyId:   "", // Missing in remote
		TargetQps:       targetQps,
		ForceSendFields: utils.FilterFields[vectorsearch.CreateEndpoint](remote.ForceSendFields, "UsagePolicyId"),
	}
}

func (r *ResourceVectorSearchEndpoint) DoRead(ctx context.Context, id string) (*VectorSearchEndpointRemote, error) {
	info, err := r.client.VectorSearchEndpoints.GetEndpointByEndpointName(ctx, id)
	if err != nil {
		return nil, err
	}
	return newVectorSearchEndpointRemote(info), nil
}

func (r *ResourceVectorSearchEndpoint) DoCreate(ctx context.Context, config *vectorsearch.CreateEndpoint) (string, *VectorSearchEndpointRemote, error) {
	waiter, err := r.client.VectorSearchEndpoints.CreateEndpoint(ctx, *config)
	if err != nil {
		return "", nil, err
	}
	id := config.Name
	return id, newVectorSearchEndpointRemote(waiter.Response), nil
}

func (r *ResourceVectorSearchEndpoint) WaitAfterCreate(ctx context.Context, id string, config *vectorsearch.CreateEndpoint) (*VectorSearchEndpointRemote, error) {
	info, err := r.client.VectorSearchEndpoints.WaitGetEndpointVectorSearchEndpointOnline(ctx, config.Name, 60*time.Minute, nil)
	if err != nil {
		return nil, err
	}
	return newVectorSearchEndpointRemote(info), nil
}

func (r *ResourceVectorSearchEndpoint) DoUpdate(ctx context.Context, id string, config *vectorsearch.CreateEndpoint, entry *PlanEntry) (*VectorSearchEndpointRemote, error) {
	if entry.Changes.HasChange(pathBudgetPolicyId) {
		_, err := r.client.VectorSearchEndpoints.UpdateEndpointBudgetPolicy(ctx, vectorsearch.PatchEndpointBudgetPolicyRequest{
			EndpointName:   id,
			BudgetPolicyId: config.BudgetPolicyId,
		})
		if err != nil {
			return nil, err
		}
	}

	if entry.Changes.HasChange(pathTargetQps) {
		_, err := r.client.VectorSearchEndpoints.PatchEndpoint(ctx, vectorsearch.PatchEndpointRequest{
			EndpointName:    id,
			TargetQps:       config.TargetQps,
			ForceSendFields: nil,
		})
		if err != nil {
			return nil, err
		}
	}

	return nil, nil
}

func (r *ResourceVectorSearchEndpoint) DoDelete(ctx context.Context, id string) error {
	return r.client.VectorSearchEndpoints.DeleteEndpointByEndpointName(ctx, id)
}
