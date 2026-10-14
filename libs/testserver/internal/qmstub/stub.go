// Package qmstub is a temporary stand-in for libs/qualitymonitors.
package qmstub

type MonitorInfo struct {
	AssetsDir               string `json:"assets_dir,omitempty"`
	Status                  string `json:"status,omitempty"`
	TableName               string `json:"table_name,omitempty"`
	DriftMetricsTableName   string `json:"drift_metrics_table_name,omitempty"`
	ProfileMetricsTableName string `json:"profile_metrics_table_name,omitempty"`
	DashboardId             string `json:"dashboard_id,omitempty"`
}

type CreateMonitor struct {
	AssetsDir string `json:"assets_dir,omitempty"`
}
