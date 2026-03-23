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

var pathMinQps = structpath.MustParsePath("min_qps")

type ResourceVectorSearchEndpoint struct {
	client *databricks.WorkspaceClient
}

func (*ResourceVectorSearchEndpoint) New(client *databricks.WorkspaceClient) *ResourceVectorSearchEndpoint {
	return &ResourceVectorSearchEndpoint{client: client}
}

func (*ResourceVectorSearchEndpoint) PrepareState(input *resources.VectorSearchEndpoint) *vectorsearch.CreateEndpoint {
	return &input.CreateEndpoint
}

func (*ResourceVectorSearchEndpoint) RemapState(info *vectorsearch.EndpointInfo) *vectorsearch.CreateEndpoint {
	minQps := int64(0)
	if info.ScalingInfo != nil {
		minQps = info.ScalingInfo.RequestedMinQps
	}

	return &vectorsearch.CreateEndpoint{
		BudgetPolicyId:  info.EffectiveBudgetPolicyId,
		EndpointType:    info.EndpointType,
		MinQps:          minQps,
		Name:            info.Name,
		ForceSendFields: utils.FilterFields[vectorsearch.CreateEndpoint](info.ForceSendFields),
	}
}

func (r *ResourceVectorSearchEndpoint) DoRead(ctx context.Context, id string) (*vectorsearch.EndpointInfo, error) {
	return r.client.VectorSearchEndpoints.GetEndpoint(ctx, vectorsearch.GetEndpointRequest{
		EndpointName: id,
	})
}

func (r *ResourceVectorSearchEndpoint) DoCreate(ctx context.Context, config *vectorsearch.CreateEndpoint) (string, *vectorsearch.EndpointInfo, error) {
	wait, err := r.client.VectorSearchEndpoints.CreateEndpoint(ctx, *config)
	if err != nil {
		return "", nil, err
	}
	info, err := wait.GetWithTimeout(20 * time.Minute)
	if err != nil {
		return "", nil, err
	}
	if info == nil {
		return config.Name, nil, nil
	}
	return info.Name, info, nil
}

// DoUpdate updates the endpoint in place using the patch-endpoint API.
// endpoint_type and name changes trigger recreate (see resources.yml).
func (r *ResourceVectorSearchEndpoint) DoUpdate(ctx context.Context, id string, config *vectorsearch.CreateEndpoint, changes Changes) (*vectorsearch.EndpointInfo, error) {
	if changes.HasChange(pathMinQps) {
		req := vectorsearch.PatchEndpointRequest{
			EndpointName:    id,
			MinQps:          config.MinQps,
			ForceSendFields: utils.FilterFields[vectorsearch.PatchEndpointRequest](config.ForceSendFields),
		}
		return r.client.VectorSearchEndpoints.PatchEndpoint(ctx, req)
	}

	return nil, nil
}

func (r *ResourceVectorSearchEndpoint) DoDelete(ctx context.Context, id string) error {
	return r.client.VectorSearchEndpoints.DeleteEndpoint(ctx, vectorsearch.DeleteEndpointRequest{
		EndpointName: id,
	})
}
