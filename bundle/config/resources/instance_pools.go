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
	ID string `json:"id,omitempty" bundle:"readonly"`
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

func (*InstancePool) RunAsSupport() RunAsSupport {
	return RunAsNotApplicable
}

func (*InstancePool) ResourceDescription() ResourceDescription {
	return ResourceDescription{
		SingularName:  "instance_pool",
		PluralName:    "instance_pools",
		SingularTitle: "Instance Pool",
		PluralTitle:   "Instance Pools",
	}
}

func (*InstancePool) PermissionsSupport() PermissionsSupport {
	return PermissionsSupport{
		BundleLevels: BundlePermissionLevels{
			CanManage: "CAN_MANAGE",
			CanView:   "CAN_ATTACH_TO",
		},
		RequestObjectType:  "instance-pools",
		ResponseObjectType: "instance-pool",
		ObjectIDField:      "id",
	}
}

func (*InstancePool) WorkspaceURL() (workspaceurls.Pattern, bool) {
	return workspaceurls.Pattern{Path: "compute/instance-pools/%s"}, true
}

func (s *InstancePool) InitializeURL(baseURL url.URL) {
	if s.ID == "" {
		return
	}
	s.URL = workspaceURL(s, baseURL, s.ID)
}

func (s *InstancePool) GetName() string {
	return s.InstancePoolName
}

func (s *InstancePool) ApplyPresets(p *PresetSettings) {
	s.InstancePoolName = p.NamePrefix + s.InstancePoolName
	if len(p.Tags) > 0 {
		if s.CustomTags == nil {
			s.CustomTags = make(map[string]string, len(p.Tags))
		}
		for _, tag := range p.Tags {
			k := p.Tagging.NormalizeKey(tag.Key)
			v := p.Tagging.NormalizeValue(tag.Value)
			if _, ok := s.CustomTags[k]; !ok {
				s.CustomTags[k] = v
			}
		}
	}
}
