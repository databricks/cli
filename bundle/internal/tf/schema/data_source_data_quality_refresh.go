// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceDataQualityRefresh struct {
	EndTimeMs   int    `json:"end_time_ms,omitempty"`
	Message     string `json:"message,omitempty"`
	ObjectId    string `json:"object_id"`
	ObjectType  string `json:"object_type"`
	RefreshId   int    `json:"refresh_id,omitempty"`
	StartTimeMs int    `json:"start_time_ms,omitempty"`
	State       string `json:"state,omitempty"`
	Trigger     string `json:"trigger,omitempty"`
}
