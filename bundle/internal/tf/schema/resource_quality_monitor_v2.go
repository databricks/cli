// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceQualityMonitorV2AnomalyDetectionConfig struct {
	LastRunId       string `json:"last_run_id,omitempty"`
	LatestRunStatus string `json:"latest_run_status,omitempty"`
}

type ResourceQualityMonitorV2 struct {
	AnomalyDetectionConfig *ResourceQualityMonitorV2AnomalyDetectionConfig `json:"anomaly_detection_config,omitempty"`
	ObjectId               string                                          `json:"object_id"`
	ObjectType             string                                          `json:"object_type"`
}
