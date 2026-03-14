// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceQualityMonitorV2AnomalyDetectionConfig struct {
	ExcludedTableFullNames []string `json:"excluded_table_full_names,omitempty"`
	LastRunId              string   `json:"last_run_id,omitempty"`
	LatestRunStatus        string   `json:"latest_run_status,omitempty"`
}

type ResourceQualityMonitorV2ProviderConfig struct {
	WorkspaceId string `json:"workspace_id"`
}

type ResourceQualityMonitorV2ValidityCheckConfigurationsPercentNullValidityCheck struct {
	ColumnNames []string `json:"column_names,omitempty"`
	UpperBound  int      `json:"upper_bound,omitempty"`
}

type ResourceQualityMonitorV2ValidityCheckConfigurationsRangeValidityCheck struct {
	ColumnNames []string `json:"column_names,omitempty"`
	LowerBound  int      `json:"lower_bound,omitempty"`
	UpperBound  int      `json:"upper_bound,omitempty"`
}

type ResourceQualityMonitorV2ValidityCheckConfigurationsUniquenessValidityCheck struct {
	ColumnNames []string `json:"column_names,omitempty"`
}

type ResourceQualityMonitorV2ValidityCheckConfigurations struct {
	Name                     string                                                                       `json:"name,omitempty"`
	PercentNullValidityCheck *ResourceQualityMonitorV2ValidityCheckConfigurationsPercentNullValidityCheck `json:"percent_null_validity_check,omitempty"`
	RangeValidityCheck       *ResourceQualityMonitorV2ValidityCheckConfigurationsRangeValidityCheck       `json:"range_validity_check,omitempty"`
	UniquenessValidityCheck  *ResourceQualityMonitorV2ValidityCheckConfigurationsUniquenessValidityCheck  `json:"uniqueness_validity_check,omitempty"`
}

type ResourceQualityMonitorV2 struct {
	AnomalyDetectionConfig      *ResourceQualityMonitorV2AnomalyDetectionConfig       `json:"anomaly_detection_config,omitempty"`
	ObjectId                    string                                                `json:"object_id"`
	ObjectType                  string                                                `json:"object_type"`
	ProviderConfig              *ResourceQualityMonitorV2ProviderConfig               `json:"provider_config,omitempty"`
	ValidityCheckConfigurations []ResourceQualityMonitorV2ValidityCheckConfigurations `json:"validity_check_configurations,omitempty"`
}
