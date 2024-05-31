// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceQualityMonitorCustomMetrics struct {
	Definition     string   `json:"definition"`
	InputColumns   []string `json:"input_columns"`
	Name           string   `json:"name"`
	OutputDataType string   `json:"output_data_type"`
	Type           string   `json:"type"`
}

type ResourceQualityMonitorDataClassificationConfig struct {
	Enabled bool `json:"enabled,omitempty"`
}

type ResourceQualityMonitorInferenceLog struct {
	Granularities      []string `json:"granularities"`
	LabelCol           string   `json:"label_col,omitempty"`
	ModelIdCol         string   `json:"model_id_col"`
	PredictionCol      string   `json:"prediction_col"`
	PredictionProbaCol string   `json:"prediction_proba_col,omitempty"`
	ProblemType        string   `json:"problem_type"`
	TimestampCol       string   `json:"timestamp_col"`
}

type ResourceQualityMonitorNotificationsOnFailure struct {
	EmailAddresses []string `json:"email_addresses,omitempty"`
}

type ResourceQualityMonitorNotificationsOnNewClassificationTagDetected struct {
	EmailAddresses []string `json:"email_addresses,omitempty"`
}

type ResourceQualityMonitorNotifications struct {
	OnFailure                      *ResourceQualityMonitorNotificationsOnFailure                      `json:"on_failure,omitempty"`
	OnNewClassificationTagDetected *ResourceQualityMonitorNotificationsOnNewClassificationTagDetected `json:"on_new_classification_tag_detected,omitempty"`
}

type ResourceQualityMonitorSchedule struct {
	PauseStatus          string `json:"pause_status,omitempty"`
	QuartzCronExpression string `json:"quartz_cron_expression"`
	TimezoneId           string `json:"timezone_id"`
}

type ResourceQualityMonitorSnapshot struct {
}

type ResourceQualityMonitorTimeSeries struct {
	Granularities []string `json:"granularities"`
	TimestampCol  string   `json:"timestamp_col"`
}

type ResourceQualityMonitor struct {
	AssetsDir                string                                          `json:"assets_dir"`
	BaselineTableName        string                                          `json:"baseline_table_name,omitempty"`
	DashboardId              string                                          `json:"dashboard_id,omitempty"`
	DriftMetricsTableName    string                                          `json:"drift_metrics_table_name,omitempty"`
	Id                       string                                          `json:"id,omitempty"`
	LatestMonitorFailureMsg  string                                          `json:"latest_monitor_failure_msg,omitempty"`
	MonitorVersion           string                                          `json:"monitor_version,omitempty"`
	OutputSchemaName         string                                          `json:"output_schema_name"`
	ProfileMetricsTableName  string                                          `json:"profile_metrics_table_name,omitempty"`
	SkipBuiltinDashboard     bool                                            `json:"skip_builtin_dashboard,omitempty"`
	SlicingExprs             []string                                        `json:"slicing_exprs,omitempty"`
	Status                   string                                          `json:"status,omitempty"`
	TableName                string                                          `json:"table_name"`
	WarehouseId              string                                          `json:"warehouse_id,omitempty"`
	CustomMetrics            []ResourceQualityMonitorCustomMetrics           `json:"custom_metrics,omitempty"`
	DataClassificationConfig *ResourceQualityMonitorDataClassificationConfig `json:"data_classification_config,omitempty"`
	InferenceLog             *ResourceQualityMonitorInferenceLog             `json:"inference_log,omitempty"`
	Notifications            *ResourceQualityMonitorNotifications            `json:"notifications,omitempty"`
	Schedule                 *ResourceQualityMonitorSchedule                 `json:"schedule,omitempty"`
	Snapshot                 *ResourceQualityMonitorSnapshot                 `json:"snapshot,omitempty"`
	TimeSeries               *ResourceQualityMonitorTimeSeries               `json:"time_series,omitempty"`
}
