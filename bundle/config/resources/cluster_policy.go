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
	compute.CreatePolicy

	// Shadows the embedded compute.CreatePolicy.Definition (a string). `any` lets the
	// definition be authored as inline YAML; ConfigureClusterPolicyDefinition normalizes
	// it to a JSON string before deploy.
	Definition any `json:"definition,omitempty"`
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
	return s.Name
}

func (s *ClusterPolicy) GetURL() string {
	return s.URL
}
