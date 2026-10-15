package resources

import (
	"context"
	"net/url"

	"github.com/databricks/cli/libs/sdkclients"
	"github.com/databricks/cli/libs/sdktypes/clusterpolicies/v2"
	"github.com/databricks/cli/libs/utils"
	sdkclusterpolicies "github.com/databricks/sdk-go/clusterpolicies/v2"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/workspaceurls"
)

type ClusterPolicy struct {
	BaseResource
	ID string `json:"id,omitempty" bundle:"readonly"`
	clusterpolicies.CreatePolicyRequest

	// Shadows the embedded compute.CreatePolicy.Definition (a string). `any` lets the
	// definition be authored as inline YAML; ConfigureClusterPolicyDefinition normalizes
	// it to a JSON string before deploy.
	Definition any `json:"definition,omitempty"`

	// Shadows the embedded compute.CreatePolicy.PolicyFamilyDefinitionOverrides (a string),
	// same as Definition: also a policy document authorable as inline YAML.
	PolicyFamilyDefinitionOverrides any `json:"policy_family_definition_overrides,omitempty"`

	Permissions []ClusterPolicyPermission `json:"permissions,omitempty"`
}

func (s *ClusterPolicy) Exists(ctx context.Context, w *sdkclients.Clients, id string) (bool, error) {
	_, err := w.ClusterPolicies.GetPolicy(ctx, sdkclusterpolicies.GetPolicyRequest{PolicyId: new(id)})
	if err != nil {
		log.Debugf(ctx, "cluster policy %s does not exist", id)
		return false, err
	}
	return true, nil
}

func (*ClusterPolicy) ResourceDescription() ResourceDescription {
	return ResourceDescription{
		SingularName:  "cluster_policy",
		PluralName:    "cluster_policies",
		SingularTitle: "Cluster Policy",
		PluralTitle:   "Cluster Policies",
	}
}

func (s *ClusterPolicy) InitializeURL(baseURL url.URL) {
	if s.ID == "" {
		return
	}
	s.URL = workspaceurls.ResourceURL(baseURL, "cluster_policies", s.ID)
}

func (s *ClusterPolicy) GetName() string {
	return utils.Deref(s.Name)
}
