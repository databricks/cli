// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceDataQualityMonitorsMonitorsAnomalyDetectionConfig struct {
}

type DataSourceDataQualityMonitorsMonitorsDataProfilingConfigCustomMetrics struct {
	Definition     string   `json:"definition"`
	InputColumns   []string `json:"input_columns"`
	Name           string   `json:"name"`
	OutputDataType string   `json:"output_data_type"`
	Type           string   `json:"type"`
}

type DataSourceDataQualityMonitorsMonitorsDataProfilingConfigInferenceLog struct {
	Granularities    []string `json:"granularities"`
	LabelColumn      string   `json:"label_column,omitempty"`
	ModelIdColumn    string   `json:"model_id_column"`
	PredictionColumn string   `json:"prediction_column"`
	ProblemType      string   `json:"problem_type"`
	TimestampColumn  string   `json:"timestamp_column"`
}

type DataSourceDataQualityMonitorsMonitorsDataProfilingConfigNotificationSettingsOnFailure struct {
	EmailAddresses []string `json:"email_addresses,omitempty"`
}

type DataSourceDataQualityMonitorsMonitorsDataProfilingConfigNotificationSettings struct {
	OnFailure *DataSourceDataQualityMonitorsMonitorsDataProfilingConfigNotificationSettingsOnFailure `json:"on_failure,omitempty"`
}

type DataSourceDataQualityMonitorsMonitorsDataProfilingConfigSchedule struct {
	PauseStatus          string `json:"pause_status,omitempty"`
	QuartzCronExpression string `json:"quartz_cron_expression"`
	TimezoneId           string `json:"timezone_id"`
}

type DataSourceDataQualityMonitorsMonitorsDataProfilingConfigSnapshot struct {
}

type DataSourceDataQualityMonitorsMonitorsDataProfilingConfigTimeSeries struct {
	Granularities   []string `json:"granularities"`
	TimestampColumn string   `json:"timestamp_column"`
}

type DataSourceDataQualityMonitorsMonitorsDataProfilingConfig struct {
	AssetsDir                   string                                                                        `json:"assets_dir,omitempty"`
	BaselineTableName           string                                                                        `json:"baseline_table_name,omitempty"`
	CustomMetrics               []DataSourceDataQualityMonitorsMonitorsDataProfilingConfigCustomMetrics       `json:"custom_metrics,omitempty"`
	DashboardId                 string                                                                        `json:"dashboard_id,omitempty"`
	DriftMetricsTableName       string                                                                        `json:"drift_metrics_table_name,omitempty"`
	EffectiveWarehouseId        string                                                                        `json:"effective_warehouse_id,omitempty"`
	InferenceLog                *DataSourceDataQualityMonitorsMonitorsDataProfilingConfigInferenceLog         `json:"inference_log,omitempty"`
	LatestMonitorFailureMessage string                                                                        `json:"latest_monitor_failure_message,omitempty"`
	MonitorVersion              int                                                                           `json:"monitor_version,omitempty"`
	MonitoredTableName          string                                                                        `json:"monitored_table_name,omitempty"`
	NotificationSettings        *DataSourceDataQualityMonitorsMonitorsDataProfilingConfigNotificationSettings `json:"notification_settings,omitempty"`
	OutputSchemaId              string                                                                        `json:"output_schema_id"`
	ProfileMetricsTableName     string                                                                        `json:"profile_metrics_table_name,omitempty"`
	Schedule                    *DataSourceDataQualityMonitorsMonitorsDataProfilingConfigSchedule             `json:"schedule,omitempty"`
	SkipBuiltinDashboard        bool                                                                          `json:"skip_builtin_dashboard,omitempty"`
	SlicingExprs                []string                                                                      `json:"slicing_exprs,omitempty"`
	Snapshot                    *DataSourceDataQualityMonitorsMonitorsDataProfilingConfigSnapshot             `json:"snapshot,omitempty"`
	Status                      string                                                                        `json:"status,omitempty"`
	TimeSeries                  *DataSourceDataQualityMonitorsMonitorsDataProfilingConfigTimeSeries           `json:"time_series,omitempty"`
	WarehouseId                 string                                                                        `json:"warehouse_id,omitempty"`
}

type DataSourceDataQualityMonitorsMonitors struct {
	AnomalyDetectionConfig *DataSourceDataQualityMonitorsMonitorsAnomalyDetectionConfig `json:"anomaly_detection_config,omitempty"`
	DataProfilingConfig    *DataSourceDataQualityMonitorsMonitorsDataProfilingConfig    `json:"data_profiling_config,omitempty"`
	ObjectId               string                                                       `json:"object_id"`
	ObjectType             string                                                       `json:"object_type"`
}

type DataSourceDataQualityMonitors struct {
	Monitors []DataSourceDataQualityMonitorsMonitors `json:"monitors,omitempty"`
	PageSize int                                     `json:"page_size,omitempty"`
}
