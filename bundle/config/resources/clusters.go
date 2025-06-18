package resources

import (
	"context"
	"net/url"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/marshal"
	"github.com/databricks/databricks-sdk-go/service/compute"
)

type ClusterPermissionLevel string

// ClusterPermission holds the permission level setting for a single principal.
// Multiple of these can be defined on any cluster.
type ClusterPermission struct {
	Level ClusterPermissionLevel `json:"level"`

	UserName             string `json:"user_name,omitempty"`
	ServicePrincipalName string `json:"service_principal_name,omitempty"`
	GroupName            string `json:"group_name,omitempty"`
}

type Cluster struct {
	ID             string              `json:"id,omitempty" bundle:"readonly"`
	Permissions    []ClusterPermission `json:"permissions,omitempty"`
	ModifiedStatus ModifiedStatus      `json:"modified_status,omitempty" bundle:"internal"`
	URL            string              `json:"url,omitempty" bundle:"internal"`

	compute.ClusterSpec
}

func (s *Cluster) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, s)
}

func (s Cluster) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(s)
}

func (s *Cluster) Exists(ctx context.Context, w *databricks.WorkspaceClient, id string) (bool, error) {
	_, err := w.Clusters.GetByClusterId(ctx, id)
	if err != nil {
		log.Debugf(ctx, "cluster %s does not exist", id)
		return false, err
	}
	return true, nil
}

func (*Cluster) ResourceDescription() ResourceDescription {
	return ResourceDescription{
		SingularName:  "cluster",
		PluralName:    "clusters",
		SingularTitle: "Cluster",
		PluralTitle:   "Clusters",
	}
}

func (s *Cluster) InitializeURL(baseURL url.URL) {
	if s.ID == "" {
		return
	}
	baseURL.Path = "compute/clusters/" + s.ID
	s.URL = baseURL.String()
}

func (s *Cluster) GetName() string {
	return s.ClusterName
}

func (s *Cluster) GetURL() string {
	return s.URL
}
