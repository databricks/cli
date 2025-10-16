// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceQualityMonitorsV2QualityMonitorsAnomalyDetectionConfig struct {
	LastRunId       string `json:"last_run_id,omitempty"`
	LatestRunStatus string `json:"latest_run_status,omitempty"`
}

type DataSourceQualityMonitorsV2QualityMonitors struct {
	AnomalyDetectionConfig *DataSourceQualityMonitorsV2QualityMonitorsAnomalyDetectionConfig `json:"anomaly_detection_config,omitempty"`
	ObjectId               string                                                            `json:"object_id"`
	ObjectType             string                                                            `json:"object_type"`
}

type DataSourceQualityMonitorsV2 struct {
	QualityMonitors []DataSourceQualityMonitorsV2QualityMonitors `json:"quality_monitors,omitempty"`
}
