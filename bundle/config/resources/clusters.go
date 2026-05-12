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

type Cluster struct {
	BaseResource
	compute.ClusterSpec

	Permissions []ClusterPermission `json:"permissions,omitempty"`
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
	s.URL = workspaceurls.ResourceURL(baseURL, "clusters", s.ID)
}

func (s *Cluster) GetName() string {
	return s.ClusterName
}

func (s *Cluster) GetURL() string {
	return s.URL
}
