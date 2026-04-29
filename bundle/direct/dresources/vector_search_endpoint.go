package dresources

import (
	"context"
	"time"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/utils"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/marshal"
	"github.com/databricks/databricks-sdk-go/service/vectorsearch"
)

var (
	pathBudgetPolicyId = structpath.MustParsePath("budget_policy_id")
	pathMinQps         = structpath.MustParsePath("min_qps")
)

// VectorSearchEndpointState is persisted in deployment state. endpoint_uuid is
// tracked so out-of-band replacement of an endpoint with the same name can be
// detected: when saved UUID differs from remote UUID, the endpoint is recreated.
type VectorSearchEndpointState struct {
	vectorsearch.CreateEndpoint
	EndpointUuid string `json:"endpoint_uuid,omitempty"`
}

// Custom marshalers required because embedded CreateEndpoint has its own.
func (s *VectorSearchEndpointState) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, s)
}

func (s VectorSearchEndpointState) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(s)
}

// VectorSearchEndpointRemote is remote state for a vector search endpoint. It embeds API response
// fields for drift comparison and adds endpoint_uuid for permissions; deployment state id remains the endpoint name.
type VectorSearchEndpointRemote struct {
	*vectorsearch.EndpointInfo
	EndpointUuid string `json:"endpoint_uuid"`
}

func newVectorSearchEndpointRemote(info *vectorsearch.EndpointInfo) *VectorSearchEndpointRemote {
	if info == nil {
		return nil
	}
	return &VectorSearchEndpointRemote{
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

func (*ResourceVectorSearchEndpoint) PrepareState(input *resources.VectorSearchEndpoint) *VectorSearchEndpointState {
	return &VectorSearchEndpointState{
		CreateEndpoint: input.CreateEndpoint,
		EndpointUuid:   "",
	}
}

func (*ResourceVectorSearchEndpoint) RemapState(remote *VectorSearchEndpointRemote) *VectorSearchEndpointState {
	var minQps int64
	if remote.ScalingInfo != nil {
		minQps = remote.ScalingInfo.RequestedMinQps
	}
	return &VectorSearchEndpointState{
		CreateEndpoint: vectorsearch.CreateEndpoint{
			Name:            remote.Name,
			EndpointType:    remote.EndpointType,
			BudgetPolicyId:  remote.BudgetPolicyId,
			UsagePolicyId:   "", // Missing in remote
			MinQps:          minQps,
			ForceSendFields: utils.FilterFields[vectorsearch.CreateEndpoint](remote.ForceSendFields, "UsagePolicyId"),
		},
		EndpointUuid: remote.EndpointUuid,
	}
}

func (r *ResourceVectorSearchEndpoint) DoRead(ctx context.Context, id string) (*VectorSearchEndpointRemote, error) {
	info, err := r.client.VectorSearchEndpoints.GetEndpointByEndpointName(ctx, id)
	if err != nil {
		return nil, err
	}
	return newVectorSearchEndpointRemote(info), nil
}

func (r *ResourceVectorSearchEndpoint) DoCreate(ctx context.Context, config *VectorSearchEndpointState) (string, *VectorSearchEndpointRemote, error) {
	waiter, err := r.client.VectorSearchEndpoints.CreateEndpoint(ctx, config.CreateEndpoint)
	if err != nil {
		return "", nil, err
	}
	id := config.Name
	if waiter.Response != nil {
		config.EndpointUuid = waiter.Response.Id
	}
	return id, newVectorSearchEndpointRemote(waiter.Response), nil
}

func (r *ResourceVectorSearchEndpoint) WaitAfterCreate(ctx context.Context, config *VectorSearchEndpointState) (*VectorSearchEndpointRemote, error) {
	info, err := r.client.VectorSearchEndpoints.WaitGetEndpointVectorSearchEndpointOnline(ctx, config.Name, 60*time.Minute, nil)
	if err != nil {
		return nil, err
	}
	return newVectorSearchEndpointRemote(info), nil
}

func (r *ResourceVectorSearchEndpoint) DoUpdate(ctx context.Context, id string, config *VectorSearchEndpointState, entry *PlanEntry) (*VectorSearchEndpointRemote, error) {
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
			ForceSendFields: nil,
		})
		if err != nil {
			return nil, err
		}
	}

	// Preserve endpoint_uuid in saved state: PrepareState leaves it empty because
	// it isn't in config, so copy from remote before SaveState writes newState.
	if remote, ok := entry.RemoteState.(*VectorSearchEndpointRemote); ok && remote != nil {
		config.EndpointUuid = remote.EndpointUuid
	}

	return nil, nil
}

func (r *ResourceVectorSearchEndpoint) DoDelete(ctx context.Context, id string) error {
	return r.client.VectorSearchEndpoints.DeleteEndpointByEndpointName(ctx, id)
}

// OverrideChangeDesc classifies endpoint_uuid drift: Recreate when saved UUID
// differs from remote (endpoint replaced out-of-band), Skip otherwise. This
// field is not in config, so a synthetic diff between saved state and an empty
// newState is expected on every plan.
func (*ResourceVectorSearchEndpoint) OverrideChangeDesc(_ context.Context, path *structpath.PathNode, change *ChangeDesc, remote *VectorSearchEndpointRemote) error {
	if path.String() != "endpoint_uuid" {
		return nil
	}
	savedUuid, _ := change.Old.(string)
	var remoteUuid string
	if remote != nil {
		remoteUuid = remote.EndpointUuid
	}
	if savedUuid != "" && remoteUuid != "" && savedUuid != remoteUuid {
		change.Action = deployplan.Recreate
		change.Reason = "endpoint replaced out-of-band"
	} else {
		change.Action = deployplan.Skip
		change.Reason = "state-only field"
	}
	return nil
}
