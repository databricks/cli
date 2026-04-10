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
	pathMinQps         = structpath.MustParsePath("min_qps")
)

type ResourceVectorSearchEndpoint struct {
	client *databricks.WorkspaceClient
}

func (*ResourceVectorSearchEndpoint) New(client *databricks.WorkspaceClient) *ResourceVectorSearchEndpoint {
	return &ResourceVectorSearchEndpoint{client: client}
}

func (*ResourceVectorSearchEndpoint) PrepareState(input *resources.VectorSearchEndpoint) *vectorsearch.CreateEndpoint {
	return &input.CreateEndpoint
}

func (*ResourceVectorSearchEndpoint) RemapState(remote *vectorsearch.EndpointInfo) *vectorsearch.CreateEndpoint {
	budgetPolicyId := remote.EffectiveBudgetPolicyId // TODO: use remote.BudgetPolicyId when available
	var minQps int64
	if remote.ScalingInfo != nil {
		minQps = remote.ScalingInfo.RequestedMinQps
	}
	return &vectorsearch.CreateEndpoint{
		Name:            remote.Name,
		EndpointType:    remote.EndpointType,
		BudgetPolicyId:  budgetPolicyId,
		MinQps:          minQps,
		ForceSendFields: utils.FilterFields[vectorsearch.CreateEndpoint](remote.ForceSendFields),
	}
}

func (r *ResourceVectorSearchEndpoint) DoRead(ctx context.Context, id string) (*vectorsearch.EndpointInfo, error) {
	return r.client.VectorSearchEndpoints.GetEndpointByEndpointName(ctx, id)
}

func (r *ResourceVectorSearchEndpoint) DoCreate(ctx context.Context, config *vectorsearch.CreateEndpoint) (string, *vectorsearch.EndpointInfo, error) {
	waiter, err := r.client.VectorSearchEndpoints.CreateEndpoint(ctx, *config)
	if err != nil {
		return "", nil, err
	}
	id := config.Name
	return id, waiter.Response, nil
}

func (r *ResourceVectorSearchEndpoint) WaitAfterCreate(ctx context.Context, config *vectorsearch.CreateEndpoint) (*vectorsearch.EndpointInfo, error) {
	return r.client.VectorSearchEndpoints.WaitGetEndpointVectorSearchEndpointOnline(ctx, config.Name, 60*time.Minute, nil)
}

func (r *ResourceVectorSearchEndpoint) DoUpdate(ctx context.Context, id string, config *vectorsearch.CreateEndpoint, entry *PlanEntry) (*vectorsearch.EndpointInfo, error) {
	if entry.Changes.HasChange(pathBudgetPolicyId) {
		_, err := r.client.VectorSearchEndpoints.UpdateEndpointBudgetPolicy(ctx, vectorsearch.PatchEndpointBudgetPolicyRequest{
			EndpointName:   id,
			BudgetPolicyId: config.BudgetPolicyId,
		})
		if err != nil {
			return nil, err
		}
	}

	if entry.Changes.HasChange(pathMinQps) {
		_, err := r.client.VectorSearchEndpoints.PatchEndpoint(ctx, vectorsearch.PatchEndpointRequest{
			EndpointName:    id,
			MinQps:          config.MinQps,
			ForceSendFields: utils.FilterFields[vectorsearch.PatchEndpointRequest](config.ForceSendFields, "MinQps"),
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
