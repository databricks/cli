// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceDataQualityMonitorAnomalyDetectionConfig struct {
}

type ResourceDataQualityMonitorDataProfilingConfigCustomMetrics struct {
	Definition     string   `json:"definition"`
	InputColumns   []string `json:"input_columns"`
	Name           string   `json:"name"`
	OutputDataType string   `json:"output_data_type"`
	Type           string   `json:"type"`
}

type ResourceDataQualityMonitorDataProfilingConfigInferenceLog struct {
	Granularities    []string `json:"granularities"`
	LabelColumn      string   `json:"label_column,omitempty"`
	ModelIdColumn    string   `json:"model_id_column"`
	PredictionColumn string   `json:"prediction_column"`
	ProblemType      string   `json:"problem_type"`
	TimestampColumn  string   `json:"timestamp_column"`
}

type ResourceDataQualityMonitorDataProfilingConfigNotificationSettingsOnFailure struct {
	EmailAddresses []string `json:"email_addresses,omitempty"`
}

type ResourceDataQualityMonitorDataProfilingConfigNotificationSettings struct {
	OnFailure *ResourceDataQualityMonitorDataProfilingConfigNotificationSettingsOnFailure `json:"on_failure,omitempty"`
}

type ResourceDataQualityMonitorDataProfilingConfigSchedule struct {
	PauseStatus          string `json:"pause_status,omitempty"`
	QuartzCronExpression string `json:"quartz_cron_expression"`
	TimezoneId           string `json:"timezone_id"`
}

type ResourceDataQualityMonitorDataProfilingConfigSnapshot struct {
}

type ResourceDataQualityMonitorDataProfilingConfigTimeSeries struct {
	Granularities   []string `json:"granularities"`
	TimestampColumn string   `json:"timestamp_column"`
}

type ResourceDataQualityMonitorDataProfilingConfig struct {
	AssetsDir                   string                                                             `json:"assets_dir,omitempty"`
	BaselineTableName           string                                                             `json:"baseline_table_name,omitempty"`
	CustomMetrics               []ResourceDataQualityMonitorDataProfilingConfigCustomMetrics       `json:"custom_metrics,omitempty"`
	DashboardId                 string                                                             `json:"dashboard_id,omitempty"`
	DriftMetricsTableName       string                                                             `json:"drift_metrics_table_name,omitempty"`
	EffectiveWarehouseId        string                                                             `json:"effective_warehouse_id,omitempty"`
	InferenceLog                *ResourceDataQualityMonitorDataProfilingConfigInferenceLog         `json:"inference_log,omitempty"`
	LatestMonitorFailureMessage string                                                             `json:"latest_monitor_failure_message,omitempty"`
	MonitorVersion              int                                                                `json:"monitor_version,omitempty"`
	MonitoredTableName          string                                                             `json:"monitored_table_name,omitempty"`
	NotificationSettings        *ResourceDataQualityMonitorDataProfilingConfigNotificationSettings `json:"notification_settings,omitempty"`
	OutputSchemaId              string                                                             `json:"output_schema_id"`
	ProfileMetricsTableName     string                                                             `json:"profile_metrics_table_name,omitempty"`
	Schedule                    *ResourceDataQualityMonitorDataProfilingConfigSchedule             `json:"schedule,omitempty"`
	SkipBuiltinDashboard        bool                                                               `json:"skip_builtin_dashboard,omitempty"`
	SlicingExprs                []string                                                           `json:"slicing_exprs,omitempty"`
	Snapshot                    *ResourceDataQualityMonitorDataProfilingConfigSnapshot             `json:"snapshot,omitempty"`
	Status                      string                                                             `json:"status,omitempty"`
	TimeSeries                  *ResourceDataQualityMonitorDataProfilingConfigTimeSeries           `json:"time_series,omitempty"`
	WarehouseId                 string                                                             `json:"warehouse_id,omitempty"`
}

type ResourceDataQualityMonitor struct {
	AnomalyDetectionConfig *ResourceDataQualityMonitorAnomalyDetectionConfig `json:"anomaly_detection_config,omitempty"`
	DataProfilingConfig    *ResourceDataQualityMonitorDataProfilingConfig    `json:"data_profiling_config,omitempty"`
	ObjectId               string                                            `json:"object_id"`
	ObjectType             string                                            `json:"object_type"`
}
