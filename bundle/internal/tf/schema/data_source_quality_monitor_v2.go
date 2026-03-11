// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceQualityMonitorV2AnomalyDetectionConfig struct {
	ExcludedTableFullNames []string `json:"excluded_table_full_names,omitempty"`
	LastRunId              string   `json:"last_run_id,omitempty"`
	LatestRunStatus        string   `json:"latest_run_status,omitempty"`
}

type DataSourceQualityMonitorV2ProviderConfig struct {
	WorkspaceId string `json:"workspace_id"`
}

type DataSourceQualityMonitorV2ValidityCheckConfigurationsPercentNullValidityCheck struct {
	ColumnNames []string `json:"column_names,omitempty"`
	UpperBound  int      `json:"upper_bound,omitempty"`
}

type DataSourceQualityMonitorV2ValidityCheckConfigurationsRangeValidityCheck struct {
	ColumnNames []string `json:"column_names,omitempty"`
	LowerBound  int      `json:"lower_bound,omitempty"`
	UpperBound  int      `json:"upper_bound,omitempty"`
}

type DataSourceQualityMonitorV2ValidityCheckConfigurationsUniquenessValidityCheck struct {
	ColumnNames []string `json:"column_names,omitempty"`
}

type DataSourceQualityMonitorV2ValidityCheckConfigurations struct {
	Name                     string                                                                         `json:"name,omitempty"`
	PercentNullValidityCheck *DataSourceQualityMonitorV2ValidityCheckConfigurationsPercentNullValidityCheck `json:"percent_null_validity_check,omitempty"`
	RangeValidityCheck       *DataSourceQualityMonitorV2ValidityCheckConfigurationsRangeValidityCheck       `json:"range_validity_check,omitempty"`
	UniquenessValidityCheck  *DataSourceQualityMonitorV2ValidityCheckConfigurationsUniquenessValidityCheck  `json:"uniqueness_validity_check,omitempty"`
}

type DataSourceQualityMonitorV2 struct {
	AnomalyDetectionConfig      *DataSourceQualityMonitorV2AnomalyDetectionConfig       `json:"anomaly_detection_config,omitempty"`
	ObjectId                    string                                                  `json:"object_id"`
	ObjectType                  string                                                  `json:"object_type"`
	ProviderConfig              *DataSourceQualityMonitorV2ProviderConfig               `json:"provider_config,omitempty"`
	ValidityCheckConfigurations []DataSourceQualityMonitorV2ValidityCheckConfigurations `json:"validity_check_configurations,omitempty"`
}
