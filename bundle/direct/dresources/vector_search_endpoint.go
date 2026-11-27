package dresources

import (
	"context"
	"time"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/utils"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/marshal"
	"github.com/databricks/databricks-sdk-go/service/vectorsearch"
)

var (
	pathBudgetPolicyId = structpath.NewPath(nil, "budget_policy_id")
	pathTargetQps      = structpath.NewPath(nil, "target_qps")
)

// VectorSearchEndpointRemote is remote state for a vector search endpoint. It embeds API response
// fields for drift comparison and adds endpoint_uuid for permissions; deployment state id remains the endpoint name.
type VectorSearchEndpointRemote struct {
	vectorsearch.EndpointInfo
	EndpointUuid string `json:"endpoint_uuid"`
	// TargetQps is mapped from EndpointInfo.ScalingInfo.RequestedTargetQps in DoRead
	// so that drift detection can compare it directly against the config field.
	TargetQps int64 `json:"target_qps,omitempty"`
}

// Custom marshalers needed because embedded vectorsearch.EndpointInfo has its own
// MarshalJSON which would otherwise take over and ignore endpoint_uuid.
func (s *VectorSearchEndpointRemote) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, s)
}

func (s VectorSearchEndpointRemote) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(s)
}

func newVectorSearchEndpointRemote(info *vectorsearch.EndpointInfo) *VectorSearchEndpointRemote {
	var targetQps int64
	if info.ScalingInfo != nil {
		targetQps = info.ScalingInfo.RequestedTargetQps
	}
	return &VectorSearchEndpointRemote{
		EndpointInfo: *info,
		EndpointUuid: info.Id,
		TargetQps:    targetQps,
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
	return &vectorsearch.CreateEndpoint{
		Name:            remote.Name,
		EndpointType:    remote.EndpointType,
		BudgetPolicyId:  remote.BudgetPolicyId,
		UsagePolicyId:   "", // Missing in remote
		TargetQps:       remote.TargetQps,
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
	create := func(ctx context.Context) (*vectorsearch.EndpointInfo, error) {
		waiter, err := r.client.VectorSearchEndpoints.CreateEndpoint(ctx, *config)
		if err != nil {
			return nil, err
		}
		return waiter.Response, nil
	}
	start := time.Now()
	info, err := create(ctx)
	if err != nil {
		// The backend can answer 504 TEMPORARILY_UNAVAILABLE after it has created the endpoint.
		// Endpoint names are chosen by the bundle and unique in the workspace; an older endpoint
		// (another user's, or the old one of a recreate that is still being deleted) keeps failing
		// the deploy.
		info, err = recoverCreate(ctx, createRecovery[vectorsearch.EndpointInfo, *vectorsearch.EndpointInfo]{
			name:  "vector search endpoint " + config.Name,
			start: start,
			get: func(ctx context.Context) (*vectorsearch.EndpointInfo, error) {
				return r.client.VectorSearchEndpoints.GetEndpointByEndpointName(ctx, config.Name)
			},
			createTime: func(found *vectorsearch.EndpointInfo) (time.Time, bool) {
				return unixMilliTime(found.CreationTimestamp)
			},
			alreadyExists:   isAlreadyExists,
			createdByCaller: nil,
			use:             useFound[vectorsearch.EndpointInfo],
			create:          repeatCreate(create),
		}, err)
		if err != nil {
			return "", nil, err
		}
	}
	return config.Name, newVectorSearchEndpointRemote(info), nil
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

func (r *ResourceVectorSearchEndpoint) DoDelete(ctx context.Context, id string, _ *vectorsearch.CreateEndpoint) error {
	return r.client.VectorSearchEndpoints.DeleteEndpointByEndpointName(ctx, id)
}
