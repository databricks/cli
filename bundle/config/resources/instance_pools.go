package resources

import (
	"context"
	"net/url"

	"github.com/databricks/cli/libs/sdkclients"
	"github.com/databricks/cli/libs/sdktypes/instancepools/v2"
	"github.com/databricks/cli/libs/utils"
	sdkinstancepools "github.com/databricks/sdk-go/instancepools/v2"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/workspaceurls"
)

type InstancePool struct {
	BaseResource
	ID string `json:"id,omitempty" bundle:"readonly"`
	instancepools.CreateInstancePoolRequest
	Permissions []InstancePoolPermission `json:"permissions,omitempty"`
}

func (s *InstancePool) Exists(ctx context.Context, w *sdkclients.Clients, id string) (bool, error) {
	_, err := w.InstancePools.GetInstancePool(ctx, sdkinstancepools.GetInstancePoolRequest{InstancePoolId: new(id)})
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
	return utils.Deref(s.InstancePoolName)
}
