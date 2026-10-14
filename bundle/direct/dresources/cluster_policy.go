package dresources

import (
	"context"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/sdkclients"
	clusterpolicies "github.com/databricks/cli/libs/sdktypes/clusterpolicies/v2"
	"github.com/databricks/cli/libs/utils"
	sdkclusterpolicies "github.com/databricks/sdk-go/clusterpolicies/v2"
)

type ResourceClusterPolicy struct {
	client *sdkclients.Clients
}

func (*ResourceClusterPolicy) New(client *sdkclients.Clients) *ResourceClusterPolicy {
	return &ResourceClusterPolicy{client: client}
}

func (*ResourceClusterPolicy) PrepareState(input *resources.ClusterPolicy) *clusterpolicies.CreatePolicyRequest {
	cp := input.CreatePolicyRequest
	// Copy the shadow fields, already normalized to JSON strings by ConfigureClusterPolicyDefinition.
	if s, ok := input.Definition.(string); ok {
		cp.Definition = new(s)
	}
	if s, ok := input.PolicyFamilyDefinitionOverrides.(string); ok {
		cp.PolicyFamilyDefinitionOverrides = new(s)
	}
	return &cp
}

// RemapState copies the config fields shared by Policy and CreatePolicyRequest;
// output-only fields (policy_id, created_at_timestamp, creator_user_name, is_default) are not in the state.
func (*ResourceClusterPolicy) RemapState(remote *clusterpolicies.Policy) *clusterpolicies.CreatePolicyRequest {
	return &clusterpolicies.CreatePolicyRequest{
		Definition:                      remote.Definition,
		Description:                     remote.Description,
		Libraries:                       remote.Libraries,
		MaxClustersPerUser:              remote.MaxClustersPerUser,
		Name:                            remote.Name,
		PolicyFamilyDefinitionOverrides: remote.PolicyFamilyDefinitionOverrides,
		PolicyFamilyId:                  remote.PolicyFamilyId,
	}
}

func (r *ResourceClusterPolicy) DoRead(ctx context.Context, id string) (*clusterpolicies.Policy, error) {
	policy, err := r.client.ClusterPolicies.GetPolicy(ctx, sdkclusterpolicies.GetPolicyRequest{PolicyId: new(id)})
	if err != nil {
		return nil, err
	}
	return clusterpolicies.PolicyFromSDK(policy)
}

func (r *ResourceClusterPolicy) DoCreate(ctx context.Context, config *clusterpolicies.CreatePolicyRequest) (string, *clusterpolicies.Policy, error) {
	request, err := clusterpolicies.CreatePolicyRequestToSDK(config)
	if err != nil {
		return "", nil, err
	}
	resp, err := r.client.ClusterPolicies.CreatePolicy(ctx, *request)
	if err != nil {
		return "", nil, err
	}
	return utils.Deref(resp.PolicyId), nil, nil
}

func (r *ResourceClusterPolicy) DoUpdate(ctx context.Context, id string, config *clusterpolicies.CreatePolicyRequest, _ *PlanEntry) (*clusterpolicies.Policy, error) {
	request, err := clusterpolicies.EditPolicyRequestToSDK(&clusterpolicies.EditPolicyRequest{
		PolicyId:                        new(id),
		Name:                            config.Name,
		Definition:                      config.Definition,
		Description:                     config.Description,
		Libraries:                       config.Libraries,
		MaxClustersPerUser:              config.MaxClustersPerUser,
		PolicyFamilyDefinitionOverrides: config.PolicyFamilyDefinitionOverrides,
		PolicyFamilyId:                  config.PolicyFamilyId,
	})
	if err != nil {
		return nil, err
	}
	_, err = r.client.ClusterPolicies.EditPolicy(ctx, *request)
	return nil, err
}

func (r *ResourceClusterPolicy) DoDelete(ctx context.Context, id string, _ *clusterpolicies.CreatePolicyRequest) error {
	_, err := r.client.ClusterPolicies.DeletePolicy(ctx, sdkclusterpolicies.DeletePolicyRequest{PolicyId: new(id)})
	return err
}
