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

type InstancePool struct {
	BaseResource
	compute.CreateInstancePool
	Permissions []InstancePoolPermission `json:"permissions,omitempty"`
}

func (s *InstancePool) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, s)
}

func (s InstancePool) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(s)
}

func (s *InstancePool) Exists(ctx context.Context, w *databricks.WorkspaceClient, id string) (bool, error) {
	_, err := w.InstancePools.GetByInstancePoolId(ctx, id)
	if err != nil {
		log.Debugf(ctx, "instance pool %s does not exist", id)
		return false, err
	}
	return true, nil
}

func (*InstancePool) ResourceDescription() ResourceDescription {
	return ResourceDescription{
		SingularName:  "instance_pool",
		PluralName:    "instance_pools",
		SingularTitle: "Instance Pool",
		PluralTitle:   "Instance Pools",
	}
}

func (s *InstancePool) InitializeURL(baseURL url.URL) {
	if s.ID == "" {
		return
	}
	s.URL = workspaceurls.ResourceURL(baseURL, "instance_pools", s.ID)
}

func (s *InstancePool) GetName() string {
	return s.InstancePoolName
}

func (s *InstancePool) GetURL() string {
	return s.URL
}
