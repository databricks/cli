// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceDashboardsDashboards struct {
	CreateTime          string `json:"create_time,omitempty"`
	DashboardId         string `json:"dashboard_id,omitempty"`
	DisplayName         string `json:"display_name,omitempty"`
	Etag                string `json:"etag,omitempty"`
	LifecycleState      string `json:"lifecycle_state,omitempty"`
	ParentPath          string `json:"parent_path,omitempty"`
	Path                string `json:"path,omitempty"`
	SerializedDashboard string `json:"serialized_dashboard,omitempty"`
	UpdateTime          string `json:"update_time,omitempty"`
	WarehouseId         string `json:"warehouse_id,omitempty"`
}

type DataSourceDashboards struct {
	DashboardNameContains string                           `json:"dashboard_name_contains,omitempty"`
	Dashboards            []DataSourceDashboardsDashboards `json:"dashboards,omitempty"`
}
