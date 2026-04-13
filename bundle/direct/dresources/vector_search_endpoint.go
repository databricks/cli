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

// VectorSearchRefreshOutput is remote state for a vector search endpoint. It embeds API response
// fields for drift comparison and adds endpoint_uuid for permissions; deployment state id remains the endpoint name.
type VectorSearchRefreshOutput struct {
	*vectorsearch.EndpointInfo
	EndpointUuid string `json:"endpoint_uuid"`
}

func newVectorSearchRefreshOutput(info *vectorsearch.EndpointInfo) *VectorSearchRefreshOutput {
	if info == nil {
		return nil
	}
	return &VectorSearchRefreshOutput{
		EndpointInfo: info,
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

func (*ResourceVectorSearchEndpoint) RemapState(remote *VectorSearchRefreshOutput) *vectorsearch.CreateEndpoint {
	if remote == nil || remote.EndpointInfo == nil {
		return &vectorsearch.CreateEndpoint{
			BudgetPolicyId:  "",
			EndpointType:    "",
			MinQps:          0,
			Name:            "",
			ForceSendFields: nil,
		}
	}
	info := remote.EndpointInfo
	budgetPolicyId := info.EffectiveBudgetPolicyId // TODO: use info.BudgetPolicyId when available
	var minQps int64
	if info.ScalingInfo != nil {
		minQps = info.ScalingInfo.RequestedMinQps
	}
	return &vectorsearch.CreateEndpoint{
		Name:            info.Name,
		EndpointType:    info.EndpointType,
		BudgetPolicyId:  budgetPolicyId,
		MinQps:          minQps,
		ForceSendFields: utils.FilterFields[vectorsearch.CreateEndpoint](info.ForceSendFields),
	}
}

func (r *ResourceVectorSearchEndpoint) DoRead(ctx context.Context, id string) (*VectorSearchRefreshOutput, error) {
	info, err := r.client.VectorSearchEndpoints.GetEndpointByEndpointName(ctx, id)
	if err != nil {
		return nil, err
	}
	return newVectorSearchRefreshOutput(info), nil
}

func (r *ResourceVectorSearchEndpoint) DoCreate(ctx context.Context, config *vectorsearch.CreateEndpoint) (string, *VectorSearchRefreshOutput, error) {
	waiter, err := r.client.VectorSearchEndpoints.CreateEndpoint(ctx, *config)
	if err != nil {
		return "", nil, err
	}
	id := config.Name
	return id, newVectorSearchRefreshOutput(waiter.Response), nil
}

func (r *ResourceVectorSearchEndpoint) WaitAfterCreate(ctx context.Context, config *vectorsearch.CreateEndpoint) (*VectorSearchRefreshOutput, error) {
	info, err := r.client.VectorSearchEndpoints.WaitGetEndpointVectorSearchEndpointOnline(ctx, config.Name, 60*time.Minute, nil)
	if err != nil {
		return nil, err
	}
	return newVectorSearchRefreshOutput(info), nil
}

func (r *ResourceVectorSearchEndpoint) DoUpdate(ctx context.Context, id string, config *vectorsearch.CreateEndpoint, entry *PlanEntry) (*VectorSearchRefreshOutput, error) {
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
