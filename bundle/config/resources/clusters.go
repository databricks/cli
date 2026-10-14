package resources

import (
	"context"
	"github.com/databricks/cli/libs/sdkclients"
	"github.com/databricks/cli/libs/sdktypes/clusterlibraries/v2"
	"github.com/databricks/cli/libs/sdktypes/clusters/v2"
	"github.com/databricks/cli/libs/utils"
	sdkclusters "github.com/databricks/sdk-go/clusters/v2"
	"net/url"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/workspaceurls"
)

type Cluster struct {
	BaseResource
	ID string `json:"id,omitempty" bundle:"readonly"`
	clusters.CreateClusterRequest

	// Lifecycle shadows BaseResource.Lifecycle to add support for lifecycle.started.
	Lifecycle *LifecycleWithStarted `json:"lifecycle,omitempty"`

	// Libraries are installed via the Libraries API, not the cluster spec.
	Libraries []clusterlibraries.Library `json:"libraries,omitempty"`

	Permissions []ClusterPermission `json:"permissions,omitempty"`
}

// GetLifecycle returns the lifecycle settings, using LifecycleWithStarted.
func (s *Cluster) GetLifecycle() LifecycleConfig {
	if s.Lifecycle == nil {
		return LifecycleWithStarted{}
	}
	return *s.Lifecycle
}

func (s *Cluster) Exists(ctx context.Context, w *sdkclients.Clients, id string) (bool, error) {
	_, err := w.Clusters.GetCluster(ctx, sdkclusters.GetClusterRequest{ClusterId: new(id)})
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
	return utils.Deref(s.ClusterName)
}
