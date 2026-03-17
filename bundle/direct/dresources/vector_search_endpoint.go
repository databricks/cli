package dresources

import (
	"context"
	"time"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/vectorsearch"
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

func (*ResourceVectorSearchEndpoint) RemapState(info *vectorsearch.EndpointInfo) *vectorsearch.CreateEndpoint {
	return &vectorsearch.CreateEndpoint{
		BudgetPolicyId:  info.EffectiveBudgetPolicyId,
		EndpointType:    info.EndpointType,
		MinQps:          0,
		Name:            info.Name,
		ForceSendFields: nil,
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

func (r *ResourceVectorSearchEndpoint) DoDelete(ctx context.Context, id string) error {
	return r.client.VectorSearchEndpoints.DeleteEndpoint(ctx, vectorsearch.DeleteEndpointRequest{
		EndpointName: id,
	})
}
