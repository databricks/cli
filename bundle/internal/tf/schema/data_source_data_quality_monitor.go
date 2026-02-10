// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceDataQualityMonitorAnomalyDetectionConfig struct {
	ExcludedTableFullNames []string `json:"excluded_table_full_names,omitempty"`
}

type DataSourceDataQualityMonitorDataProfilingConfigCustomMetrics struct {
	Definition     string   `json:"definition"`
	InputColumns   []string `json:"input_columns"`
	Name           string   `json:"name"`
	OutputDataType string   `json:"output_data_type"`
	Type           string   `json:"type"`
}

type DataSourceDataQualityMonitorDataProfilingConfigInferenceLog struct {
	Granularities    []string `json:"granularities"`
	LabelColumn      string   `json:"label_column,omitempty"`
	ModelIdColumn    string   `json:"model_id_column"`
	PredictionColumn string   `json:"prediction_column"`
	ProblemType      string   `json:"problem_type"`
	TimestampColumn  string   `json:"timestamp_column"`
}

type DataSourceDataQualityMonitorDataProfilingConfigNotificationSettingsOnFailure struct {
	EmailAddresses []string `json:"email_addresses,omitempty"`
}

type DataSourceDataQualityMonitorDataProfilingConfigNotificationSettings struct {
	OnFailure *DataSourceDataQualityMonitorDataProfilingConfigNotificationSettingsOnFailure `json:"on_failure,omitempty"`
}

type DataSourceDataQualityMonitorDataProfilingConfigSchedule struct {
	PauseStatus          string `json:"pause_status,omitempty"`
	QuartzCronExpression string `json:"quartz_cron_expression"`
	TimezoneId           string `json:"timezone_id"`
}

type DataSourceDataQualityMonitorDataProfilingConfigSnapshot struct {
}

type DataSourceDataQualityMonitorDataProfilingConfigTimeSeries struct {
	Granularities   []string `json:"granularities"`
	TimestampColumn string   `json:"timestamp_column"`
}

type DataSourceDataQualityMonitorDataProfilingConfig struct {
	AssetsDir                   string                                                               `json:"assets_dir,omitempty"`
	BaselineTableName           string                                                               `json:"baseline_table_name,omitempty"`
	CustomMetrics               []DataSourceDataQualityMonitorDataProfilingConfigCustomMetrics       `json:"custom_metrics,omitempty"`
	DashboardId                 string                                                               `json:"dashboard_id,omitempty"`
	DriftMetricsTableName       string                                                               `json:"drift_metrics_table_name,omitempty"`
	EffectiveWarehouseId        string                                                               `json:"effective_warehouse_id,omitempty"`
	InferenceLog                *DataSourceDataQualityMonitorDataProfilingConfigInferenceLog         `json:"inference_log,omitempty"`
	LatestMonitorFailureMessage string                                                               `json:"latest_monitor_failure_message,omitempty"`
	MonitorVersion              int                                                                  `json:"monitor_version,omitempty"`
	MonitoredTableName          string                                                               `json:"monitored_table_name,omitempty"`
	NotificationSettings        *DataSourceDataQualityMonitorDataProfilingConfigNotificationSettings `json:"notification_settings,omitempty"`
	OutputSchemaId              string                                                               `json:"output_schema_id"`
	ProfileMetricsTableName     string                                                               `json:"profile_metrics_table_name,omitempty"`
	Schedule                    *DataSourceDataQualityMonitorDataProfilingConfigSchedule             `json:"schedule,omitempty"`
	SkipBuiltinDashboard        bool                                                                 `json:"skip_builtin_dashboard,omitempty"`
	SlicingExprs                []string                                                             `json:"slicing_exprs,omitempty"`
	Snapshot                    *DataSourceDataQualityMonitorDataProfilingConfigSnapshot             `json:"snapshot,omitempty"`
	Status                      string                                                               `json:"status,omitempty"`
	TimeSeries                  *DataSourceDataQualityMonitorDataProfilingConfigTimeSeries           `json:"time_series,omitempty"`
	WarehouseId                 string                                                               `json:"warehouse_id,omitempty"`
}

type DataSourceDataQualityMonitor struct {
	AnomalyDetectionConfig *DataSourceDataQualityMonitorAnomalyDetectionConfig `json:"anomaly_detection_config,omitempty"`
	DataProfilingConfig    *DataSourceDataQualityMonitorDataProfilingConfig    `json:"data_profiling_config,omitempty"`
	ObjectId               string                                              `json:"object_id"`
	ObjectType             string                                              `json:"object_type"`
}
