package resources

import (
	"context"
	"net/url"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/telemetry/protos"
	"github.com/databricks/cli/libs/workspaceurls"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/marshal"
	"github.com/databricks/databricks-sdk-go/service/compute"
)

type Cluster struct {
	BaseResource
	ID string `json:"id,omitempty" bundle:"readonly"`
	compute.ClusterSpec

	// Lifecycle shadows BaseResource.Lifecycle to add support for lifecycle.started.
	Lifecycle *LifecycleWithStarted `json:"lifecycle,omitempty"`

	// Libraries are installed via the Libraries API, not the cluster spec.
	Libraries []compute.Library `json:"libraries,omitempty"`

	Permissions []ClusterPermission `json:"permissions,omitempty"`
}

// GetLifecycle returns the lifecycle settings, using LifecycleWithStarted.
func (s *Cluster) GetLifecycle() LifecycleConfig {
	if s.Lifecycle == nil {
		return LifecycleWithStarted{}
	}
	return *s.Lifecycle
}

func (*Cluster) LifecycleStartedTelemetryKey() string {
	return "cluster_lifecycle_started"
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

func (*Cluster) RunAsSupport() RunAsSupport {
	return RunAsNotApplicable
}

func (*Cluster) Approval() Approval {
	return Approval{}
}

func (*Cluster) ResourceDescription() ResourceDescription {
	return ResourceDescription{
		SingularName:  "cluster",
		PluralName:    "clusters",
		SingularTitle: "Cluster",
		PluralTitle:   "Clusters",
	}
}

func (*Cluster) PermissionsSupport() PermissionsSupport {
	return PermissionsSupport{
		// https://docs.databricks.com/aws/en/security/auth/access-control/#compute-acls
		BundleLevels: BundlePermissionLevels{
			CanManage: "CAN_MANAGE",
			CanView:   "CAN_ATTACH_TO",
			CanRun:    "CAN_RESTART",
		},
		RequestObjectType:  "clusters",
		ResponseObjectType: "cluster",
		ObjectIDField:      "id",
	}
}

func (*Cluster) WorkspaceURL() (workspaceurls.Pattern, bool) {
	return workspaceurls.Pattern{Path: "compute/clusters/%s"}, true
}

func (s *Cluster) InitializeURL(baseURL url.URL) {
	if s.ID == "" {
		return
	}
	s.URL = workspaceURL(s, baseURL, s.ID)
}

func (s *Cluster) GetName() string {
	return s.ClusterName
}

func (s *Cluster) ApplyPresets(p *PresetSettings) {
	s.ClusterName = p.NamePrefix + s.ClusterName
	if len(p.Tags) > 0 {
		if s.CustomTags == nil {
			s.CustomTags = make(map[string]string, len(p.Tags))
		}
		for _, tag := range p.Tags {
			normalisedKey := p.Tagging.NormalizeKey(tag.Key)
			normalisedValue := p.Tagging.NormalizeValue(tag.Value)
			if _, ok := s.CustomTags[normalisedKey]; !ok {
				s.CustomTags[normalisedKey] = normalisedValue
			}
		}
	}
}

func (s *Cluster) FieldDefaults(DefaultsContext) []FieldDefault {
	return []FieldDefault{
		// Same as terraform
		// https://github.com/databricks/terraform-provider-databricks/blob/v1.75.0/clusters/resource_cluster.go#L315
		{"autotermination_minutes", 60},
		{"workload_type.clients.notebooks", true},
		{"workload_type.clients.jobs", true},
	}
}

func (s *Cluster) GetID() string {
	return s.ID
}

func (*Cluster) DeployTelemetryIDsField(event *protos.BundleDeployEvent) *[]string {
	return &event.ResourceClusterIDs
}

func (*Cluster) DeployTelemetryCountField(event *protos.BundleDeployEvent) *int64 {
	return &event.ResourceClusterCount
}
