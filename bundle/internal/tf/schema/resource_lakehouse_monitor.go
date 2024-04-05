// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceLakehouseMonitorCustomMetrics struct {
	Definition     string   `json:"definition,omitempty"`
	InputColumns   []string `json:"input_columns,omitempty"`
	Name           string   `json:"name,omitempty"`
	OutputDataType string   `json:"output_data_type,omitempty"`
	Type           string   `json:"type,omitempty"`
}

type ResourceLakehouseMonitorDataClassificationConfig struct {
	Enabled bool `json:"enabled,omitempty"`
}

type ResourceLakehouseMonitorInferenceLog struct {
	Granularities      []string `json:"granularities,omitempty"`
	LabelCol           string   `json:"label_col,omitempty"`
	ModelIdCol         string   `json:"model_id_col,omitempty"`
	PredictionCol      string   `json:"prediction_col,omitempty"`
	PredictionProbaCol string   `json:"prediction_proba_col,omitempty"`
	ProblemType        string   `json:"problem_type,omitempty"`
	TimestampCol       string   `json:"timestamp_col,omitempty"`
}

type ResourceLakehouseMonitorNotificationsOnFailure struct {
	EmailAddresses []string `json:"email_addresses,omitempty"`
}

type ResourceLakehouseMonitorNotifications struct {
	OnFailure *ResourceLakehouseMonitorNotificationsOnFailure `json:"on_failure,omitempty"`
}

type ResourceLakehouseMonitorSchedule struct {
	PauseStatus          string `json:"pause_status,omitempty"`
	QuartzCronExpression string `json:"quartz_cron_expression,omitempty"`
	TimezoneId           string `json:"timezone_id,omitempty"`
}

type ResourceLakehouseMonitorSnapshot struct {
}

type ResourceLakehouseMonitorTimeSeries struct {
	Granularities []string `json:"granularities,omitempty"`
	TimestampCol  string   `json:"timestamp_col,omitempty"`
}

type ResourceLakehouseMonitor struct {
	AssetsDir                string                                            `json:"assets_dir"`
	BaselineTableName        string                                            `json:"baseline_table_name,omitempty"`
	DashboardId              string                                            `json:"dashboard_id,omitempty"`
	DriftMetricsTableName    string                                            `json:"drift_metrics_table_name,omitempty"`
	Id                       string                                            `json:"id,omitempty"`
	LatestMonitorFailureMsg  string                                            `json:"latest_monitor_failure_msg,omitempty"`
	MonitorVersion           string                                            `json:"monitor_version,omitempty"`
	OutputSchemaName         string                                            `json:"output_schema_name"`
	ProfileMetricsTableName  string                                            `json:"profile_metrics_table_name,omitempty"`
	SkipBuiltinDashboard     bool                                              `json:"skip_builtin_dashboard,omitempty"`
	SlicingExprs             []string                                          `json:"slicing_exprs,omitempty"`
	Status                   string                                            `json:"status,omitempty"`
	TableName                string                                            `json:"table_name"`
	WarehouseId              string                                            `json:"warehouse_id,omitempty"`
	CustomMetrics            []ResourceLakehouseMonitorCustomMetrics           `json:"custom_metrics,omitempty"`
	DataClassificationConfig *ResourceLakehouseMonitorDataClassificationConfig `json:"data_classification_config,omitempty"`
	InferenceLog             *ResourceLakehouseMonitorInferenceLog             `json:"inference_log,omitempty"`
	Notifications            *ResourceLakehouseMonitorNotifications            `json:"notifications,omitempty"`
	Schedule                 *ResourceLakehouseMonitorSchedule                 `json:"schedule,omitempty"`
	Snapshot                 *ResourceLakehouseMonitorSnapshot                 `json:"snapshot,omitempty"`
	TimeSeries               *ResourceLakehouseMonitorTimeSeries               `json:"time_series,omitempty"`
}
