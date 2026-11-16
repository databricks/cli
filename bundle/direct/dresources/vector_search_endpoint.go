package dresources

import (
	"context"
	"errors"
	"time"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/retry"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/utils"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/marshal"
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

// DoCreate retries transient errors itself rather than marking them retrySafe: the backend
// can answer 504 TEMPORARILY_UNAVAILABLE after it has already created the endpoint, and the
// retry is then rejected as already existing. In that case the endpoint is used if it was
// created after the first attempt started; an older one (another user's endpoint, or the
// old endpoint of a recreate that is still being deleted) keeps failing the deploy.
func (r *ResourceVectorSearchEndpoint) DoCreate(ctx context.Context, config *vectorsearch.CreateEndpoint) (string, *VectorSearchEndpointRemote, error) {
	start := time.Now()
	afterTransient := false
	info, err := retry.With(ctx, retry.IsTransient, func() (*vectorsearch.EndpointInfo, error) {
		waiter, err := r.client.VectorSearchEndpoints.CreateEndpoint(ctx, *config)
		if err == nil {
			return waiter.Response, nil
		}
		if afterTransient && isEndpointAlreadyExists(err) {
			info, getErr := r.client.VectorSearchEndpoints.GetEndpointByEndpointName(ctx, config.Name)
			if getErr != nil {
				return nil, getErr
			}
			if !createdSince(info, start) {
				return nil, err
			}
			log.Warnf(ctx, "vector search endpoint %q was created by an earlier attempt that failed with a transient error, using it", config.Name)
			return info, nil
		}
		afterTransient = afterTransient || retry.IsTransient(ctx, err)
		return nil, err
	})
	if err != nil {
		return "", nil, err
	}
	return config.Name, newVectorSearchEndpointRemote(info), nil
}

// isEndpointAlreadyExists matches both codes: the cloud returns 409 ALREADY_EXISTS (observed on
// AWS, 2026-11), while RESOURCE_ALREADY_EXISTS is the generic code for the same conflict.
func isEndpointAlreadyExists(err error) bool {
	return errors.Is(err, apierr.ErrAlreadyExists) || errors.Is(err, apierr.ErrResourceAlreadyExists)
}

// endpointClockSkew allows for the server's clock running behind the client's.
const endpointClockSkew = time.Minute

// createdSince reports whether the endpoint was created no earlier than start, the time of the
// first create attempt, so that it can only have been created by that attempt.
func createdSince(info *vectorsearch.EndpointInfo, start time.Time) bool {
	return info.CreationTimestamp >= start.Add(-endpointClockSkew).UnixMilli()
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
