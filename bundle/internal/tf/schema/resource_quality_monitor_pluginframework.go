// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceQualityMonitorPluginframework struct {
	AssetsDir               string   `json:"assets_dir"`
	BaselineTableName       string   `json:"baseline_table_name,omitempty"`
	DashboardId             string   `json:"dashboard_id,omitempty"`
	DriftMetricsTableName   string   `json:"drift_metrics_table_name,omitempty"`
	LatestMonitorFailureMsg string   `json:"latest_monitor_failure_msg,omitempty"`
	MonitorVersion          string   `json:"monitor_version,omitempty"`
	OutputSchemaName        string   `json:"output_schema_name"`
	ProfileMetricsTableName string   `json:"profile_metrics_table_name,omitempty"`
	SkipBuiltinDashboard    bool     `json:"skip_builtin_dashboard,omitempty"`
	SlicingExprs            []string `json:"slicing_exprs,omitempty"`
	Status                  string   `json:"status,omitempty"`
	TableName               string   `json:"table_name"`
	WarehouseId             string   `json:"warehouse_id,omitempty"`
}
