package resources

import (
	"context"
	"net/url"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/workspaceurls"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/marshal"
	"github.com/databricks/databricks-sdk-go/service/compute"
)

type ClusterPolicy struct {
	BaseResource
	ID string `json:"id,omitempty" bundle:"readonly"`
	compute.CreatePolicy

	// Shadows the embedded compute.CreatePolicy.Definition (a string). `any` lets the
	// definition be authored as inline YAML; ConfigureClusterPolicyDefinition normalizes
	// it to a JSON string before deploy.
	Definition any `json:"definition,omitempty"`

	// Shadows the embedded compute.CreatePolicy.PolicyFamilyDefinitionOverrides (a string),
	// same as Definition: also a policy document authorable as inline YAML.
	PolicyFamilyDefinitionOverrides any `json:"policy_family_definition_overrides,omitempty"`

	Permissions []ClusterPolicyPermission `json:"permissions,omitempty"`
}

func (s *ClusterPolicy) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, s)
}

func (s ClusterPolicy) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(s)
}

func (s *ClusterPolicy) Exists(ctx context.Context, w *databricks.WorkspaceClient, id string) (bool, error) {
	_, err := w.ClusterPolicies.GetByPolicyId(ctx, id)
	if err != nil {
		log.Debugf(ctx, "cluster policy %s does not exist", id)
		return false, err
	}
	return true, nil
}

func (*ClusterPolicy) RunAsSupport() RunAsSupport {
	return RunAsNotApplicable
}

func (*ClusterPolicy) Approval() Approval {
	return Approval{}
}

func (*ClusterPolicy) ResourceDescription() ResourceDescription {
	return ResourceDescription{
		SingularName:  "cluster_policy",
		PluralName:    "cluster_policies",
		SingularTitle: "Cluster Policy",
		PluralTitle:   "Cluster Policies",
	}
}

func (*ClusterPolicy) PermissionsSupport() PermissionsSupport {
	return PermissionsSupport{
		RequestObjectType:  "cluster-policies",
		ResponseObjectType: "cluster-policy",
		ObjectIDField:      "id",
		// Cluster policies only support CAN_USE; injecting the current user as
		// CAN_MANAGE/IS_OWNER would be rejected by the permissions API.
		SkipFixup: true,
		// Only admins create policies and admins can use every policy, so the
		// creator's grant is never needed in the configuration.
		CreatorLevel: "CAN_USE",
	}
}

func (*ClusterPolicy) WorkspaceURL() (workspaceurls.Pattern, bool) {
	return workspaceurls.Pattern{Path: "compute/policies/%s"}, true
}

func (s *ClusterPolicy) InitializeURL(baseURL url.URL) {
	if s.ID == "" {
		return
	}
	s.URL = workspaceURL(s, baseURL, s.ID)
}

func (s *ClusterPolicy) GetName() string {
	return s.Name
}

// ApplyPresets prefixes the policy name: it is a user-facing display name
// (unique, 1-100 chars), not the API id (policy_id), so prefixing it in dev
// mode avoids collisions between developers without changing identity.
func (s *ClusterPolicy) ApplyPresets(p *PresetSettings) {
	s.Name = p.NamePrefix + s.Name
}
