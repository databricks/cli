package resources

import (
	"context"
	"net/url"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/workspaceurls"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/marshal"
	"github.com/databricks/databricks-sdk-go/service/sql"
)

type SqlWarehouse struct {
	BaseResource
	ID string `json:"id,omitempty" bundle:"readonly"`
	sql.CreateWarehouseRequest

	// Lifecycle shadows BaseResource.Lifecycle to add support for lifecycle.started.
	Lifecycle *LifecycleWithStarted `json:"lifecycle,omitempty"`

	Permissions []SqlWarehousePermission `json:"permissions,omitempty"`
}

// GetLifecycle returns the lifecycle settings, using LifecycleWithStarted.
func (sw *SqlWarehouse) GetLifecycle() LifecycleConfig {
	if sw.Lifecycle == nil {
		return LifecycleWithStarted{}
	}
	return *sw.Lifecycle
}

func (*SqlWarehouse) LifecycleStartedTelemetryKey() string {
	return "sql_warehouse_lifecycle_started"
}

func (sw *SqlWarehouse) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, sw)
}

func (sw SqlWarehouse) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(sw)
}

func (sw *SqlWarehouse) Exists(ctx context.Context, w *databricks.WorkspaceClient, id string) (bool, error) {
	_, err := w.Warehouses.GetById(ctx, id)
	if err != nil {
		log.Debugf(ctx, "sql warehouse %s does not exist", id)
		return false, err
	}
	return true, nil
}

func (*SqlWarehouse) RunAsSupport() RunAsSupport {
	return RunAsNotApplicable
}

func (*SqlWarehouse) Approval() Approval {
	return Approval{}
}

func (*SqlWarehouse) ResourceDescription() ResourceDescription {
	return ResourceDescription{
		SingularName:  "sql_warehouse",
		PluralName:    "sql_warehouses",
		SingularTitle: "Sql Warehouse",
		PluralTitle:   "SQL Warehouses",
	}
}

func (*SqlWarehouse) PermissionsSupport() PermissionsSupport {
	return PermissionsSupport{
		BundleLevels: BundlePermissionLevels{
			CanManage: "CAN_MANAGE",
			CanView:   "CAN_VIEW",
			CanRun:    "CAN_MONITOR",
		},
		RequestObjectType:  "sql/warehouses",
		ResponseObjectType: "warehouses",
		ObjectIDField:      "id",
		HasIsOwner:         true,
	}
}

func (*SqlWarehouse) WorkspaceURL() (workspaceurls.Pattern, bool) {
	return workspaceurls.Pattern{Path: "sql/warehouses/%s"}, true
}

func (sw *SqlWarehouse) InitializeURL(baseURL url.URL) {
	if sw.ID == "" {
		return
	}
	sw.URL = workspaceURL(sw, baseURL, sw.ID)
}

func (sw *SqlWarehouse) GetName() string {
	return sw.Name
}

func (sw *SqlWarehouse) ApplyPresets(p *PresetSettings) {
	sw.Name = p.NamePrefix + sw.Name
	if len(p.Tags) > 0 {
		if sw.Tags == nil {
			sw.Tags = &sql.EndpointTags{}
		}
		for _, tag := range p.Tags {
			normalisedKey := p.Tagging.NormalizeKey(tag.Key)
			normalisedValue := p.Tagging.NormalizeValue(tag.Value)

			// Check if the tag already exists
			exists := false
			for _, t := range sw.Tags.CustomTags {
				if t.Key == normalisedKey {
					exists = true
					break
				}
			}

			if !exists {
				sw.Tags.CustomTags = append(sw.Tags.CustomTags, sql.EndpointTagPair{
					Key:   normalisedKey,
					Value: normalisedValue,
				})
			}
		}
	}
}

func (sw *SqlWarehouse) FieldDefaults(DefaultsContext) []FieldDefault {
	return []FieldDefault{
		// Same as terraform
		// https://github.com/databricks/terraform-provider-databricks/blob/v1.75.0/sql/resource_sql_endpoint.go#L59
		{"auto_stop_mins", 120},
		{"enable_photon", true},
		{"max_num_clusters", 1},
		{"spot_instance_policy", "COST_OPTIMIZED"},
	}
}
