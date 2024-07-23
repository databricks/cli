// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceDashboard struct {
	CreateTime              string `json:"create_time,omitempty"`
	DashboardChangeDetected bool   `json:"dashboard_change_detected,omitempty"`
	DashboardId             string `json:"dashboard_id,omitempty"`
	DisplayName             string `json:"display_name"`
	EmbedCredentials        bool   `json:"embed_credentials,omitempty"`
	Etag                    string `json:"etag,omitempty"`
	FilePath                string `json:"file_path,omitempty"`
	Id                      string `json:"id,omitempty"`
	LifecycleState          string `json:"lifecycle_state,omitempty"`
	Md5                     string `json:"md5,omitempty"`
	ParentPath              string `json:"parent_path"`
	Path                    string `json:"path,omitempty"`
	SerializedDashboard     string `json:"serialized_dashboard,omitempty"`
	UpdateTime              string `json:"update_time,omitempty"`
	WarehouseId             string `json:"warehouse_id"`
}
