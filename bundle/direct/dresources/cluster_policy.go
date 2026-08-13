package dresources

import (
	"context"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/utils"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/compute"
)

type ResourceClusterPolicy struct {
	client *databricks.WorkspaceClient
}

func (*ResourceClusterPolicy) New(client *databricks.WorkspaceClient) *ResourceClusterPolicy {
	return &ResourceClusterPolicy{client: client}
}

func (*ResourceClusterPolicy) PrepareState(input *resources.ClusterPolicy) *compute.CreatePolicy {
	cp := input.CreatePolicy
	// The top-level Definition shadows the embedded string; ConfigureClusterPolicyDefinition
	// has already normalized it to a JSON string by this point.
	if s, ok := input.Definition.(string); ok {
		cp.Definition = s
	}
	return &cp
}

// RemapState copies the config fields shared by Policy and CreatePolicy;
// output-only fields (policy_id, created_at_timestamp, creator_user_name, is_default) are not in the state.
func (*ResourceClusterPolicy) RemapState(remote *compute.Policy) *compute.CreatePolicy {
	return &compute.CreatePolicy{
		Definition:                      remote.Definition,
		Description:                     remote.Description,
		Libraries:                       remote.Libraries,
		MaxClustersPerUser:              remote.MaxClustersPerUser,
		Name:                            remote.Name,
		PolicyFamilyDefinitionOverrides: remote.PolicyFamilyDefinitionOverrides,
		PolicyFamilyId:                  remote.PolicyFamilyId,
		ForceSendFields:                 utils.FilterFields[compute.CreatePolicy](remote.ForceSendFields),
	}
}

func (r *ResourceClusterPolicy) DoRead(ctx context.Context, id string) (*compute.Policy, error) {
	return r.client.ClusterPolicies.GetByPolicyId(ctx, id)
}

func (r *ResourceClusterPolicy) DoCreate(ctx context.Context, config *compute.CreatePolicy) (string, *compute.Policy, error) {
	resp, err := r.client.ClusterPolicies.Create(ctx, *config)
	if err != nil {
		return "", nil, err
	}

	return resp.PolicyId, nil, nil
}

func (r *ResourceClusterPolicy) DoUpdate(ctx context.Context, id string, config *compute.CreatePolicy, _ *PlanEntry) (*compute.Policy, error) {
	return nil, r.client.ClusterPolicies.Edit(ctx, compute.EditPolicy{
		PolicyId:                        id,
		Name:                            config.Name,
		Definition:                      config.Definition,
		Description:                     config.Description,
		Libraries:                       config.Libraries,
		MaxClustersPerUser:              config.MaxClustersPerUser,
		PolicyFamilyDefinitionOverrides: config.PolicyFamilyDefinitionOverrides,
		PolicyFamilyId:                  config.PolicyFamilyId,
		ForceSendFields:                 utils.FilterFields[compute.EditPolicy](config.ForceSendFields),
	})
}

func (r *ResourceClusterPolicy) DoDelete(ctx context.Context, id string, _ *compute.CreatePolicy) error {
	return r.client.ClusterPolicies.DeleteByPolicyId(ctx, id)
}
