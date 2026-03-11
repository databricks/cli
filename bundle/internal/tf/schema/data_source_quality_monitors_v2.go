// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceQualityMonitorsV2ProviderConfig struct {
	WorkspaceId string `json:"workspace_id"`
}

type DataSourceQualityMonitorsV2QualityMonitorsAnomalyDetectionConfig struct {
	ExcludedTableFullNames []string `json:"excluded_table_full_names,omitempty"`
	LastRunId              string   `json:"last_run_id,omitempty"`
	LatestRunStatus        string   `json:"latest_run_status,omitempty"`
}

type DataSourceQualityMonitorsV2QualityMonitorsProviderConfig struct {
	WorkspaceId string `json:"workspace_id"`
}

type DataSourceQualityMonitorsV2QualityMonitorsValidityCheckConfigurationsPercentNullValidityCheck struct {
	ColumnNames []string `json:"column_names,omitempty"`
	UpperBound  int      `json:"upper_bound,omitempty"`
}

type DataSourceQualityMonitorsV2QualityMonitorsValidityCheckConfigurationsRangeValidityCheck struct {
	ColumnNames []string `json:"column_names,omitempty"`
	LowerBound  int      `json:"lower_bound,omitempty"`
	UpperBound  int      `json:"upper_bound,omitempty"`
}

type DataSourceQualityMonitorsV2QualityMonitorsValidityCheckConfigurationsUniquenessValidityCheck struct {
	ColumnNames []string `json:"column_names,omitempty"`
}

type DataSourceQualityMonitorsV2QualityMonitorsValidityCheckConfigurations struct {
	Name                     string                                                                                         `json:"name,omitempty"`
	PercentNullValidityCheck *DataSourceQualityMonitorsV2QualityMonitorsValidityCheckConfigurationsPercentNullValidityCheck `json:"percent_null_validity_check,omitempty"`
	RangeValidityCheck       *DataSourceQualityMonitorsV2QualityMonitorsValidityCheckConfigurationsRangeValidityCheck       `json:"range_validity_check,omitempty"`
	UniquenessValidityCheck  *DataSourceQualityMonitorsV2QualityMonitorsValidityCheckConfigurationsUniquenessValidityCheck  `json:"uniqueness_validity_check,omitempty"`
}

type DataSourceQualityMonitorsV2QualityMonitors struct {
	AnomalyDetectionConfig      *DataSourceQualityMonitorsV2QualityMonitorsAnomalyDetectionConfig       `json:"anomaly_detection_config,omitempty"`
	ObjectId                    string                                                                  `json:"object_id"`
	ObjectType                  string                                                                  `json:"object_type"`
	ProviderConfig              *DataSourceQualityMonitorsV2QualityMonitorsProviderConfig               `json:"provider_config,omitempty"`
	ValidityCheckConfigurations []DataSourceQualityMonitorsV2QualityMonitorsValidityCheckConfigurations `json:"validity_check_configurations,omitempty"`
}

type DataSourceQualityMonitorsV2 struct {
	PageSize        int                                          `json:"page_size,omitempty"`
	ProviderConfig  *DataSourceQualityMonitorsV2ProviderConfig   `json:"provider_config,omitempty"`
	QualityMonitors []DataSourceQualityMonitorsV2QualityMonitors `json:"quality_monitors,omitempty"`
}
